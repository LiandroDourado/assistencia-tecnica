package servidor

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/liandrodourado/assistencia-tecnica/internal/banco"
	"github.com/liandrodourado/assistencia-tecnica/internal/seguranca"
)

const nomeCookie = "sessao"

type Usuario struct {
	ID     int64  `json:"id"`
	Nome   string `json:"nome"`
	Email  string `json:"email"`
	Perfil string `json:"perfil"`
}

type chaveUsuario struct{}

func usuarioDe(r *http.Request) Usuario {
	return r.Context().Value(chaveUsuario{}).(Usuario)
}

func (u Usuario) eAdmin() bool { return u.Perfil == "admin" }

// exigirLogin libera a rota só para quem tem sessão válida de usuário ativo.
func (s *Servidor) exigirLogin(prox http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(nomeCookie)
		if err != nil || c.Value == "" {
			naoAutenticado(w)
			return
		}
		var u Usuario
		err = s.db.Leitura.QueryRowContext(r.Context(),
			`SELECT u.id, u.nome, u.email, u.perfil FROM sessoes s JOIN usuarios u ON u.id = s.usuario_id
			  WHERE s.token_hash = ? AND s.expira_em > datetime('now') AND u.ativo = 1`,
			seguranca.HashToken(c.Value)).Scan(&u.ID, &u.Nome, &u.Email, &u.Perfil)
		if err != nil {
			naoAutenticado(w)
			return
		}
		prox.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), chaveUsuario{}, u)))
	})
}

func naoAutenticado(w http.ResponseWriter) {
	responderErro(w, http.StatusUnauthorized, "NAO_AUTENTICADO", "Sessão expirada ou inexistente. Entre novamente.")
}

// estado diz à interface se falta criar o primeiro administrador.
func (s *Servidor) estado(w http.ResponseWriter, r *http.Request) {
	var total int
	if err := s.db.Leitura.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM usuarios`).Scan(&total); err != nil {
		erroInterno(w, err)
		return
	}
	responderJSON(w, http.StatusOK, map[string]any{
		"configurado": total > 0,
		"nome_loja":   banco.Config(r.Context(), s.db.Leitura, "empresa.nome", "Assistência Técnica"),
		"versao":      s.versao,
	})
}

type dadosPrimeiroAcesso struct {
	NomeLoja string `json:"nome_loja"`
	Nome     string `json:"nome"`
	Email    string `json:"email"`
	Senha    string `json:"senha"`
}

// primeiroAcesso cria o administrador inicial. Só funciona com o banco vazio,
// e a verificação acontece dentro da transação de escrita: dois navegadores
// tentando ao mesmo tempo nunca criam dois administradores.
func (s *Servidor) primeiroAcesso(w http.ResponseWriter, r *http.Request) {
	var d dadosPrimeiroAcesso
	if !lerJSON(w, r, &d) {
		return
	}
	nome, email, msg := validarNomeEmail(d.Nome, d.Email)
	if msg != "" {
		erroValidacao(w, msg)
		return
	}
	if msg := seguranca.ProblemaSenhaNova(d.Senha); msg != "" {
		erroValidacao(w, msg)
		return
	}
	hash, err := seguranca.HashSenha(d.Senha)
	if err != nil {
		erroInterno(w, err)
		return
	}

	ctx := r.Context()
	tx, err := s.db.Escrita.BeginTx(ctx, nil)
	if err != nil {
		erroInterno(w, err)
		return
	}
	defer tx.Rollback()
	var total int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM usuarios`).Scan(&total); err != nil {
		erroInterno(w, err)
		return
	}
	if total > 0 {
		responderErro(w, http.StatusConflict, "JA_CONFIGURADO", "O sistema já foi configurado. Entre com seu usuário.")
		return
	}
	var id int64
	if err := tx.QueryRowContext(ctx,
		`INSERT INTO usuarios (nome, email, senha_hash, perfil) VALUES (?, ?, ?, 'admin') RETURNING id`,
		nome, email, hash).Scan(&id); err != nil {
		erroInterno(w, err)
		return
	}
	if loja := strings.TrimSpace(d.NomeLoja); loja != "" {
		if _, err := tx.ExecContext(ctx, `UPDATE configuracoes SET valor = ? WHERE chave = 'empresa.nome'`, loja); err != nil {
			erroInterno(w, err)
			return
		}
	}
	if err := auditar(ctx, tx, ptr(id), "usuario", id, "primeiro_acesso", nil,
		map[string]string{"nome": nome, "email": email, "perfil": "admin"}); err != nil {
		erroInterno(w, err)
		return
	}
	cookie, err := s.abrirSessao(ctx, tx, id)
	if err != nil {
		erroInterno(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		erroInterno(w, err)
		return
	}
	http.SetCookie(w, cookie)
	responderJSON(w, http.StatusCreated, Usuario{ID: id, Nome: nome, Email: email, Perfil: "admin"})
}

