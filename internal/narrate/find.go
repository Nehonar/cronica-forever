package narrate

import (
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

// InstallCommand es el instalador oficial de Claude Code para este sistema
// (https://code.claude.com/docs/en/setup).
func InstallCommand() string {
	if runtime.GOOS == "windows" {
		return "irm https://claude.ai/install.ps1 | iex"
	}
	return "curl -fsSL https://claude.ai/install.sh | bash"
}
