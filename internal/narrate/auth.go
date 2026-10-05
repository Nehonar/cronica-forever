package narrate

import (
	"github.com/Nehonar/cronica-forever/internal/system"

	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ErrNotLoggedIn indica que Claude Code no tiene sesión iniciada.
var ErrNotLoggedIn = errors.New("Claude Code no tiene la sesión iniciada")

// CheckAuth comprueba, sin gastar uso, que Claude Code está instalado y con sesión.
func (c ClaudeCLI) CheckAuth(ctx context.Context) error {
	cmdName, found := FindClaude(c.Command)
	if !found {
		return fmt.Errorf("Claude Code no está instalado")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, cmdName, "auth", "status")
	system.Hide(cmd)
	out, err := cmd.Output()
	var nf *exec.Error
	if errors.As(err, &nf) {
		return fmt.Errorf("no encuentro el comando %q: ¿está instalado Claude Code?", cmdName)
	}
	var st struct {
		LoggedIn bool `json:"loggedIn"`
	}
	if jerr := json.Unmarshal(out, &st); jerr != nil {
		if err != nil {
			return ErrNotLoggedIn
		}
		// Formato desconocido: si el comando no falló, damos la sesión por buena.
		if strings.Contains(strings.ToLower(string(out)), "not logged") {
			return ErrNotLoggedIn
		}
		return nil
	}
	if !st.LoggedIn {
		return ErrNotLoggedIn
	}
	return nil
}

// AuthChecker lo implementan los narradores que pueden comprobar la sesión.
type AuthChecker interface {
	CheckAuth(ctx context.Context) error
}
