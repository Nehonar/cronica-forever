package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"time"

	cronicaforever "github.com/Nehonar/cronica-forever"
	"github.com/Nehonar/cronica-forever/internal/app"
	"github.com/Nehonar/cronica-forever/internal/narrate"
)

// runDemo prepara una carpeta con la web y los datos de ejemplo, narra la
// sesión de prueba con Claude y abre la web en el navegador.
func runDemo(args []string) error {
	fl := flag.NewFlagSet("demo", flag.ExitOnError)
	dir := fl.String("carpeta", "cronica-demo", "")
	fake := fl.Bool("prueba", false, "")
	port := fl.Int("puerto", 8000, "")
	noServe := fl.Bool("sin-web", false, "")
	fl.Parse(args)

	abs, err := filepath.Abs(*dir)
	if err != nil {
		return err
	}
	fmt.Printf("Preparando la demostración en %s\n", abs)
	copies := map[string]string{
		"docs/index.html":             "docs/index.html",
		"docs/app.js":                 "docs/app.js",
		"docs/estilo.css":             "docs/estilo.css",
		"personajes/Tobias-Demo.json": "personajes/Tobias-Demo.json",
		"samples/Cronica.lua":         "Cronica.lua",
	}
	for src, dst := range copies {
		b, err := fs.ReadFile(cronicaforever.Files, src)
		if err != nil {
			return err
		}
		out := filepath.Join(abs, filepath.FromSlash(dst))
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		// No pisar una ficha que el usuario haya editado.
		if _, err := os.Stat(out); err == nil && filepath.Ext(out) == ".json" {
			continue
		}
		if err := os.WriteFile(out, b, 0o644); err != nil {
			return err
		}
	}

	var n narrate.Narrator = narrate.ClaudeCLI{}
	if *fake {
		n = narrate.Fake{}
	} else {
		fmt.Println("Narrando la sesión de ejemplo con Claude (tarda alrededor de medio minuto)…")
	}
	r := &app.Runner{
		Cfg:      app.Config{SavedVariables: filepath.Join(abs, "Cronica.lua"), Repo: abs},
		Narrator: n,
		Out:      os.Stdout,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	res, err := r.Process(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Listo: %d relato(s) nuevo(s).\n", res.NewStories)
	if *noServe {
		return nil
	}
	return serve(ctx, filepath.Join(abs, "docs"), *port)
}

// runView sirve la web de la carpeta docs del repositorio.
func runView(args []string) error {
	fl := flag.NewFlagSet("ver", flag.ExitOnError)
	cfgPath := fl.String("config", "", "")
	repo := fl.String("repo", "", "")
	port := fl.Int("puerto", 8000, "")
	fl.Parse(args)
	*cfgPath = app.ResolveConfig(*cfgPath)
	dir := *repo
	if dir == "" {
		if cfg, err := app.LoadConfig(*cfgPath); err == nil && cfg.Repo != "" {
			dir = cfg.Repo
		} else {
			dir = "."
		}
	}
	docs := filepath.Join(dir, "docs")
	if _, err := os.Stat(filepath.Join(docs, "index.html")); err != nil {
		return fmt.Errorf("no encuentro la web en %s", docs)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return serve(ctx, docs, *port)
}

func serve(ctx context.Context, dir string, port int) error {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return fmt.Errorf("no puedo usar el puerto %d (¿está ocupado? prueba con -puerto 8080): %w", port, err)
	}
	url := fmt.Sprintf("http://localhost:%d/", port)
	srv := &http.Server{Handler: http.FileServer(http.Dir(dir)), ReadHeaderTimeout: 10 * time.Second}
	fmt.Printf("\nAbre en el navegador: %s\n(Ctrl+C para cerrar)\n", url)
	openBrowser(url)
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		srv.Shutdown(sctx)
	}()
	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start() // si no hay navegador, basta con el enlace impreso
}
