package servidor

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/liandrodourado/assistencia-tecnica/internal/banco"
	"github.com/liandrodourado/assistencia-tecnica/internal/seguranca"
)

const senha = "senha-forte-123"

type teste struct {
	t  *testing.T
	db *banco.Banco
	h  http.Handler
}

func novo(t *testing.T) *teste {
	t.Helper()
	db, err := banco.Abrir(context.Background(), filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Fechar)
	app := fstest.MapFS{
		"index.html":            {Data: []byte("<!doctype html><title>SPA</title>")},
		"_app/immutable/app.js": {Data: []byte("console.log(1)")},
	}
	return &teste{t: t, db: db, h: Novo(db, app, "teste", false).Rotas()}
}

type resposta struct {
	status int
	corpo  map[string]any
	lista  []map[string]any
	cookie *http.Cookie
	header http.Header
	texto  string
}

func (x *teste) req(metodo, caminho, cookie string, corpo any) resposta {
	x.t.Helper()
	var b *bytes.Reader
	if corpo != nil {
		j, _ := json.Marshal(corpo)
		b = bytes.NewReader(j)
	} else {
		b = bytes.NewReader(nil)
	}
	r := httptest.NewRequest(metodo, caminho, b)
	if corpo != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	if cookie != "" {
		r.AddCookie(&http.Cookie{Name: nomeCookie, Value: cookie})
	}
	w := httptest.NewRecorder()
	x.h.ServeHTTP(w, r)
	res := resposta{status: w.Code, header: w.Header(), texto: w.Body.String()}
	json.Unmarshal(w.Body.Bytes(), &res.corpo)
	json.Unmarshal(w.Body.Bytes(), &res.lista)
	for _, c := range w.Result().Cookies() {
		if c.Name == nomeCookie {
			res.cookie = c
		}
	}
	return res
}

// configurar faz o primeiro acesso e devolve o cookie do administrador.
func (x *teste) configurar() string {
	r := x.req("POST", "/api/primeiro-acesso", "", map[string]string{
		"nome_loja": "Conserta Já", "nome": "Liandro", "email": "admin@loja.com", "senha": senha})
	if r.status != http.StatusCreated {
		x.t.Fatalf("primeiro acesso: %d %s", r.status, r.texto)
	}
	return r.cookie.Value
}

func (x *teste) criar(admin, email, perfil string) {
	r := x.req("POST", "/api/usuarios", admin, map[string]string{"nome": "Pessoa " + perfil, "email": email, "senha": senha, "perfil": perfil})
	if r.status != http.StatusCreated {
		x.t.Fatalf("criar %s: %d %s", email, r.status, r.texto)
	}
}

func (x *teste) entrar(email string) string {
	r := x.req("POST", "/api/auth/entrar", "", map[string]string{"email": email, "senha": senha})
	if r.status != http.StatusOK {
		x.t.Fatalf("entrar %s: %d %s", email, r.status, r.texto)
	}
	return r.cookie.Value
}

func (x *teste) contar(sql string, args ...any) int {
	var n int
	if err := x.db.Leitura.QueryRow(sql, args...).Scan(&n); err != nil {
		x.t.Fatal(err)
	}
	return n
}

func igual[T comparable](t *testing.T, nome string, veio, esperado T) {
	t.Helper()
	if veio != esperado {
		t.Fatalf("%s: veio %v, esperado %v", nome, veio, esperado)
	}
}

// ---------------------------------------------------------------------------

func TestSaudeEInterface(t *testing.T) {
	x := novo(t)
	r := x.req("GET", "/api/saude", "", nil)
	igual(t, "status", r.status, 200)
	igual(t, "saude", r.corpo["status"], any("ok"))

	// Qualquer caminho da interface devolve o index.html (roteamento no navegador).
	for _, c := range []string{"/", "/entrar", "/usuarios", "/os/123"} {
		r = x.req("GET", c, "", nil)
		igual(t, c, r.status, 200)
		if !strings.Contains(r.texto, "<title>SPA</title>") {
			t.Fatalf("%s não devolveu o index.html", c)
		}
	}
	r = x.req("GET", "/_app/immutable/app.js", "", nil)
	igual(t, "arquivo", r.texto, "console.log(1)")
	igual(t, "cache", r.header.Get("Cache-Control"), "public, max-age=31536000, immutable")

	// Endereço inexistente da API responde JSON em português, não o index.html.
	r = x.req("GET", "/api/nao-existe", "", nil)
	igual(t, "status", r.status, 404)
	igual(t, "erro", r.corpo["erro"], any("Endereço da API não encontrado."))
	igual(t, "nosniff", r.header.Get("X-Content-Type-Options"), "nosniff")
}

func TestSemInterfaceCompilada(t *testing.T) {
	db, _ := banco.Abrir(context.Background(), filepath.Join(t.TempDir(), "t.db"))
	defer db.Fechar()
	h := Novo(db, fstest.MapFS{}, "teste", false).Rotas()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "O servidor está funcionando") {
		t.Fatalf("página de aviso ausente: %d %s", w.Code, w.Body.String())
	}
}

