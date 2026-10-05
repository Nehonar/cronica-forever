package main

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"

	cronicaforever "github.com/Nehonar/cronica-forever"
)

// ensureWeb copia (o actualiza) la web que va dentro del programa a la carpeta
// docs de la crónica, para que se pueda ver en local y publicar en GitHub Pages.
func ensureWeb(repo string) error {
	docs := filepath.Join(repo, "docs")
	if err := os.MkdirAll(filepath.Join(docs, "data"), 0o755); err != nil {
		return err
	}
	for _, name := range []string{"index.html", "app.js", "estilo.css"} {
		b, err := fs.ReadFile(cronicaforever.Files, "docs/"+name)
		if err != nil {
			return err
		}
		dst := filepath.Join(docs, name)
		if cur, err := os.ReadFile(dst); err == nil && bytes.Equal(cur, b) {
			continue
		}
		if err := os.WriteFile(dst, b, 0o644); err != nil {
			return err
		}
	}
	nj := filepath.Join(docs, ".nojekyll")
	if _, err := os.Stat(nj); err != nil {
		os.WriteFile(nj, nil, 0o644)
	}
	return nil
}
