package narrate

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// FindClaude busca el ejecutable de Claude Code. Primero en el PATH y, si no
// está (pasa a menudo en servicios que arrancan con el PC), en la carpeta
// donde lo deja el instalador oficial (~/.local/bin).
func FindClaude(configured string) (string, bool) {
	if configured != "" && configured != "claude" {
		if p, err := exec.LookPath(configured); err == nil {
			return p, true
		}
		return configured, false
	}
	if p, err := exec.LookPath("claude"); err == nil {
		return p, true
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "claude", false
	}
	name := "claude"
	if runtime.GOOS == "windows" {
		name = "claude.exe"
	}
	p := filepath.Join(home, ".local", "bin", name)
	if _, err := os.Stat(p); err == nil {
		return p, true
	}
	return "claude", false
}

// InstallerURL es el instalador oficial de Claude Code para este sistema
// (https://code.claude.com/docs/en/setup).
func InstallerURL() string {
	if runtime.GOOS == "windows" {
		return "https://claude.ai/install.ps1"
	}
	return "https://claude.ai/install.sh"
}

// DownloadInstaller descarga el instalador oficial a un archivo temporal y
// devuelve el comando para ejecutarlo.
func DownloadInstaller(ctx context.Context) (name string, args []string, cleanup func(), err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, InstallerURL(), nil)
	if err != nil {
		return "", nil, nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", nil, nil, fmt.Errorf("no puedo descargar el instalador: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", nil, nil, fmt.Errorf("no puedo descargar el instalador: %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", nil, nil, err
	}
	ext := ".sh"
	if runtime.GOOS == "windows" {
		ext = ".ps1"
		// PowerShell 5 lee los .ps1 sin BOM como ANSI: añadirlo para que respete UTF-8.
		if !bytes.HasPrefix(body, []byte("\xef\xbb\xbf")) {
			body = append([]byte("\xef\xbb\xbf"), body...)
		}
	}
	f, err := os.CreateTemp("", "instalar-claude-*"+ext)
	if err != nil {
		return "", nil, nil, err
	}
	path := f.Name()
	cleanup = func() { os.Remove(path) }
	if _, err := f.Write(body); err != nil {
		f.Close()
		cleanup()
		return "", nil, nil, err
	}
	f.Close()
	if runtime.GOOS == "windows" {
		return "powershell", []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", path}, cleanup, nil
	}
	return "bash", []string{path}, cleanup, nil
}
