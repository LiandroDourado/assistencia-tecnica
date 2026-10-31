package banco

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestAbrirMigraEReabre(t *testing.T) {
	ctx := context.Background()
	arq := filepath.Join(t.TempDir(), "teste.db")
	b, err := Abrir(ctx, arq)
	if err != nil {
		t.Fatal(err)
	}
	var modo string
	b.Leitura.QueryRow(`PRAGMA journal_mode`).Scan(&modo)
	if modo != "wal" {
		t.Fatalf("journal_mode = %q, esperado wal", modo)
	}
	var fk int
	b.Leitura.QueryRow(`PRAGMA foreign_keys`).Scan(&fk)
	if fk != 1 {
		t.Fatal("chaves estrangeiras desligadas")
	}
	b.Fechar()

	// Reabrir não reaplica nada.
	b, err = Abrir(ctx, arq)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Fechar()
	var n int
	b.Leitura.QueryRow(`SELECT COUNT(*) FROM migracoes_aplicadas`).Scan(&n)
	if n != 1 {
		t.Fatalf("migrações aplicadas = %d", n)
	}
	if Config(ctx, b.Leitura, "empresa.nome", "") != "Minha Assistência" {
		t.Fatal("configuração inicial ausente")
	}
}

func TestAuditoriaImutavel(t *testing.T) {
	b, err := Abrir(context.Background(), filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer b.Fechar()
	b.Escrita.Exec(`INSERT INTO auditoria (entidade, acao) VALUES ('x', 'y')`)
	for _, q := range []string{`UPDATE auditoria SET acao = 'z'`, `DELETE FROM auditoria`} {
		_, err := b.Escrita.Exec(q)
		if err == nil || !strings.Contains(err.Error(), "AUDITORIA_IMUTAVEL") {
			t.Fatalf("%s: esperado AUDITORIA_IMUTAVEL, veio %v", q, err)
		}
	}
}