func TestPrimeiroAcesso(t *testing.T) {
	x := novo(t)
	r := x.req("GET", "/api/estado", "", nil)
	igual(t, "configurado", r.corpo["configurado"], any(false))

	r = x.req("POST", "/api/primeiro-acesso", "", map[string]string{"nome": "A", "email": "a@b.com", "senha": "curta"})
	igual(t, "senha curta", r.corpo["erro"], any("A senha deve ter pelo menos 8 caracteres."))
	r = x.req("POST", "/api/primeiro-acesso", "", map[string]string{"nome": "A", "email": "sem-arroba", "senha": senha})
	igual(t, "email", r.corpo["erro"], any("E-mail inválido."))

	cookie := x.configurar()
	r = x.req("GET", "/api/auth/eu", cookie, nil)
	igual(t, "perfil", r.corpo["perfil"], any("admin"))
	r = x.req("GET", "/api/estado", "", nil)
	igual(t, "configurado", r.corpo["configurado"], any(true))
	igual(t, "loja", r.corpo["nome_loja"], any("Conserta Já"))

	// Depois de configurado, ninguém mais cria administrador por aqui.
	r = x.req("POST", "/api/primeiro-acesso", "", map[string]string{"nome": "Intruso", "email": "x@y.com", "senha": senha})
	igual(t, "status", r.status, http.StatusConflict)
	igual(t, "usuarios", x.contar(`SELECT COUNT(*) FROM usuarios`), 1)
}

func TestPrimeiroAcessoSimultaneoCriaUmSoAdmin(t *testing.T) {
	x := novo(t)
	var wg sync.WaitGroup
	status := make(chan int, 8)
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := x.req("POST", "/api/primeiro-acesso", "", map[string]string{
				"nome": "Admin", "email": "a" + string(rune('a'+i)) + "@loja.com", "senha": senha})
			status <- r.status
		}()
	}
	wg.Wait()
	close(status)
	criados := 0
	for s := range status {
		if s == http.StatusCreated {
			criados++
		}
	}
	igual(t, "criados", criados, 1)
	igual(t, "usuarios", x.contar(`SELECT COUNT(*) FROM usuarios`), 1)
}

func TestCookieSeguroESessaoComHash(t *testing.T) {
	x := novo(t)
	x.configurar()
	r := x.req("POST", "/api/auth/entrar", "", map[string]string{"email": "ADMIN@loja.com", "senha": senha})
	igual(t, "status", r.status, 200)
	c := r.cookie
	if !c.HttpOnly || c.SameSite != http.SameSiteStrictMode || c.Path != "/" {
		t.Fatalf("cookie inseguro: %+v", c)
	}
	igual(t, "hash no banco", x.contar(`SELECT COUNT(*) FROM sessoes WHERE token_hash = ?`, seguranca.HashToken(c.Value)), 1)
	igual(t, "token puro no banco", x.contar(`SELECT COUNT(*) FROM sessoes WHERE token_hash = ?`, c.Value), 0)
}

func TestExigeLogin(t *testing.T) {
	x := novo(t)
	for _, c := range []string{"", "token-inventado"} {
		r := x.req("GET", "/api/usuarios", c, nil)
		igual(t, "status", r.status, 401)
		igual(t, "erro", r.corpo["erro"], any("Sessão expirada ou inexistente. Entre novamente."))
	}
}

func TestBloqueioApos5Tentativas(t *testing.T) {
	x := novo(t)
	x.configurar()
	for range 5 {
		r := x.req("POST", "/api/auth/entrar", "", map[string]string{"email": "admin@loja.com", "senha": "errada"})
		igual(t, "status", r.status, 401)
		igual(t, "erro", r.corpo["erro"], any("E-mail ou senha incorretos."))
	}
	r := x.req("POST", "/api/auth/entrar", "", map[string]string{"email": "admin@loja.com", "senha": senha})
	igual(t, "bloqueado", r.status, http.StatusTooManyRequests)
	igual(t, "auditoria", x.contar(`SELECT COUNT(*) FROM auditoria WHERE acao = 'login_bloqueado'`), 1)

	// E-mail inexistente: mesma resposta de senha errada.
	r = x.req("POST", "/api/auth/entrar", "", map[string]string{"email": "ninguem@loja.com", "senha": "x"})
	igual(t, "inexistente", r.corpo["erro"], any("E-mail ou senha incorretos."))
}

func TestSairEncerraSessao(t *testing.T) {
	x := novo(t)
	c := x.configurar()
	x.req("POST", "/api/auth/sair", c, nil)
	igual(t, "depois de sair", x.req("GET", "/api/auth/eu", c, nil).status, 401)
}

