package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Nehonar/cronica-forever/internal/narrate"
	"github.com/Nehonar/cronica-forever/internal/system"
)

// Setup prepara Claude Code desde la terminal (órdenes «cronica preparar» e
// «cronica iniciar»): si no está instalado o no tiene sesión, lo pregunta y
// ejecuta el instalador o el inicio de sesión OFICIALES de Anthropic. Este
// programa nunca ve tus credenciales: el inicio de sesión se hace en el
// navegador, en la página de Anthropic. (La bandeja usa la página «Preparar
// Crónica» en su lugar.)
type Setup struct {
	Claude string    // comando configurado (vacío = buscarlo)
	Out    io.Writer // mensajes
	In     io.Reader // respuestas (nil = no preguntar por terminal)
	// Para pruebas:
	ask      func(title, question string) (yes, ok bool)
	run      func(name string, args ...string) error
	download func(ctx context.Context) (string, []string, func(), error)
	wait     time.Duration
}

func (s *Setup) say(format string, a ...any) { fmt.Fprintf(s.Out, format+"\n", a...) }

func (s *Setup) askUser(question string) (yes, asked bool) {
	if s.ask == nil {
		s.ask = system.Ask
	}
	if s.In == nil && system.HasDesktop() {
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

// exec ejecuta un comando en esta misma terminal.
func (s *Setup) exec(name string, args ...string) error {
	if s.run != nil {
		return s.run(name, args...)
	}
	cmd := exec.Command(name, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, s.Out, s.Out
	return cmd.Run()
}

// Ensure comprueba instalación y sesión y ofrece arreglarlas. Devuelve nil si
// al final Claude Code está listo.
func (s *Setup) Ensure(ctx context.Context) error {
	if s.wait == 0 {
		s.wait = 2 * time.Minute
	}
	if s.download == nil {
		s.download = narrate.DownloadInstaller
	}
	check := func() error { return narrate.ClaudeCLI{Command: s.Claude}.CheckAuth(ctx) }

	path, found := narrate.FindClaude(s.Claude)
	if !found {
		s.say("Claude Code no está instalado en este equipo.")
		yes, asked := s.askUser("Crónica necesita Claude Code para escribir tus relatos y no está instalado.\n\n" +
			"¿Quieres instalarlo ahora con el instalador oficial de Anthropic (" + narrate.InstallerURL() + ")?\n\n" +
			"Después se abrirá el navegador para que inicies sesión con tu cuenta de Claude.")
		if !asked || !yes {
			s.say("Puedes instalarlo cuando quieras desde la página de configuración de Crónica, o siguiendo https://code.claude.com/docs/en/setup")
			return fmt.Errorf("Claude Code no está instalado")
		}
		name, args, cleanup, err := s.download(ctx)
		if err != nil {
			return err
		}
		defer cleanup()
		if err := s.exec(name, args...); err != nil {
			return fmt.Errorf("el instalador ha fallado: %w", err)
		}
		if path, found = narrate.FindClaude(s.Claude); !found {
			return fmt.Errorf("no encuentro Claude Code después de instalarlo")
		}
		s.say("Abriendo el navegador para que inicies sesión…")
		if err := s.exec(path, "auth", "login"); err != nil {
			return fmt.Errorf("el inicio de sesión no ha terminado: %w", err)
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
		s.say("Para iniciar sesión más tarde: claude auth login")
		return err
	}
	if err := s.exec(path, "auth", "login"); err != nil {
		return fmt.Errorf("el inicio de sesión no ha terminado: %w", err)
	}
	return s.waitReady(ctx, check)
}

// waitReady espera a que Claude Code quede listo (instalado y con sesión).
func (s *Setup) waitReady(ctx context.Context, check func() error) error {
	deadline := time.Now().Add(s.wait)
	for {
		if check() == nil {
			s.say("✓ Listo: Claude Code tiene la sesión iniciada.")
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("Claude Code sigue sin estar listo")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}
