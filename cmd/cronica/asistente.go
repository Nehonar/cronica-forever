package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	cronicaforever "github.com/Nehonar/cronica-forever"
	"github.com/Nehonar/cronica-forever/internal/app"
	"github.com/Nehonar/cronica-forever/internal/system"
	"github.com/Nehonar/cronica-forever/internal/wow"
)

type wizard struct {
	in *bufio.Reader
}

func newWizard() *wizard { return &wizard{in: bufio.NewReader(os.Stdin)} }

func (w *wizard) ask(question string) string {
	fmt.Print(question + " ")
	line, _ := w.in.ReadString('\n')
	return strings.TrimSpace(line)
}

func (w *wizard) yes(question string, def bool) bool {
	hint := "[s/N]"
	if def {
		hint = "[S/n]"
	}
	a := strings.ToLower(w.ask(question + " " + hint))
	if a == "" {
		return def
	}
	return a == "s" || a == "si" || a == "sí" || a == "y" || a == "yes"
}

func title(n int, s string) {
	fmt.Printf("\n── %d. %s ──────────────────────────\n", n, s)
}

// runWizard es el asistente de instalación: se ejecuta la primera vez que abres
// el programa (o con «cronica iniciar»).
func runWizard(args []string) error {
	fl := flag.NewFlagSet("iniciar", flag.ExitOnError)
	cfgPath := fl.String("config", "", "")
	fl.Parse(args)
	path := *cfgPath
	if path == "" {
		path = filepath.Join(app.DefaultHome(), "cronica.json")
	}
	home := filepath.Dir(path)
	w := newWizard()

	fmt.Println("Crónica " + version + " — asistente de instalación")
	fmt.Println("Tus datos se guardarán en:", home)
	cfg := app.Config{Claude: "claude", MaxPerRun: 12, ChainWindow: 90, Repo: home}
	if old, err := app.LoadConfig(path); err == nil {
		if !w.yes("Ya hay una configuración. ¿Quieres rehacerla?", false) {
			return nil
		}
		cfg = old
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		return err
	}

	// 1. Juego y addon.
	title(1, "World of Warcraft Forever")
	if f, err := w.chooseFlavor(); err != nil {
		fmt.Println("✗", err)
		fmt.Println("  Puedes repetirlo más tarde con: cronica iniciar")
	} else {
		cfg.WoW, cfg.SavedVariables = f.Path, ""
		if acc := wow.Accounts(f); len(acc) == 1 {
			cfg.SavedVariables = wow.SavedVariablesPath(f, acc[0])
		}
		dir, err := wow.InstallAddon(f, cronicaforever.Files)
		if err != nil {
			fmt.Println("✗ No he podido instalar el addon:", err)
		} else {
			fmt.Println("✓ Addon instalado en", dir)
			cfg.AddonTexts = filepath.Join(dir, "CronicaTextos.lua")
		}
	}

	// 2. Publicar en GitHub (opcional).
	title(2, "Publicar tu crónica en una web (opcional)")
	fmt.Println("Crónica siempre se ve en tu PC. Si además quieres una web pública en GitHub Pages,")
	fmt.Println("necesitas un repositorio de GitHub y Git instalado.")
	if w.yes("¿Quieres publicarla en GitHub?", cfg.Publish) {
		if repo, ok := w.setupGitHub(home); ok {
			cfg.Repo, cfg.Publish = repo, true
		} else {
			cfg.Repo, cfg.Publish = home, false
		}
	} else {
		cfg.Repo, cfg.Publish = home, false
	}
	if err := ensureWeb(cfg.Repo); err != nil {
		fmt.Println("✗ No he podido preparar la web:", err)
	}

	// Guardar ya la configuración.
	b, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Println("✓ Configuración guardada en", path)

	// 3. Claude.
	title(3, "Claude")
	if err := (&app.Setup{Claude: cfg.Claude, Out: os.Stdout, In: os.Stdin}).Ensure(context.Background()); err != nil {
		fmt.Println("·", err, "— Crónica te lo volverá a preguntar al arrancar.")
	}

	// 4. Arranque automático.
	title(4, "Arranque")
	exe, _ := os.Executable()
	if exe, err := filepath.EvalSymlinks(exe); err == nil && w.yes("¿Quieres que Crónica se abra sola al encender el PC (icono en la bandeja)?", true) {
		msg, err := system.Install(system.Service{Exe: exe, Config: path, WorkDir: cfg.Repo, LogFile: system.DefaultLogFile()})
		if err != nil {
			fmt.Println("✗", err)
		} else {
			fmt.Println("✓", msg)
		}
	} else {
		fmt.Println("· Para abrirla cuando quieras, ejecuta el programa de nuevo (o «cronica bandeja»).")
	}

	fmt.Println("\nListo. En el juego, activa el addon «Crónica» y asigna teclas en Opciones → Atajos → Crónica.")
	fmt.Println("La primera vez que entres con un personaje, el icono de Crónica te propondrá crear su historia.")
	return nil
}

