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
	// Todo lo que va dentro del programa bajo docs/ (la crónica, la app de banco
	// y oficios y su tabla de recetas); los datos de cada uno no se tocan.
	err := fs.WalkDir(cronicaforever.Files, "docs", func(name string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() {
			return err
		}
		b, err := fs.ReadFile(cronicaforever.Files, name)
		if err != nil {
			return err
		}
		dst := filepath.Join(repo, filepath.FromSlash(name))
		if cur, err := os.ReadFile(dst); err == nil && bytes.Equal(cur, b) {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dst, b, 0o644)
	})
	if err != nil {
		return err
	}
	nj := filepath.Join(docs, ".nojekyll")
	if _, err := os.Stat(nj); err != nil {
		os.WriteFile(nj, nil, 0o644)
	}
	return nil
}
