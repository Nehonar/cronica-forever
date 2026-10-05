package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/Nehonar/cronica-forever/internal/narrate"
	"github.com/Nehonar/cronica-forever/internal/system"
)

// Setup prepara Claude Code: si no está instalado o no tiene sesión, lo pregunta
// (con una ventana si hay escritorio, o en la terminal) y abre el instalador o el
// inicio de sesión OFICIALES de Anthropic. Este programa nunca ve tus credenciales:
// el inicio de sesión se hace en el navegador, en la página de Anthropic.
type Setup struct {
	Claude string    // comando configurado (vacío = buscarlo)
	Out    io.Writer // mensajes
	In     io.Reader // respuestas en modo terminal (nil = no preguntar por terminal)
	// Para pruebas:
	ask      func(title, question string) (yes, ok bool)
	terminal func(title, command string) error
	wait     time.Duration
}

func (s *Setup) say(format string, a ...any) { fmt.Fprintf(s.Out, format+"\n", a...) }

func (s *Setup) askUser(question string) (yes, asked bool) {
	if s.ask == nil {
		s.ask = system.Ask
	}
	if system.HasDesktop() {
		if yes, ok := s.ask("Crónica", question); ok {
			return yes, true
		}
	}
	if s.In == nil {
		return false, false
	}
	fmt.Fprintf(s.Out, "%s [s/N] ", question)
	line, _ := bufio.NewReader(s.In).ReadString('\n')
	line = strings.ToLower(strings.TrimSpace(line))
	return line == "s" || line == "si" || line == "sí" || line == "y" || line == "yes", true
}

// run ejecuta un comando visible: en una ventana de terminal nueva si hay
// escritorio, o en esta misma terminal si estamos en modo interactivo.
func (s *Setup) run(title, command string) error {
	if s.terminal == nil {
		s.terminal = system.OpenTerminal
	}
	if s.In == nil || system.HasDesktop() {
		if err := s.terminal(title, command); err == nil {
			return nil
		} else if s.In == nil {
			return err
		}
	}
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell", "-NoProfile", "-Command", command)
	} else {
		cmd = exec.Command("bash", "-c", command)
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

func shellQuote(p string) string {
	if runtime.GOOS == "windows" {
		return "& '" + strings.ReplaceAll(p, "'", "''") + "'"
	}
	return "'" + strings.ReplaceAll(p, "'", `'\''`) + "'"
}

// Ensure comprueba instalación y sesión y ofrece arreglarlas. Devuelve nil si
// al final Claude Code está listo.
func (s *Setup) Ensure(ctx context.Context) error {
	if s.wait == 0 {
		s.wait = 15 * time.Minute
	}
	check := func() error { return narrate.ClaudeCLI{Command: s.Claude}.CheckAuth(ctx) }

	path, found := narrate.FindClaude(s.Claude)
	if !found {
		s.say("Claude Code no está instalado en este equipo.")
		install := narrate.InstallCommand()
		yes, asked := s.askUser("Crónica necesita Claude Code para escribir tus relatos y no está instalado.\n\n" +
			"¿Quieres instalarlo ahora con el instalador oficial de Anthropic?\n\nSe abrirá una terminal que ejecutará:\n" + install +
			"\n\nDespués se abrirá el navegador para que inicies sesión con tu cuenta de Claude.")
		if !asked {
			s.say("Instálalo con: %s\nY después inicia sesión con: claude", install)
			return fmt.Errorf("Claude Code no está instalado")
		}
		if !yes {
			s.say("De acuerdo, no instalo nada. Cuando quieras: %s", install)
			return fmt.Errorf("Claude Code no está instalado")
		}
		login := "~/.local/bin/claude auth login"
		if runtime.GOOS == "windows" {
			login = `& "$env:USERPROFILE\.local\bin\claude.exe" auth login`
		}
		sep := " && "
		if runtime.GOOS == "windows" {
			sep = "; "
		}
		if err := s.run("Instalar Claude Code", install+sep+login); err != nil {
			return fmt.Errorf("no he podido abrir el instalador: %w", err)
		}
		return s.waitReady(ctx, check)
	}

	err := check()
	if err == nil {
		s.say("✓ Claude Code está instalado y con la sesión iniciada.")
		return nil
	}
	if !errors.Is(err, narrate.ErrNotLoggedIn) {
		return err
	}
	yes, asked := s.askUser("Crónica no puede escribir tus relatos: Claude Code no tiene la sesión iniciada.\n\n" +
		"¿Quieres iniciar sesión ahora? Se abrirá el navegador en la página oficial de Anthropic; " +
		"Crónica no ve ni guarda tus credenciales.")
	if !asked || !yes {
		s.say("Para iniciar sesión más tarde, ejecuta en una terminal: claude auth login")
		return err
	}
	if err := s.run("Iniciar sesión en Claude", shellQuote(path)+" auth login"); err != nil {
		return fmt.Errorf("no he podido abrir el inicio de sesión: %w", err)
	}
	return s.waitReady(ctx, check)
}

// waitReady espera a que Claude Code quede listo (instalado y con sesión).
func (s *Setup) waitReady(ctx context.Context, check func() error) error {
	s.say("Esperando a que termines de instalar o iniciar sesión…")
	deadline := time.Now().Add(s.wait)
	for time.Now().Before(deadline) {
		if check() == nil {
			s.say("✓ Listo: Claude Code tiene la sesión iniciada.")
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
	return fmt.Errorf("Claude Code sigue sin estar listo")
}
