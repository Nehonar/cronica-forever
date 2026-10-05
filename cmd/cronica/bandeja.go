package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"

	cronicaforever "github.com/Nehonar/cronica-forever"
	"github.com/Nehonar/cronica-forever/internal/app"
	"github.com/Nehonar/cronica-forever/internal/narrate"
	"github.com/Nehonar/cronica-forever/internal/system"
	"github.com/Nehonar/cronica-forever/internal/tray"
	"github.com/Nehonar/cronica-forever/internal/ui"
	"github.com/Nehonar/cronica-forever/internal/wow"
)

// runTray arranca Crónica completa: vigila el archivo del addon, sirve la web y
// el chat del cronista en local, y muestra el icono en la bandeja del sistema.
func runTray(args []string) error {
	fl := flag.NewFlagSet("bandeja", flag.ExitOnError)
	cfgPath := fl.String("config", "", "")
	logFile := fl.String("registro", "", "")
	atBoot := fl.Bool("arranque", false, "")
	headless := fl.Bool("sin-bandeja", false, "")
	port := fl.Int("puerto", 8737, "")
	fake := fl.Bool("prueba", false, "")
	sv := fl.String("sv", "", "")
	repo := fl.String("repo", "", "")
	fl.Parse(args)
	*cfgPath = app.ResolveConfig(*cfgPath)

	// ¿Ya está abierta? Entonces no abrir otra: como mucho, enseñar la página.
	if h, ok := alreadyRunning(*port); ok {
		if !*atBoot {
			openBrowser(fmt.Sprintf("http://127.0.0.1:%d%s", *port, h.Page))
		}
		return nil
	}

	home := filepath.Dir(*cfgPath)
	if abs, err := filepath.Abs(home); err == nil {
		home = abs
	}
	cfg, err := app.LoadConfig(*cfgPath)
	firstRun := false
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) && *sv == "" {
			return fmt.Errorf("no puedo leer %s: %w", *cfgPath, err)
		}
		// Primera vez: se configura desde la página «Preparar Crónica».
		cfg, firstRun = app.DefaultConfig(home), true
		os.MkdirAll(home, 0o755)
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
	ensureWeb(cfg.Repo)
	// Mantener el addon al día con la versión del programa (no toca CronicaTextos.lua).
	if cfg.WoW != "" {
		if _, err := os.Stat(cfg.WoW); err == nil {
			wow.InstallAddon(wow.Flavor{Path: cfg.WoW}, cronicaforever.Files)
		}
	}
	needsSetup := firstRun || (cfg.WoW == "" && cfg.SavedVariables == "")

	if *logFile == "" {
		*logFile = system.DefaultLogFile()
	}
	out := io.Writer(os.Stdout)
	if *logFile != "" {
		if err := os.MkdirAll(filepath.Dir(*logFile), 0o755); err == nil {
			if f, err := os.OpenFile(*logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
				defer f.Close()
				out = io.MultiWriter(os.Stdout, f)
			}
		}
	}
	fmt.Fprintf(out, "%s  Crónica %s arrancando (configuración: %s)\n", time.Now().Format("15:04:05"), version, *cfgPath)

	var n narrate.Narrator = narrate.ClaudeCLI{Command: cfg.Claude, Model: cfg.Model}
	if *fake {
		n = narrate.Fake{}
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var (
		mu      sync.Mutex
		trayApp *tray.App
		last    app.Result
	)
	r := &app.Runner{Cfg: cfg, Narrator: n, Out: out, Notify: system.Notify, Inform: system.Toast}
	update := func() {
		mu.Lock()
		defer mu.Unlock()
		if trayApp == nil {
			return
		}
		st := tray.State{
			Status:    "vigilando · " + time.Now().Format("15:04"),
			ClaudeOK:  r.Status().OK || *fake,
			LastStory: last.LastTitle,
		}
		for _, c := range last.NeedsBackstory {
			st.NewChars = append(st.NewChars, tray.Character{Key: c.Key, Label: c.Name + " (" + c.Race + " " + c.Class + ")"})
		}
		if r.Config().Publish {
			p := r.PublishStatus()
			switch {
			case p.Err != "":
				st.GitHub, st.GitHubError = "no se ha podido publicar", true
			case !p.At.IsZero():
				st.GitHub = "publicado a las " + p.At.Format("15:04")
			default:
				st.GitHub = "al día"
			}
		}
		if last.Pending > 0 {
			st.PendingText = fmt.Sprintf("%d misión(es) o cadena(s) por completar", last.Pending)
		}
		trayApp.Update(st)
	}
	r.OnPublish = func() { go update() }
	r.OnResult = func(res app.Result, err error) {
		mu.Lock()
		if res.LastTitle == "" {
			res.LastTitle = last.LastTitle
		}
		last = res
		mu.Unlock()
		update()
	}

	exe, _ := os.Executable()
	if e, err := filepath.EvalSymlinks(exe); err == nil {
		exe = e
	}
	srv := &ui.Server{Runner: r, Narrator: n, Port: *port,
		ConfigPath: *cfgPath, Version: version, Exe: exe, AddonFiles: cronicaforever.Files,
		PrepareWeb: ensureWeb, Fake: *fake,
		OnChange: update,
		OnClaudeReady: func() {
			r.CheckClaude(ctx)
			update()
			r.Process(ctx)
		},
	}
	if err := srv.Start(); err != nil {
		return fmt.Errorf("no puedo arrancar el servidor local: %w", err)
	}
	defer srv.Stop()
	fmt.Fprintf(out, "%s  Web local: %s  ·  Misiones: %smisiones/  ·  Cronista: %spersonaje/\n", time.Now().Format("15:04:05"), srv.URL(), srv.URL(), srv.URL())

	// show lleva la pestaña de Crónica ya abierta a path y la trae delante; si
	// no hay ninguna (o no se puede traer delante), abre una nueva.
	show := func(path string) {
		if system.FocusWindow("Crónica (este PC)") && srv.Navigate(path) {
			return
		}
		openBrowser(srv.URL() + strings.TrimPrefix(path, "/"))
	}
	setupPage := func() { show("/preparar/") }
	go func() {
		if needsSetup {
			if !*atBoot {
				setupPage()
			}
		} else if !*fake {
			if *atBoot {
				time.Sleep(20 * time.Second)
			}
			if err := r.CheckClaude(ctx); err != nil {
				update()
				if !*atBoot {
					setupPage()
				} else if yes, ok := system.Ask("Crónica", "El cronista no puede escribir: Claude no tiene la sesión iniciada o no está instalado.\n\n¿Quieres arreglarlo ahora? Se abrirá la página de configuración de Crónica."); ok && yes {
					setupPage()
				}
			}
		}
		update()
		r.Watch(ctx, 2*time.Second)
	}()

	if *headless {
		<-ctx.Done()
		return nil
	}
	tray.Run(tray.Actions{
		OpenChronicle: func() { show("/") },
		OpenQuests:    func() { show("/misiones/") },
		OpenCharacter: func(key string) {
			u := "/personaje/"
			if key != "" {
				u += "?p=" + url.QueryEscape(key)
			}
			show(u)
		},
		NarrateNow: func() { go r.Process(ctx) },
		FixClaude:  setupPage,
		Settings:   setupPage,
		Quit:       cancel,
	}, func(a *tray.App) {
		mu.Lock()
		trayApp = a
		mu.Unlock()
		update()
		go func() { <-ctx.Done(); a.Quit() }()
	})
	return nil
}

// alreadyRunning pregunta en el puerto de Crónica si ya hay una abierta.
func alreadyRunning(port int) (ui.Hello, bool) {
	var h ui.Hello
	c := http.Client{Timeout: 2 * time.Second}
	resp, err := c.Get(fmt.Sprintf("http://127.0.0.1:%d/api/hola", port))
	if err != nil {
		return h, false
	}
	defer resp.Body.Close()
	if json.NewDecoder(resp.Body).Decode(&h) != nil || h.App != "cronica" {
		return h, false
	}
	if h.Page == "" {
		h.Page = "/"
	}
	return h, true
}
