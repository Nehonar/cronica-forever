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
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"time"

	"github.com/Nehonar/cronica-forever/internal/app"
	"github.com/Nehonar/cronica-forever/internal/narrate"
)

var version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]
	var err error
	switch cmd {
	case "iniciar":
		err = initConfig(args)
	case "procesar", "vigilar":
		err = run(cmd, args)
	case "demo":
		err = runDemo(args)
	case "ver":
		err = runView(args)
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
  cronica demo                 prueba completa: narra una sesión de ejemplo y abre la web
  cronica ver                  abre en el navegador la web de tu crónica
  cronica iniciar              crea cronica.json con la configuración
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
	cfgPath := fl.String("config", "cronica.json", "")
	sv := fl.String("sv", "", "")
	repo := fl.String("repo", "", "")
	publish := fl.Bool("publicar", false, "")
	max := fl.Int("max", 0, "")
	fake := fl.Bool("prueba", false, "")
	fl.Parse(args)

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
	if cfg.SavedVariables == "" {
		return fmt.Errorf("falta la ruta de Cronica.lua (campo «savedvariables» en %s)", *cfgPath)
	}

	var n narrate.Narrator = narrate.ClaudeCLI{Command: cfg.Claude, Model: cfg.Model}
	if *fake {
		n = narrate.Fake{}
	}
	r := &app.Runner{Cfg: cfg, Narrator: n, Out: os.Stdout}

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

func initConfig(args []string) error {
	fl := flag.NewFlagSet("iniciar", flag.ExitOnError)
	path := fl.String("config", "cronica.json", "")
	fl.Parse(args)
	if _, err := os.Stat(*path); err == nil {
		return fmt.Errorf("%s ya existe; edítalo directamente", *path)
	}
	wow := "/ruta/a/World of Warcraft/_forever_/WTF/Account/TU_CUENTA/SavedVariables/Cronica.lua"
	addon := "/ruta/a/World of Warcraft/_forever_/Interface/AddOns/Cronica/CronicaTextos.lua"
	if runtime.GOOS == "windows" {
		wow = `C:\Program Files (x86)\World of Warcraft\_forever_\WTF\Account\TU_CUENTA\SavedVariables\Cronica.lua`
		addon = `C:\Program Files (x86)\World of Warcraft\_forever_\Interface\AddOns\Cronica\CronicaTextos.lua`
	}
	cfg := app.Config{
		SavedVariables: wow,
		Repo:           ".",
		AddonTexts:     addon,
		Claude:         "claude",
		Publish:        false,
		MaxPerRun:      12,
		ChainWindow:    90,
	}
	b, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(*path, append(b, '\n'), 0o644); err != nil {
		return err
	}
	abs, _ := filepath.Abs(*path)
	fmt.Printf("He creado %s.\nRevisa las rutas (la carpeta exacta de Forever puede variar) y luego ejecuta «cronica procesar».\n", abs)
	return nil
}
