// Package wow localiza la instalación de World of Warcraft e instala el addon.
package wow

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Flavor es una versión instalada del juego (una carpeta tipo _classic_, _forever_…).
type Flavor struct {
	Name string // nombre de la carpeta, p. ej. «_forever_»
	Path string // ruta completa
}

// candidates devuelve carpetas donde suele estar World of Warcraft.
func candidates() []string {
	var out []string
	if runtime.GOOS == "windows" {
		for d := 'C'; d <= 'H'; d++ {
			root := string(d) + `:\`
			for _, sub := range []string{`Program Files (x86)\World of Warcraft`, `Program Files\World of Warcraft`, `World of Warcraft`, `Games\World of Warcraft`, `Juegos\World of Warcraft`, `Battle.net\World of Warcraft`} {
				out = append(out, filepath.Join(root, sub))
			}
		}
		return out
	}
	home, _ := os.UserHomeDir()
	globs := []string{
		filepath.Join(home, "Games", "*", "drive_c", "Program Files (x86)", "World of Warcraft"), // Lutris
		filepath.Join(home, "Games", "*", "drive_c", "Program Files", "World of Warcraft"),
		filepath.Join(home, ".wine", "drive_c", "Program Files (x86)", "World of Warcraft"),
		filepath.Join(home, ".local", "share", "Steam", "steamapps", "compatdata", "*", "pfx", "drive_c", "Program Files (x86)", "World of Warcraft"),
	}
	for _, g := range globs {
		m, _ := filepath.Glob(g)
		out = append(out, m...)
	}
	return out
}

// Find busca instalaciones del juego. Las que parecen de Forever van primero.
func Find(extra ...string) []Flavor {
	var out []Flavor
	seen := map[string]bool{}
	for _, root := range append(extra, candidates()...) {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() || !strings.HasPrefix(e.Name(), "_") {
				continue
			}
			p := filepath.Join(root, e.Name())
			if seen[p] || !isFlavor(p) {
				continue
			}
			seen[p] = true
			out = append(out, Flavor{Name: e.Name(), Path: p})
		}
	}
	rank := func(f Flavor) int {
		n := strings.ToLower(f.Name)
		switch {
		case strings.Contains(n, "forever"):
			return 0
		case strings.Contains(n, "beta"):
			return 1
		case strings.Contains(n, "ptr"):
			return 3
		}
		return 2
	}
	sort.SliceStable(out, func(i, j int) bool { return rank(out[i]) < rank(out[j]) })
	return out
}

func isFlavor(p string) bool {
	for _, x := range []string{"Interface", "WTF", "Wow.exe", "WowB.exe", "WowT.exe", "WowClassic.exe"} {
		if _, err := os.Stat(filepath.Join(p, x)); err == nil {
			return true
		}
	}
	return false
}

// Accounts lista las cuentas (carpetas de WTF/Account) de una instalación.
func Accounts(f Flavor) []string {
	entries, err := os.ReadDir(filepath.Join(f.Path, "WTF", "Account"))
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() && e.Name() != "SavedVariables" {
			out = append(out, e.Name())
		}
	}
	return out
}

// SavedVariablesPath es la ruta de Cronica.lua para una cuenta.
func SavedVariablesPath(f Flavor, account string) string {
	return filepath.Join(f.Path, "WTF", "Account", account, "SavedVariables", "Cronica.lua")
}

// AddonDir es la carpeta del addon Crónica en esa instalación.
func AddonDir(f Flavor) string { return filepath.Join(f.Path, "Interface", "AddOns", "Cronica") }

// LatestSavedVariables busca, entre todas las cuentas, el Cronica.lua más reciente.
func LatestSavedVariables(flavorPath string) string {
	m, _ := filepath.Glob(filepath.Join(flavorPath, "WTF", "Account", "*", "SavedVariables", "Cronica.lua"))
	best, bestT := "", int64(0)
	for _, p := range m {
		if info, err := os.Stat(p); err == nil && info.ModTime().Unix() > bestT {
			best, bestT = p, info.ModTime().Unix()
		}
	}
	return best
}

// InstallAddon copia los archivos del addon (desde files, con la carpeta
// «addon/Cronica») a Interface/AddOns/Cronica. No pisa CronicaTextos.lua si ya
// existe, porque lo escribe el programa con tus relatos.
func InstallAddon(f Flavor, files fs.FS) (string, error) {
	dst := AddonDir(f)
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return "", err
	}
	entries, err := fs.ReadDir(files, "addon/Cronica")
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		out := filepath.Join(dst, e.Name())
		if e.Name() == "CronicaTextos.lua" {
			if _, err := os.Stat(out); err == nil {
				continue
			}
		}
		b, err := fs.ReadFile(files, "addon/Cronica/"+e.Name())
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(out, b, 0o644); err != nil {
			return "", fmt.Errorf("no puedo escribir en %s (¿WoW está en Archivos de programa y hace falta permiso?): %w", dst, err)
		}
	}
	return dst, nil
}
