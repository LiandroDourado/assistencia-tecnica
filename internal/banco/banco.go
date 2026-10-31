// Package banco abre o SQLite e aplica as migrações.
//
// São duas conexões ao mesmo arquivo: uma de escrita, com uma única conexão
// (o SQLite só grava uma coisa por vez; assim ninguém recebe "banco ocupado"
// e duas pessoas nunca reservam a mesma última peça), e uma de leitura, com
// várias conexões em paralelo graças ao modo WAL.
package banco

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"net/url"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migracoes/*.sql
var migracoes embed.FS

type Banco struct {
	Escrita *sql.DB
	Leitura *sql.DB
}

func dsn(arquivo string) string {
	q := url.Values{}
	for _, p := range []string{"foreign_keys(1)", "journal_mode(WAL)", "busy_timeout(10000)", "synchronous(NORMAL)"} {
		q.Add("_pragma", p)
	}
	q.Set("_txlock", "immediate")
	return "file:" + arquivo + "?" + q.Encode()
}

// Abrir abre (ou cria) o banco no arquivo informado e aplica as migrações pendentes.
func Abrir(ctx context.Context, arquivo string) (*Banco, error) {
	escrita, err := sql.Open("sqlite", dsn(arquivo))
	if err != nil {
		return nil, err
	}
	escrita.SetMaxOpenConns(1)
	escrita.SetConnMaxIdleTime(0)

	if err := migrar(ctx, escrita); err != nil {
		escrita.Close()
		return nil, err
	}

	leitura, err := sql.Open("sqlite", dsn(arquivo))
	if err != nil {
		escrita.Close()
		return nil, err
	}
	leitura.SetMaxOpenConns(4)
	return &Banco{Escrita: escrita, Leitura: leitura}, nil
}

func (b *Banco) Fechar() {
	b.Leitura.Close()
	b.Escrita.Close()
}

// migrar aplica, em ordem, cada arquivo de migracoes/ ainda não registrado.
// Cada migração roda numa transação: ou entra inteira, ou não entra.
func migrar(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS migracoes_aplicadas (
		versao TEXT PRIMARY KEY, aplicada_em TEXT NOT NULL DEFAULT (datetime('now')))`); err != nil {
		return err
	}
	arquivos, err := fs.Glob(migracoes, "migracoes/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(arquivos)
	for _, caminho := range arquivos {
		versao := strings.TrimSuffix(strings.TrimPrefix(caminho, "migracoes/"), ".sql")
		var existe int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM migracoes_aplicadas WHERE versao = ?`, versao).Scan(&existe); err != nil {
			return err
		}
		if existe > 0 {
			continue
		}
		conteudo, err := migracoes.ReadFile(caminho)
		if err != nil {
			return err
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, string(conteudo)); err != nil {
			tx.Rollback()
			return fmt.Errorf("migração %s: %w", versao, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO migracoes_aplicadas (versao) VALUES (?)`, versao); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// Config lê uma configuração; devolve padrao se não existir.
func Config(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, chave, padrao string) string {
	var v string
	if err := q.QueryRowContext(ctx, `SELECT valor FROM configuracoes WHERE chave = ?`, chave).Scan(&v); err != nil {
		return padrao
	}
	return v
}
