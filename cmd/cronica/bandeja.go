package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"time"

	"github.com/Nehonar/cronica-forever/internal/app"
	"github.com/Nehonar/cronica-forever/internal/narrate"
	"github.com/Nehonar/cronica-forever/internal/system"
	"github.com/Nehonar/cronica-forever/internal/tray"
	"github.com/Nehonar/cronica-forever/internal/ui"
)

// runTray arranca Crónica completa: vigila el archivo del addon, sirve la web y
// el chat del cronista en local, y muestra el icono en la bandeja del sistema.
func runTray(args []string) error {
	fl := flag.NewFlagSet("bandeja", flag.ExitOnError)
	cfgPath := fl.String("config", "cronica.json", "")
	logFile := fl.String("registro", "", "")
	atBoot := fl.Bool("arranque", false, "")
	headless := fl.Bool("sin-bandeja", false, "")
	port := fl.Int("puerto", 8737, "")
	fake := fl.Bool("prueba", false, "")
	sv := fl.String("sv", "", "")
	repo := fl.String("repo", "", "")
	fl.Parse(args)

	cfg, err := app.LoadConfig(*cfgPath)
	if err != nil && *sv == "" {
		return fmt.Errorf("no puedo leer %s (ejecuta antes «cronica iniciar»): %w", *cfgPath, err)
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

	out := io.Writer(os.Stdout)
	if *logFile != "" {
		if err := os.MkdirAll(filepath.Dir(*logFile), 0o755); err == nil {
			if f, err := os.OpenFile(*logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
				defer f.Close()
				out = io.MultiWriter(os.Stdout, f)
			}
		}
	}

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
		if last.Pending > 0 {
			st.PendingText = fmt.Sprintf("%d misión(es) o cadena(s) por completar", last.Pending)
		}
		trayApp.Update(st)
	}
	r.OnResult = func(res app.Result, err error) {
		mu.Lock()
		if res.LastTitle == "" {
			res.LastTitle = last.LastTitle
		}
		last = res
		mu.Unlock()
		update()
	}

	srv := &ui.Server{Runner: r, Narrator: n, Port: *port}
	if err := srv.Start(); err != nil {
		return fmt.Errorf("no puedo arrancar el servidor local: %w", err)
	}
	defer srv.Stop()
	fmt.Fprintf(out, "%s  Web local: %s  ·  Misiones: %smisiones/  ·  Cronista: %spersonaje/\n", time.Now().Format("15:04:05"), srv.URL(), srv.URL(), srv.URL())

	setup := func() {
		if *fake {
			return
		}
		(&app.Setup{Claude: cfg.Claude, Out: out}).Ensure(ctx)
		r.CheckClaude(ctx)
		update()
	}
	go func() {
		if *atBoot {
			time.Sleep(20 * time.Second)
		}
		setup()
		r.Watch(ctx, 2*time.Second)
	}()

	if *headless {
		<-ctx.Done()
		return nil
	}
	tray.Run(tray.Actions{
		OpenChronicle: func() { openBrowser(srv.URL()) },
		OpenQuests:    func() { openBrowser(srv.URL() + "misiones/") },
		OpenCharacter: func(key string) {
			u := srv.URL() + "personaje/"
			if key != "" {
				u += "?p=" + key
			}
			openBrowser(u)
		},
		NarrateNow: func() { go r.Process(ctx) },
		FixClaude:  func() { go setup() },
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
