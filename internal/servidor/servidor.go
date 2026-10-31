// Package servidor contém as rotas HTTP da API e a entrega da interface.
package servidor

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/liandrodourado/assistencia-tecnica/internal/banco"
)

type Servidor struct {
	db           *banco.Banco
	cookieSeguro bool
	versao       string
	app          fs.FS
}

func Novo(db *banco.Banco, app fs.FS, versao string, cookieSeguro bool) *Servidor {
	return &Servidor{db: db, app: app, versao: versao, cookieSeguro: cookieSeguro}
}

func (s *Servidor) Rotas() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(cabecalhosSeguranca)
	r.Use(registrarRequisicao)

	r.Route("/api", func(r chi.Router) {
		r.Get("/saude", s.saude)
		r.Get("/estado", s.estado)
		r.Post("/primeiro-acesso", s.primeiroAcesso)
		r.Post("/auth/entrar", s.entrar)
		r.Post("/auth/sair", s.sair)

		r.Group(func(r chi.Router) {
			r.Use(s.exigirLogin)
			r.Get("/auth/eu", s.eu)
			r.Put("/auth/senha", s.trocarPropriaSenha)
			r.Get("/usuarios", s.listarUsuarios)
			r.Post("/usuarios", s.criarUsuario)
			r.Put("/usuarios/{id}", s.alterarUsuario)
		})

		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			responderErro(w, http.StatusNotFound, "NAO_ENCONTRADO", "Endereço da API não encontrado.")
		})
		r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
			responderErro(w, http.StatusMethodNotAllowed, "METODO_NAO_PERMITIDO", "Método não permitido neste endereço.")
		})
	})

	r.NotFound(s.interface_)
	return r
}

func cabecalhosSeguranca(prox http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("Content-Security-Policy", "frame-ancestors 'none'; object-src 'none'; base-uri 'self'")
		prox.ServeHTTP(w, r)
	})
}

func registrarRequisicao(prox http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inicio := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		prox.ServeHTTP(ww, r)
		if r.URL.Path != "/api/saude" {
			slog.Info("requisição", "metodo", r.Method, "caminho", r.URL.Path, "status", ww.Status(),
				"duracao_ms", time.Since(inicio).Milliseconds())
		}
	})
}

func (s *Servidor) saude(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if err := s.db.Leitura.PingContext(ctx); err != nil {
		responderErro(w, http.StatusServiceUnavailable, "BANCO_INDISPONIVEL", "Banco de dados indisponível.")
		return
	}
	responderJSON(w, http.StatusOK, map[string]string{"status": "ok", "versao": s.versao})
}

// LimparSessoes apaga sessões vencidas. Chamado periodicamente pelo main.
func (s *Servidor) LimparSessoes(ctx context.Context) {
	if _, err := s.db.Escrita.ExecContext(ctx, `DELETE FROM sessoes WHERE expira_em <= datetime('now')`); err != nil {
		slog.Warn("limpeza de sessões falhou", "erro", err)
	}
}
