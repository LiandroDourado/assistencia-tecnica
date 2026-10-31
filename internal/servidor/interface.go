package servidor

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

const semInterface = `<!doctype html><html lang="pt-BR"><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Assistência Técnica</title>
<body style="font-family:system-ui,sans-serif;background:#1a1f26;color:#e8ecf1;padding:2rem;line-height:1.5">
<h1>O servidor está funcionando</h1>
<p>A interface não foi incluída nesta compilação. Gere a imagem pelo Dockerfile do projeto.</p>
</body></html>`

// interface_ entrega os arquivos da SPA. Qualquer caminho que não seja um
// arquivo existente recebe o index.html, e o roteador do navegador resolve.
func (s *Servidor) interface_(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		responderErro(w, http.StatusMethodNotAllowed, "METODO_NAO_PERMITIDO", "Método não permitido neste endereço.")
		return
	}
	nome := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if nome != "" && nome != "index.html" {
		if info, err := fs.Stat(s.app, nome); err == nil && !info.IsDir() {
			// Arquivos com hash no nome (_app/immutable) nunca mudam.
			if strings.HasPrefix(nome, "_app/immutable/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			http.ServeFileFS(w, r, s.app, nome)
			return
		}
	}
	indice, err := fs.ReadFile(s.app, "index.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	if err != nil {
		w.Write([]byte(semInterface))
		return
	}
	w.Write(indice)
}
