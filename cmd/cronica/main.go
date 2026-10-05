// Crónica: convierte lo que haces en WoW: Forever en la historia de tu personaje.
//
//	cronica demo                 prueba completa con datos de ejemplo y abre la web
//	cronica iniciar              crea cronica.json con la configuración
//	cronica procesar             una pasada: lee el addon, narra lo nuevo y guarda
//	cronica vigilar              se queda esperando y procesa cada vez que el juego guarda
//	cronica version
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	cronicaforever "github.com/Nehonar/cronica-forever"
	"github.com/Nehonar/cronica-forever/internal/app"
	"github.com/Nehonar/cronica-forever/internal/narrate"
	"github.com/Nehonar/cronica-forever/internal/system"
	"github.com/Nehonar/cronica-forever/internal/wow"
)

var version = "0.3.0"

func main() {
	if len(os.Args) < 2 {
		// Doble clic: Crónica en la bandeja; la primera vez, además, la página «Preparar Crónica».
		if err := defaultAction(); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			if f, ferr := os.OpenFile(system.DefaultLogFile(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); ferr == nil {
				fmt.Fprintf(f, "%s  Error: %v\n", time.Now().Format("2006-01-02 15:04:05"), err)
				f.Close()
			}
			system.Alert("Crónica", "Crónica no ha podido arrancar:\n\n"+err.Error())
			os.Exit(1)
		}
		return
	}
	cmd, args := os.Args[1], os.Args[2:]
	var err error
	switch cmd {
	case "iniciar":
		err = runWizard(args)
	case "procesar", "vigilar":
		err = run(cmd, args)
	case "demo":
		err = runDemo(args)
	case "ver":
		err = runView(args)
	case "instalar":
		err = runInstall(args)
	case "desinstalar":
		var msg string
		if msg, err = system.Uninstall(); err == nil {
			fmt.Println(msg)
		}
	case "estado":
		err = runStatus(args)
	case "preparar":
		err = runPrepare(args)
	case "bandeja":
		err = runTray(args)
	case "instalar-addon":
		err = installAddonCmd(args)
	case "version", "-v", "--version":
		fmt.Println("cronica", version)
	case "ayuda", "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "Orden desconocida: %s\n\n", cmd)
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Print(`Crónica ` + version + ` — la historia de tu personaje de WoW: Forever

Uso:
  cronica bandeja              Crónica completa: icono en la bandeja, vigila, narra y abre el chat del cronista
  cronica demo                 prueba completa: narra una sesión de ejemplo y abre la web
  cronica ver                  abre en el navegador la web de tu crónica
  cronica iniciar              busca WoW, instala el addon y crea cronica.json
  cronica instalar-addon       reinstala o actualiza el addon en la carpeta del juego
  cronica preparar             comprueba Claude Code y, si hace falta, te ayuda a instalarlo e iniciar sesión
  cronica instalar             arranca «vigilar» solo al iniciar sesión en el PC (una sola vez)
  cronica desinstalar          quita el arranque automático
  cronica estado               comprueba Claude, la configuración y el registro
  cronica procesar [opciones]  una pasada: lee el addon, narra lo nuevo y guarda
  cronica vigilar  [opciones]  procesa cada vez que el juego guarda (al salir o con /reload)
  cronica version

Opciones:
  -config ruta     archivo de configuración (por defecto cronica.json)
  -sv ruta         archivo Cronica.lua del addon (sustituye al de la configuración)
  -repo ruta       carpeta del repositorio (sustituye a la de la configuración)
  -publicar        subir los cambios a GitHub
  -max n           máximo de relatos nuevos por pasada
  -prueba          no llama a Claude: usa textos de prueba
  -carpeta ruta    (demo) dónde preparar la demostración (por defecto ./cronica-demo)
  -puerto n        (demo, ver) puerto de la web local (por defecto 8000)
`)
}

func run(cmd string, args []string) error {
	fl := flag.NewFlagSet(cmd, flag.ExitOnError)
	cfgPath := fl.String("config", "", "")
	sv := fl.String("sv", "", "")
	repo := fl.String("repo", "", "")
	publish := fl.Bool("publicar", false, "")
	max := fl.Int("max", 0, "")
	fake := fl.Bool("prueba", false, "")
	logFile := fl.String("registro", "", "")
	atBoot := fl.Bool("arranque", false, "")
	fl.Parse(args)
	*cfgPath = app.ResolveConfig(*cfgPath)

	out := io.Writer(os.Stdout)
	if *logFile != "" {
		if err := os.MkdirAll(filepath.Dir(*logFile), 0o755); err == nil {
			if f, err := os.OpenFile(*logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
				defer f.Close()
				out = io.MultiWriter(os.Stdout, f)
			}
		}
	}

	cfg, err := app.LoadConfig(*cfgPath)
	if err != nil && !(errors.Is(err, fs.ErrNotExist) && *sv != "") {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("no encuentro %s. Ejecuta primero «cronica iniciar»", *cfgPath)
		}
		return err
	}
	if *sv != "" {
		cfg.SavedVariables = *sv
	}
	if *repo != "" {
		cfg.Repo = *repo
	}
	if cfg.Repo == "" {
		cfg.Repo = "."
	}
	if *publish {
		cfg.Publish = true
	}
	if *max > 0 {
		cfg.MaxPerRun = *max
	}
	if cfg.SavedVariables == "" && cfg.WoW == "" {
		return fmt.Errorf("falta la carpeta del juego (campo «wow») o la ruta de Cronica.lua («savedvariables») en %s", *cfgPath)
	}

	if *atBoot {
		// Al arrancar el PC, dar tiempo a que el escritorio y la red estén listos,
		// y si Claude no está listo, preguntar con una ventana.
		time.Sleep(20 * time.Second)
		if !*fake {
			(&app.Setup{Claude: cfg.Claude, Out: out}).Ensure(context.Background())
		}
	}

	var n narrate.Narrator = narrate.ClaudeCLI{Command: cfg.Claude, Model: cfg.Model}
	if *fake {
		n = narrate.Fake{}
	}
	r := &app.Runner{Cfg: cfg, Narrator: n, Out: out, Notify: system.Notify}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if cmd == "vigilar" {
		return r.Watch(ctx, 5*time.Second)
	}
	res, err := r.Process(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Listo: %d relato(s) nuevo(s), %d pendiente(s), %d fallido(s).\n", res.NewStories, res.Pending, res.Failed)
	return nil
}

// installAddonCmd reinstala (o actualiza) el addon en la carpeta del juego configurada.
func installAddonCmd(args []string) error {
	fl := flag.NewFlagSet("instalar-addon", flag.ExitOnError)
	cfgPath := fl.String("config", "", "")
	fl.Parse(args)
	*cfgPath = app.ResolveConfig(*cfgPath)
	cfg, err := app.LoadConfig(*cfgPath)
	if err != nil {
		return fmt.Errorf("no puedo leer %s (ejecuta antes «cronica iniciar»): %w", *cfgPath, err)
	}
	var f wow.Flavor
	if cfg.WoW != "" {
		f = wow.Flavor{Name: filepath.Base(cfg.WoW), Path: cfg.WoW}
	} else {
		w := newWizard()
		if f, err = w.chooseFlavor(); err != nil {
			return err
		}
	}
	dir, err := wow.InstallAddon(f, cronicaforever.Files)
	if err != nil {
		return err
	}
	fmt.Println("✓ Addon instalado en", dir, "— si el juego está abierto, haz /reload.")
	return nil
}

func defaultAction() (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("error inesperado: %v", p)
		}
	}()
	hideOwnConsole()
	return runTray(nil)
}