// abrirSessao grava a sessão (só o hash do token) e devolve o cookie.
func (s *Servidor) abrirSessao(ctx context.Context, tx *sql.Tx, usuarioID int64) (*http.Cookie, error) {
	horas, _ := strconv.Atoi(banco.Config(ctx, tx, "seguranca.sessao_horas", "12"))
	if horas <= 0 {
		horas = 12
	}
	token, err := seguranca.NovoToken()
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM sessoes WHERE expira_em <= datetime('now')`); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO sessoes (token_hash, usuario_id, expira_em) VALUES (?, ?, datetime('now', ?))`,
		seguranca.HashToken(token), usuarioID, "+"+strconv.Itoa(horas)+" hours"); err != nil {
		return nil, err
	}
	return &http.Cookie{
		Name: nomeCookie, Value: token, Path: "/", MaxAge: horas * 3600,
		HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: s.cookieSeguro,
	}, nil
}

type dadosEntrar struct {
	Email string `json:"email"`
	Senha string `json:"senha"`
}

func (s *Servidor) entrar(w http.ResponseWriter, r *http.Request) {
	var d dadosEntrar
	if !lerJSON(w, r, &d) {
		return
	}
	ctx := r.Context()
	email := strings.ToLower(strings.TrimSpace(d.Email))

	var (
		u         Usuario
		hash      string
		bloqueado bool
	)
	err := s.db.Leitura.QueryRowContext(ctx,
		`SELECT id, nome, email, perfil, senha_hash, (bloqueado_ate IS NOT NULL AND bloqueado_ate > datetime('now'))
		   FROM usuarios WHERE email = ? AND ativo = 1`, email).
		Scan(&u.ID, &u.Nome, &u.Email, &u.Perfil, &hash, &bloqueado)
	if errors.Is(err, sql.ErrNoRows) {
		seguranca.ConsumirTempo(d.Senha)
		credenciaisInvalidas(w)
		return
	}
	if err != nil {
		erroInterno(w, err)
		return
	}
	if bloqueado {
		responderErro(w, http.StatusTooManyRequests, "BLOQUEADO",
			"Acesso bloqueado temporariamente por excesso de tentativas. Tente novamente em alguns minutos.")
		return
	}

	// A verificação (pesada) acontece fora da transação de escrita.
	ok := seguranca.SenhaConfere(d.Senha, hash)

	tx, err := s.db.Escrita.BeginTx(ctx, nil)
	if err != nil {
		erroInterno(w, err)
		return
	}
	defer tx.Rollback()

	if !ok {
		max, _ := strconv.Atoi(banco.Config(ctx, tx, "seguranca.max_tentativas", "5"))
		minutos, _ := strconv.Atoi(banco.Config(ctx, tx, "seguranca.bloqueio_minutos", "15"))
		var tentativas int
		if err := tx.QueryRowContext(ctx,
			`UPDATE usuarios SET tentativas_falhas = tentativas_falhas + 1 WHERE id = ? RETURNING tentativas_falhas`,
			u.ID).Scan(&tentativas); err != nil {
			erroInterno(w, err)
			return
		}
		if tentativas >= max {
			if _, err := tx.ExecContext(ctx,
				`UPDATE usuarios SET tentativas_falhas = 0, bloqueado_ate = datetime('now', ?) WHERE id = ?`,
				"+"+strconv.Itoa(minutos)+" minutes", u.ID); err != nil {
				erroInterno(w, err)
				return
			}
			if err := auditar(ctx, tx, ptr(u.ID), "usuario", u.ID, "login_bloqueado", nil, map[string]int{"minutos": minutos}); err != nil {
				erroInterno(w, err)
				return
			}
		}
		if err := tx.Commit(); err != nil {
			erroInterno(w, err)
			return
		}
		credenciaisInvalidas(w)
		return
	}

	if _, err := tx.ExecContext(ctx, `UPDATE usuarios SET tentativas_falhas = 0, bloqueado_ate = NULL WHERE id = ?`, u.ID); err != nil {
		erroInterno(w, err)
		return
	}
	if err := auditar(ctx, tx, ptr(u.ID), "usuario", u.ID, "login", nil, nil); err != nil {
		erroInterno(w, err)
		return
	}
	cookie, err := s.abrirSessao(ctx, tx, u.ID)
	if err != nil {
		erroInterno(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		erroInterno(w, err)
		return
	}
	http.SetCookie(w, cookie)
	responderJSON(w, http.StatusOK, u)
}

func credenciaisInvalidas(w http.ResponseWriter) {
	responderErro(w, http.StatusUnauthorized, "CREDENCIAIS_INVALIDAS", "E-mail ou senha incorretos.")
}

func (s *Servidor) sair(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(nomeCookie); err == nil {
		s.db.Escrita.ExecContext(r.Context(), `DELETE FROM sessoes WHERE token_hash = ?`, seguranca.HashToken(c.Value))
	}
	http.SetCookie(w, &http.Cookie{Name: nomeCookie, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: s.cookieSeguro})
	responderJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Servidor) eu(w http.ResponseWriter, r *http.Request) {
	responderJSON(w, http.StatusOK, usuarioDe(r))
}

type dadosTrocaSenha struct {
	SenhaAtual string `json:"senha_atual"`
	NovaSenha  string `json:"nova_senha"`
}

// trocarPropriaSenha: qualquer usuário, informando a senha atual. As outras
// sessões dele são encerradas; a atual continua aberta.
func (s *Servidor) trocarPropriaSenha(w http.ResponseWriter, r *http.Request) {
	u := usuarioDe(r)
	var d dadosTrocaSenha
	if !lerJSON(w, r, &d) {
		return
	}
	if msg := seguranca.ProblemaSenhaNova(d.NovaSenha); msg != "" {
		erroValidacao(w, msg)
		return
	}
	ctx := r.Context()
	var hashAtual string
	if err := s.db.Leitura.QueryRowContext(ctx, `SELECT senha_hash FROM usuarios WHERE id = ?`, u.ID).Scan(&hashAtual); err != nil {
		erroInterno(w, err)
		return
	}
	if !seguranca.SenhaConfere(d.SenhaAtual, hashAtual) {
		erroValidacao(w, "A senha atual não confere.")
		return
	}
	novo, err := seguranca.HashSenha(d.NovaSenha)
	if err != nil {
		erroInterno(w, err)
		return
	}
	c, _ := r.Cookie(nomeCookie)
	tx, err := s.db.Escrita.BeginTx(ctx, nil)
	if err != nil {
		erroInterno(w, err)
		return
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE usuarios SET senha_hash = ? WHERE id = ?`, novo, u.ID); err != nil {
		erroInterno(w, err)
		return
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM sessoes WHERE usuario_id = ? AND token_hash <> ?`,
		u.ID, seguranca.HashToken(c.Value)); err != nil {
		erroInterno(w, err)
		return
	}
	if err := auditar(ctx, tx, ptr(u.ID), "usuario", u.ID, "trocar_senha", nil, nil); err != nil {
		erroInterno(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		erroInterno(w, err)
		return
	}
	responderJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
