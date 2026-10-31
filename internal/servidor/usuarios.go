package servidor

import (
	"net/http"
	"net/mail"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/liandrodourado/assistencia-tecnica/internal/seguranca"
)

var perfisValidos = map[string]bool{"admin": true, "balcao": true, "tecnico": true}

func validarNomeEmail(nome, email string) (string, string, string) {
	nome = strings.TrimSpace(nome)
	email = strings.ToLower(strings.TrimSpace(email))
	if nome == "" {
		return "", "", "Informe o nome."
	}
	if len([]rune(nome)) > 100 {
		return "", "", "O nome deve ter no máximo 100 caracteres."
	}
	if a, err := mail.ParseAddress(email); err != nil || a.Address != email || !strings.Contains(email[strings.Index(email, "@"):], ".") {
		return "", "", "E-mail inválido."
	}
	return nome, email, ""
}

type UsuarioLinha struct {
	ID        int64  `json:"id"`
	Nome      string `json:"nome"`
	Email     string `json:"email"`
	Perfil    string `json:"perfil"`
	Ativo     bool   `json:"ativo"`
	Bloqueado bool   `json:"bloqueado"`
	CriadoEm  string `json:"criado_em"`
}

const selectUsuario = `SELECT id, nome, email, perfil, ativo,
	(bloqueado_ate IS NOT NULL AND bloqueado_ate > datetime('now')), criado_em FROM usuarios`

type linhaScanner interface{ Scan(...any) error }

func lerUsuario(l linhaScanner) (UsuarioLinha, error) {
	var u UsuarioLinha
	err := l.Scan(&u.ID, &u.Nome, &u.Email, &u.Perfil, &u.Ativo, &u.Bloqueado, &u.CriadoEm)
	return u, err
}

func (s *Servidor) listarUsuarios(w http.ResponseWriter, r *http.Request) {
	if !usuarioDe(r).eAdmin() {
		semPermissao(w)
		return
	}
	linhas, err := s.db.Leitura.QueryContext(r.Context(), selectUsuario+` ORDER BY ativo DESC, nome`)
	if err != nil {
		erroInterno(w, err)
		return
	}
	defer linhas.Close()
	lista := []UsuarioLinha{}
	for linhas.Next() {
		u, err := lerUsuario(linhas)
		if err != nil {
			erroInterno(w, err)
			return
		}
		lista = append(lista, u)
	}
	responderJSON(w, http.StatusOK, lista)
}

type dadosNovoUsuario struct {
	Nome   string `json:"nome"`
	Email  string `json:"email"`
	Senha  string `json:"senha"`
	Perfil string `json:"perfil"`
}

func (s *Servidor) criarUsuario(w http.ResponseWriter, r *http.Request) {
	eu := usuarioDe(r)
	if !eu.eAdmin() {
		semPermissao(w)
		return
	}
	var d dadosNovoUsuario
	if !lerJSON(w, r, &d) {
		return
	}
	nome, email, msg := validarNomeEmail(d.Nome, d.Email)
	if msg != "" {
		erroValidacao(w, msg)
		return
	}
	if !perfisValidos[d.Perfil] {
		erroValidacao(w, "Perfil inválido: escolha Administrador, Balcão ou Técnico.")
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
	var id int64
	err = tx.QueryRowContext(ctx,
		`INSERT INTO usuarios (nome, email, senha_hash, perfil) VALUES (?, ?, ?, ?) RETURNING id`,
		nome, email, hash, d.Perfil).Scan(&id)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: usuarios.email") {
			responderErro(w, http.StatusConflict, "EMAIL_EM_USO", "Este e-mail já está em uso.")
			return
		}
		erroInterno(w, err)
		return
	}
	if err := auditar(ctx, tx, ptr(eu.ID), "usuario", id, "criar", nil,
		map[string]string{"nome": nome, "email": email, "perfil": d.Perfil}); err != nil {
		erroInterno(w, err)
		return
	}
	u, err := lerUsuario(tx.QueryRowContext(ctx, selectUsuario+` WHERE id = ?`, id))
	if err != nil {
		erroInterno(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		erroInterno(w, err)
		return
	}
	responderJSON(w, http.StatusCreated, u)
}

type dadosAlterarUsuario struct {
	Nome      *string `json:"nome"`
	Perfil    *string `json:"perfil"`
	Ativo     *bool   `json:"ativo"`
	NovaSenha *string `json:"nova_senha"`
}