// chooseFlavor pregunta por la carpeta de WoW Forever: propone la que detecta y,
// si no es esa, abre el selector de carpetas (o deja escribir la ruta).
func (w *wizard) chooseFlavor() (wow.Flavor, error) {
	found := wow.Find()
	if len(found) > 0 {
		fmt.Println("He encontrado estas instalaciones del juego:")
		for i, f := range found {
			fmt.Printf("  %d) %s\n", i+1, f.Path)
		}
		a := w.ask(fmt.Sprintf("¿Cuál es WoW Forever? [número, Intro = 1, o «o» para elegir otra carpeta]"))
		if a == "" {
			return found[0], nil
		}
		if n, err := strconv.Atoi(a); err == nil && n >= 1 && n <= len(found) {
			return found[n-1], nil
		}
	} else {
		fmt.Println("No encuentro el juego en las carpetas habituales.")
	}
	for tries := 0; tries < 3; tries++ {
		fmt.Println("Elige la carpeta de WoW Forever (la que tiene Interface y WTF dentro, por ejemplo «_forever_»).")
		p, ok := system.PickFolder("Carpeta de WoW Forever (la que contiene Interface y WTF)")
		if !ok {
			p = w.ask("Escribe la ruta de la carpeta:")
		}
		if p == "" {
			return wow.Flavor{}, fmt.Errorf("no se ha elegido carpeta")
		}
		f, list, err := wow.FromPath(p)
		if err != nil {
			fmt.Println("✗", err)
			continue
		}
		if len(list) > 0 {
			for i, x := range list {
				fmt.Printf("  %d) %s\n", i+1, x.Path)
			}
			n, _ := strconv.Atoi(w.ask(fmt.Sprintf("Hay varias versiones dentro. ¿Cuál es Forever? [1-%d]", len(list))))
			if n < 1 || n > len(list) {
				n = 1
			}
			f = list[n-1]
		}
		fmt.Println("✓ Carpeta del juego:", f.Path)
		return f, nil
	}
	return wow.Flavor{}, fmt.Errorf("no he podido encontrar la carpeta del juego")
}

var reGitHub = regexp.MustCompile(`^https://github\.com/[\w.-]+/[\w.-]+?(\.git)?/?$`)

// setupGitHub clona el repositorio del usuario en la carpeta de datos y
// comprueba que se puede subir. El inicio de sesión en GitHub lo hace el gestor
// de credenciales de Git (abre el navegador la primera vez): Crónica no ve tu contraseña.
func (w *wizard) setupGitHub(home string) (string, bool) {
	if _, err := exec.LookPath("git"); err != nil {
		fmt.Println("✗ No encuentro Git.")
		if runtime.GOOS == "windows" {
			fmt.Println("  Instálalo desde https://git-scm.com/download/win (o: winget install --id Git.Git -e)")
		} else {
			fmt.Println("  Instálalo con el gestor de paquetes de tu sistema (por ejemplo: sudo apt install git)")
		}
		fmt.Println("  y vuelve a ejecutar «cronica iniciar». De momento la crónica se queda solo en tu PC.")
		return "", false
	}
	url := w.ask("Dirección de tu repositorio (por ejemplo https://github.com/TuUsuario/mi-cronica):")
	if !reGitHub.MatchString(url) {
		fmt.Println("✗ Esa dirección no parece de un repositorio de GitHub. La crónica se queda solo en tu PC.")
		return "", false
	}
	dir := filepath.Join(home, "web")
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		fmt.Println("· Ya tenía una copia; la actualizo…")
		exec.Command("git", "-C", dir, "pull", "--rebase").Run()
	} else {
		fmt.Println("· Descargando el repositorio…")
		cmd := exec.Command("git", "clone", url, dir)
		cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
		if err := cmd.Run(); err != nil {
			fmt.Println("✗ No he podido descargarlo:", err)
			return "", false
		}
	}
	fmt.Println("· Comprobando que puedes subir cambios (si es la primera vez, Git abrirá el navegador para que entres en GitHub)…")
	cmd := exec.Command("git", "-C", dir, "push", "--dry-run")
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
	if err := cmd.Run(); err != nil {
		fmt.Println("✗ No puedo subir a ese repositorio:", err)
		fmt.Println("  Revisa que es tuyo y que has iniciado sesión. La crónica se guardará igualmente y se publicará cuando lo arregles.")
	} else {
		fmt.Println("✓ GitHub listo: cada relato nuevo se publicará solo.")
	}
	fmt.Println("  Recuerda activar GitHub Pages en el repositorio: Settings → Pages → rama principal, carpeta /docs.")
	return dir, true
}
