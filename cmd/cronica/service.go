package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Nehonar/cronica-forever/internal/app"
	"github.com/Nehonar/cronica-forever/internal/narrate"
	"github.com/Nehonar/cronica-forever/internal/system"
)

func runInstall(args []string) error {
	fl := flag.NewFlagSet("instalar", flag.ExitOnError)
	cfgPath := fl.String("config", "", "")
	fl.Parse(args)
	*cfgPath = app.ResolveConfig(*cfgPath)
	absCfg, err := filepath.Abs(*cfgPath)
	if err != nil {
		return err
	}
	cfg, err := app.LoadConfig(absCfg)
	if err != nil {
		return fmt.Errorf("no puedo leer %s (ejecuta antes «cronica iniciar»): %w", absCfg, err)
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if exe, err = filepath.EvalSymlinks(exe); err != nil {
		return err
	}
	work := cfg.Repo
	if work == "" {
		work = filepath.Dir(absCfg)
	}
	if err := (&app.Setup{Claude: cfg.Claude, Out: os.Stdout, In: os.Stdin}).Ensure(context.Background()); err != nil {
		fmt.Println("Aviso:", err, "— al arrancar el PC te lo volveré a preguntar.")
	}
	msg, err := system.Install(system.Service{Exe: exe, Config: absCfg, WorkDir: work, LogFile: system.DefaultLogFile()})
	if err != nil {
		return err
	}
	fmt.Println(msg)
	fmt.Println("A partir de ahora «cronica vigilar» arranca solo al iniciar sesión. Para quitarlo: cronica desinstalar")
	return nil
}

func runStatus(args []string) error {
	fl := flag.NewFlagSet("estado", flag.ExitOnError)
	cfgPath := fl.String("config", "", "")
	fl.Parse(args)
	*cfgPath = app.ResolveConfig(*cfgPath)

	cfg, cfgErr := app.LoadConfig(*cfgPath)
	fmt.Println("Crónica", version)
	if err := (narrate.ClaudeCLI{Command: cfg.Claude}).CheckAuth(context.Background()); err != nil {
		fmt.Println("✗ Claude:", err)
	} else {
		fmt.Println("✓ Claude Code tiene la sesión iniciada")
	}
	if cfgErr != nil {
		fmt.Println("✗ Configuración:", cfgErr)
	} else {
		fmt.Println("✓ Configuración:", *cfgPath)
		check := func(label, p string) {
			if p == "" {
				fmt.Printf("  · %s: (sin configurar)\n", label)
			} else if _, err := os.Stat(p); err != nil {
				fmt.Printf("  ✗ %s: %s (no existe todavía)\n", label, p)
			} else {
				fmt.Printf("  ✓ %s: %s\n", label, p)
			}
		}
		check("Archivo del addon", cfg.SavedVariables)
		check("Repositorio", cfg.Repo)
		check("Textos para el addon", cfg.AddonTexts)
		fmt.Printf("  · Publicar en GitHub: %v\n", cfg.Publish)
	}
	logf := system.DefaultLogFile()
	f, err := os.Open(logf)
	if err != nil {
		fmt.Println("· Registro: aún no hay (", logf, ")")
		return nil
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
		if len(lines) > 10 {
			lines = lines[1:]
		}
	}
	fmt.Println("· Últimas líneas del registro (", logf, "):")
	for _, l := range lines {
		fmt.Println("   ", l)
	}
	return nil
}

func runPrepare(args []string) error {
	fl := flag.NewFlagSet("preparar", flag.ExitOnError)
	cfgPath := fl.String("config", "", "")
	fl.Parse(args)
	*cfgPath = app.ResolveConfig(*cfgPath)
	cfg, _ := app.LoadConfig(*cfgPath) // sin configuración también sirve
	return (&app.Setup{Claude: cfg.Claude, Out: os.Stdout, In: os.Stdin}).Ensure(context.Background())
}
