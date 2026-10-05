package app

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeClaude crea un «claude» falso cuya sesión depende de que exista un archivo.
func fakeClaude(t *testing.T) (cmd, flag string) {
	dir := t.TempDir()
	flag = filepath.Join(dir, "sesion")
	cmd = filepath.Join(dir, "claude")
	script := "#!/bin/sh\nif [ -f '" + flag + "' ]; then echo '{\"loggedIn\": true}'; else echo '{\"loggedIn\": false}'; exit 1; fi\n"
	os.WriteFile(cmd, []byte(script), 0o755)
	return cmd, flag
}

func TestSetupOffersLoginAndWaits(t *testing.T) {
	t.Setenv("DISPLAY", ":0")
	cmd, flag := fakeClaude(t)
	var asked, opened string
	s := &Setup{
		Claude: cmd, Out: io.Discard, wait: 20 * time.Second,
		ask: func(_, q string) (bool, bool) { asked = q; return true, true },
		terminal: func(_, c string) error {
			opened = c
			os.WriteFile(flag, nil, 0o644) // el usuario inicia sesión en el navegador
			return nil
		},
	}
	if err := s.Ensure(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(asked, "iniciar sesión") || !strings.Contains(opened, "auth login") {
		t.Fatalf("pregunta=%q orden=%q", asked, opened)
	}
}

func TestSetupRespectsNo(t *testing.T) {
	t.Setenv("DISPLAY", ":0")
	cmd, _ := fakeClaude(t)
	opened := false
	s := &Setup{
		Claude: cmd, Out: io.Discard,
		ask:      func(_, _ string) (bool, bool) { return false, true },
		terminal: func(_, _ string) error { opened = true; return nil },
	}
	if err := s.Ensure(context.Background()); err == nil || opened {
		t.Fatalf("si dice que no, no se abre nada (err=%v, abierto=%v)", err, opened)
	}
}

func TestSetupOffersOfficialInstaller(t *testing.T) {
	t.Setenv("DISPLAY", ":0")
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	var asked, opened string
	s := &Setup{
		Out: io.Discard, wait: time.Second,
		ask:      func(_, q string) (bool, bool) { asked = q; return true, true },
		terminal: func(_, c string) error { opened = c; return nil },
	}
	s.Ensure(context.Background())
	if !strings.Contains(asked, "no está instalado") || !strings.Contains(opened, "https://claude.ai/install.sh") || !strings.Contains(opened, "auth login") {
		t.Fatalf("pregunta=%q orden=%q", asked, opened)
	}
}