func (s *Servidor) alterarUsuario(w http.ResponseWriter, r *http.Request) {
	eu := usuarioDe(r)
	if !eu.eAdmin() {
		semPermissao(w)
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		erroValidacao(w, "Identificador inválido no endereço.")
		return
	}
	var d dadosAlterarUsuario
	if !lerJSON(w, r, &d) {
		return
	}
	var novoHash string
	if d.NovaSenha != nil {
		if msg := seguranca.ProblemaSenhaNova(*d.NovaSenha); msg != "" {
			erroValidacao(w, msg)
			return
		}
		if novoHash, err = seguranca.HashSenha(*d.NovaSenha); err != nil {
			erroInterno(w, err)
			return
		}
	}

	ctx := r.Context()
	tx, err := s.db.Escrita.BeginTx(ctx, nil)
	if err != nil {
		erroInterno(w, err)
		return
	}
	defer tx.Rollback()
	atual, err := lerUsuario(tx.QueryRowContext(ctx, selectUsuario+` WHERE id = ?`, id))
	if err != nil {
		responderErro(w, http.StatusNotFound, "NAO_ENCONTRADO", "Usuário não encontrado.")
		return
	}

	novo := atual
	if d.Nome != nil {
		n := strings.TrimSpace(*d.Nome)
		if n == "" || len([]rune(n)) > 100 {
			erroValidacao(w, "Informe um nome com até 100 caracteres.")
			return
		}
		novo.Nome = n
	}
	if d.Perfil != nil {
		if !perfisValidos[*d.Perfil] {
			erroValidacao(w, "Perfil inválido: escolha Administrador, Balcão ou Técnico.")
			return
		}
		novo.Perfil = *d.Perfil
	}
	if d.Ativo != nil {
		novo.Ativo = *d.Ativo
	}

	// O sistema nunca fica sem um administrador ativo.
	if atual.Perfil == "admin" && atual.Ativo && (novo.Perfil != "admin" || !novo.Ativo) {
		var admins int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM usuarios WHERE perfil = 'admin' AND ativo = 1`).Scan(&admins); err != nil {
			erroInterno(w, err)
			return
		}
		if admins <= 1 {
			erroValidacao(w, "Não é possível remover ou desativar o único administrador ativo.")
			return
		}
	}

	if _, err := tx.ExecContext(ctx, `UPDATE usuarios SET nome = ?, perfil = ?, ativo = ? WHERE id = ?`,
		novo.Nome, novo.Perfil, novo.Ativo, id); err != nil {
		erroInterno(w, err)
		return
	}
	if novoHash != "" {
		if _, err := tx.ExecContext(ctx,
			`UPDATE usuarios SET senha_hash = ?, tentativas_falhas = 0, bloqueado_ate = NULL WHERE id = ?`, novoHash, id); err != nil {
			erroInterno(w, err)
			return
		}
	}
	// Desativar, trocar o perfil ou redefinir a senha encerra as sessões abertas.
	if novoHash != "" || !novo.Ativo || novo.Perfil != atual.Perfil {
		if _, err := tx.ExecContext(ctx, `DELETE FROM sessoes WHERE usuario_id = ?`, id); err != nil {
			erroInterno(w, err)
			return
		}
	}

	antes, depois := map[string]any{}, map[string]any{}
	if novo.Nome != atual.Nome {
		antes["nome"], depois["nome"] = atual.Nome, novo.Nome
	}
	if novo.Perfil != atual.Perfil {
		antes["perfil"], depois["perfil"] = atual.Perfil, novo.Perfil
	}
	if novo.Ativo != atual.Ativo {
		antes["ativo"], depois["ativo"] = atual.Ativo, novo.Ativo
	}
	if novoHash != "" {
		depois["senha_redefinida"] = true // nunca a senha em si
	}
	if len(depois) > 0 {
		if err := auditar(ctx, tx, ptr(eu.ID), "usuario", id, "alterar", antes, depois); err != nil {
			erroInterno(w, err)
			return
		}
	}
	resultado, err := lerUsuario(tx.QueryRowContext(ctx, selectUsuario+` WHERE id = ?`, id))
	if err != nil {
		erroInterno(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		erroInterno(w, err)
		return
	}
	responderJSON(w, http.StatusOK, resultado)
}