func TestPermissoesDeUsuarios(t *testing.T) {
	x := novo(t)
	admin := x.configurar()
	x.criar(admin, "balcao@loja.com", "balcao")
	x.criar(admin, "tecnico@loja.com", "tecnico")
	for _, email := range []string{"balcao@loja.com", "tecnico@loja.com"} {
		c := x.entrar(email)
		igual(t, email+" lista", x.req("GET", "/api/usuarios", c, nil).status, 403)
		igual(t, email+" cria", x.req("POST", "/api/usuarios", c, map[string]string{"nome": "X", "email": "x@x.com", "senha": senha, "perfil": "admin"}).status, 403)
	}
	r := x.req("GET", "/api/usuarios", admin, nil)
	igual(t, "lista", len(r.lista), 3)

	r = x.req("POST", "/api/usuarios", admin, map[string]string{"nome": "Dup", "email": "BALCAO@loja.com", "senha": senha, "perfil": "balcao"})
	igual(t, "duplicado", r.corpo["erro"], any("Este e-mail já está em uso."))
	r = x.req("POST", "/api/usuarios", admin, map[string]string{"nome": "P", "email": "p@loja.com", "senha": senha, "perfil": "gerente"})
	igual(t, "perfil", r.status, 422)
}

func TestUnicoAdminNaoPodeSerRemovido(t *testing.T) {
	x := novo(t)
	admin := x.configurar()
	id := "1"
	r := x.req("PUT", "/api/usuarios/"+id, admin, map[string]any{"ativo": false})
	igual(t, "desativar", r.corpo["erro"], any("Não é possível remover ou desativar o único administrador ativo."))
	r = x.req("PUT", "/api/usuarios/"+id, admin, map[string]any{"perfil": "balcao"})
	igual(t, "rebaixar", r.status, 422)

	// Com um segundo administrador, pode.
	x.criar(admin, "admin2@loja.com", "admin")
	r = x.req("PUT", "/api/usuarios/2", admin, map[string]any{"perfil": "balcao"})
	igual(t, "rebaixar segundo", r.status, 200)
}

func TestRedefinirSenhaDerrubaSessoesENaoVaiParaAuditoria(t *testing.T) {
	x := novo(t)
	admin := x.configurar()
	x.criar(admin, "balcao@loja.com", "balcao")
	balcao := x.entrar("balcao@loja.com")
	r := x.req("PUT", "/api/usuarios/2", admin, map[string]any{"nova_senha": "outra-senha-456"})
	igual(t, "status", r.status, 200)
	igual(t, "sessão antiga", x.req("GET", "/api/auth/eu", balcao, nil).status, 401)

	var log string
	x.db.Leitura.QueryRow(`SELECT depois FROM auditoria WHERE acao = 'alterar'`).Scan(&log)
	if strings.Contains(log, "outra-senha") || !strings.Contains(log, "senha_redefinida") {
		t.Fatalf("auditoria inesperada: %s", log)
	}
	r = x.req("POST", "/api/auth/entrar", "", map[string]string{"email": "balcao@loja.com", "senha": "outra-senha-456"})
	igual(t, "nova senha", r.status, 200)
}

func TestDesativarImpedeLogin(t *testing.T) {
	x := novo(t)
	admin := x.configurar()
	x.criar(admin, "tecnico@loja.com", "tecnico")
	tec := x.entrar("tecnico@loja.com")
	x.req("PUT", "/api/usuarios/2", admin, map[string]any{"ativo": false})
	igual(t, "sessão", x.req("GET", "/api/auth/eu", tec, nil).status, 401)
	r := x.req("POST", "/api/auth/entrar", "", map[string]string{"email": "tecnico@loja.com", "senha": senha})
	igual(t, "login", r.status, 401)
}

func TestTrocarPropriaSenha(t *testing.T) {
	x := novo(t)
	admin := x.configurar()
	x.criar(admin, "tecnico@loja.com", "tecnico")
	c1 := x.entrar("tecnico@loja.com")
	c2 := x.entrar("tecnico@loja.com") // outro aparelho
	r := x.req("PUT", "/api/auth/senha", c1, map[string]string{"senha_atual": "errada", "nova_senha": "nova-senha-789"})
	igual(t, "senha atual", r.corpo["erro"], any("A senha atual não confere."))
	r = x.req("PUT", "/api/auth/senha", c1, map[string]string{"senha_atual": senha, "nova_senha": "nova-senha-789"})
	igual(t, "troca", r.status, 200)
	igual(t, "sessão atual continua", x.req("GET", "/api/auth/eu", c1, nil).status, 200)
	igual(t, "outra sessão cai", x.req("GET", "/api/auth/eu", c2, nil).status, 401)
}

func TestErrosDeEntradaEmPortugues(t *testing.T) {
	x := novo(t)
	r := httptest.NewRequest("POST", "/api/auth/entrar", strings.NewReader("{quebrado"))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	x.h.ServeHTTP(w, r)
	if !strings.Contains(w.Body.String(), "JSON malformado.") {
		t.Fatal(w.Body.String())
	}
	r = httptest.NewRequest("POST", "/api/auth/entrar", strings.NewReader("email=a"))
	w = httptest.NewRecorder()
	x.h.ServeHTTP(w, r)
	if !strings.Contains(w.Body.String(), "Envie os dados no formato JSON.") {
		t.Fatal(w.Body.String())
	}
	admin := x.configurar()
	res := x.req("PUT", "/api/usuarios/abc", admin, map[string]any{})
	igual(t, "id", res.corpo["erro"], any("Identificador inválido no endereço."))
}
