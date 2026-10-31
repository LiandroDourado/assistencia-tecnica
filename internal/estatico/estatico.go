// Package estatico embute a interface (SvelteKit compilado) no executável.
// O build do front-end grava em internal/estatico/app/.
package estatico

import (
	"embed"
	"io/fs"
)

//go:embed all:app
var arquivos embed.FS

// App devolve os arquivos da interface, com a raiz em app/.
func App() fs.FS {
	sub, _ := fs.Sub(arquivos, "app")
	return sub
}
