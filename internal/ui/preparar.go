package ui

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Nehonar/cronica-forever/internal/app"
	"github.com/Nehonar/cronica-forever/internal/narrate"
	"github.com/Nehonar/cronica-forever/internal/system"
	"github.com/Nehonar/cronica-forever/internal/wow"
)

// Página «Preparar Crónica»: la carpeta del juego, Claude, GitHub y el arranque,
// todo desde el navegador. Los comandos (instalador de Claude, inicio de sesión,
// git) se ejecutan sin ventanas y su salida se enseña en la página. Las
// credenciales nunca pasan por aquí: el inicio de sesión se hace en la página de
// Anthropic y el de GitHub en la ventana del gestor de credenciales de Git.

// task es un trabajo largo en segundo plano (instalar, iniciar sesión, git).
type task struct {
	mu      sync.Mutex
	name    string
	out     strings.Builder
	running bool
	ok      bool
	err     string
	stdin   io.WriteCloser
	cancel  context.CancelFunc
	started time.Time
}

// TaskView es lo que ve la página de un trabajo.
type TaskView struct {
	Name    string `json:"nombre"`
	Output  string `json:"salida"`
	Running bool   `json:"activa"`
	OK      bool   `json:"ok"`
	Err     string `json:"error,omitempty"`
	URL     string `json:"url,omitempty"` // enlace de inicio de sesión, si lo hay
	Code    bool   `json:"codigo"`        // el comando espera que pegues un código
}

var (
	reANSI   = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]|\x1b\][^\x07]*\x07`)
	reURL    = regexp.MustCompile(`https://[^\s"'<>]+`)
	rePaste  = regexp.MustCompile(`(?i)paste code|pega el código`)
	maxShown = 12000
)

func (t *task) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.out.WriteString(reANSI.ReplaceAllString(strings.ReplaceAll(string(p), "\r\n", "\n"), ""))
	return len(p), nil
}

func (t *task) say(format string, a ...any) { fmt.Fprintf(t, format+"\n", a...) }

func (t *task) view() TaskView {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := t.out.String()
	v := TaskView{Name: t.name, Running: t.running, OK: t.ok, Err: t.err}
	for _, u := range reURL.FindAllString(out, -1) {
		if strings.Contains(u, "oauth") || strings.Contains(u, "login") || strings.Contains(u, "device") {
			v.URL = u
		}
	}
	v.Code = t.running && t.stdin != nil && rePaste.MatchString(out)
	if len(out) > maxShown {
		out = "…\n" + out[len(out)-maxShown:]
	}
	v.Output = out
	return v
}

// exec ejecuta un comando sin ventana, con su salida en el trabajo.
func (t *task) exec(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	system.Hide(cmd)
	cmd.Stdout, cmd.Stderr = t, t
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "NO_COLOR=1")
	in, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	t.mu.Lock()
	t.stdin = in
	t.mu.Unlock()
	defer func() {
		t.mu.Lock()
		t.stdin = nil
		t.mu.Unlock()
		in.Close()
	}()
	if err := cmd.Run(); err != nil {
		var nf *exec.Error
		if errors.As(err, &nf) {
			return fmt.Errorf("no encuentro %s", name)
		}
		return err
	}
	return nil
}

// startTask lanza un trabajo si no hay otro en marcha.
func (s *Server) startTask(name string, f func(ctx context.Context, t *task) error) error {
	s.taskMu.Lock()
	defer s.taskMu.Unlock()
	if s.task != nil {
		s.task.mu.Lock()
		busy := s.task.running
		s.task.mu.Unlock()
		if busy {
			return errors.New("espera a que termine lo que está en marcha")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	t := &task{name: name, running: true, cancel: cancel, started: time.Now()}
	s.task = t
	go func() {
		defer cancel()
		err := f(ctx, t)
		t.mu.Lock()
		t.running, t.ok = false, err == nil
		if err != nil {
			t.err = err.Error()
		}
		t.mu.Unlock()
		if s.OnChange != nil {
			s.OnChange()
		}
	}()
	return nil
}

func (s *Server) currentTask() *TaskView {
	s.taskMu.Lock()
	t := s.task
	s.taskMu.Unlock()
	if t == nil {
		return nil
	}
	v := t.view()
	return &v
}

// ---------- estado ----------

// SetupState es el estado de la instalación que enseña la página.
type SetupState struct {
	Version    string      `json:"version"`
	Configured bool        `json:"configurado"`
	Home       string      `json:"datos"`
	Game       GameState   `json:"juego"`
	Claude     ClaudeState `json:"claude"`
	GitHub     GitState    `json:"github"`
	Autostart  bool        `json:"arranque"`
	CanAuto    bool        `json:"puede_arrancar"`
	Task       *TaskView   `json:"tarea,omitempty"`
}

type GameState struct {
	Path    string       `json:"ruta"`
	Addon   bool         `json:"addon"`
	Data    bool         `json:"datos"` // el addon ya ha guardado algo
	Found   []FlavorView `json:"detectadas"`
	Choices []FlavorView `json:"opciones,omitempty"`
	Error   string       `json:"error,omitempty"`
}

type FlavorView struct {
	Name string `json:"nombre"`
	Path string `json:"ruta"`
}

type ClaudeState struct {
	Installed bool   `json:"instalado"`
	LoggedIn  bool   `json:"sesion"`
	Path      string `json:"ruta,omitempty"`
	Error     string `json:"error,omitempty"`
}

type GitState struct {
	Git     bool   `json:"git"`
	Publish bool   `json:"publicar"`
	URL     string `json:"url,omitempty"`
}

func flavorViews(list []wow.Flavor) []FlavorView {
	var out []FlavorView
	for _, f := range list {
		out = append(out, FlavorView{Name: f.Name, Path: f.Path})
	}
	return out
}

func (s *Server) setupState(ctx context.Context) SetupState {
	cfg := s.Runner.Config()
	st := SetupState{Version: s.Version, Home: filepath.Dir(s.ConfigPath), Task: s.currentTask()}
	if _, err := os.Stat(s.ConfigPath); err == nil && cfg.WoW != "" {
		st.Configured = true
	}
	st.Game.Path = cfg.WoW
	if cfg.WoW != "" {
		_, err := os.Stat(filepath.Join(wow.AddonDir(wow.Flavor{Path: cfg.WoW}), "Cronica.toc"))
		st.Game.Addon = err == nil
		if sv := s.Runner.SVPath(); sv != "" {
			_, err := os.Stat(sv)
			st.Game.Data = err == nil
		}
	} else {
		st.Game.Found = flavorViews(wow.Find())
	}

	path, found := narrate.FindClaude(cfg.Claude)
	st.Claude.Installed = found
	if found {
		st.Claude.Path = path
		if s.Fake {
			st.Claude.LoggedIn = true
		} else if err := (narrate.ClaudeCLI{Command: cfg.Claude}).CheckAuth(ctx); err == nil {
			st.Claude.LoggedIn = true
		} else if !errors.Is(err, narrate.ErrNotLoggedIn) {
			st.Claude.Error = err.Error()
		}
	}

	_, gerr := exec.LookPath("git")
	st.GitHub.Git = gerr == nil
	st.GitHub.Publish = cfg.Publish
	if cfg.Publish && st.GitHub.Git {
		cmd := exec.Command("git", "-C", cfg.Repo, "remote", "get-url", "origin")
		system.Hide(cmd)
		if out, err := cmd.Output(); err == nil {
			st.GitHub.URL = strings.TrimSpace(string(out))
		}
	}
	st.CanAuto = runtime.GOOS == "windows" || runtime.GOOS == "linux"
	st.Autostart = system.Installed()
	return st
}

func (s *Server) saveConfig(f func(*app.Config)) error {
	cfg := s.Runner.UpdateConfig(f)
	err := app.SaveConfig(s.ConfigPath, cfg)
	if s.OnChange != nil {
		go s.OnChange()
	}
	return err
}

// ---------- rutas ----------

func (s *Server) apiSetupState(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.setupState(r.Context()))
}

func readBody(r *http.Request, v any) error {
	return json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(v)
}

func (s *Server) apiSetupGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "usa POST")
		return
	}
	var req struct {
		Path string `json:"ruta"`
		Pick bool   `json:"elegir"`
	}
	if err := readBody(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "petición no válida")
		return
	}
	st := func(g GameState) {
		full := s.setupState(r.Context())
		if g.Error != "" || len(g.Choices) > 0 {
			full.Game.Error, full.Game.Choices = g.Error, g.Choices
		}
		writeJSON(w, full)
	}
	p := req.Path
	if req.Pick {
		var ok bool
		p, ok = system.PickFolder("Elige la carpeta de WoW Forever (la que tiene Interface y WTF dentro)")
		if !ok {
			st(GameState{Error: "No he podido abrir el selector de carpetas. Escribe la ruta a mano."})
			return
		}
		if p == "" { // cancelado
			st(GameState{})
			return
		}
	}
	f, list, err := wow.FromPath(p)
	if err != nil {
		st(GameState{Error: err.Error()})
		return
	}
	if len(list) > 0 {
		st(GameState{Choices: flavorViews(list)})
		return
	}
	dir, err := wow.InstallAddon(f, s.AddonFiles)
	if err != nil {
		st(GameState{Error: "No he podido instalar el addon: " + err.Error()})
		return
	}
	err = s.saveConfig(func(c *app.Config) {
		c.WoW, c.SavedVariables = f.Path, ""
		if acc := wow.Accounts(f); len(acc) == 1 {
			c.SavedVariables = wow.SavedVariablesPath(f, acc[0])
		}
		c.AddonTexts = filepath.Join(dir, "CronicaTextos.lua")
	})
	if err != nil {
		st(GameState{Error: "No he podido guardar la configuración: " + err.Error()})
		return
	}
	st(GameState{})
}

func (s *Server) apiSetupClaude(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "usa POST")
		return
	}
	var req struct {
		Action string `json:"accion"` // instalar | sesion
	}
	if err := readBody(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "petición no válida")
		return
	}
	cfg := s.Runner.Config()
	login := func(ctx context.Context, t *task) error {
		path, found := narrate.FindClaude(cfg.Claude)
		if !found {
			return errors.New("no encuentro Claude Code después de instalarlo")
		}
		t.say("Abriendo el navegador para que inicies sesión en tu cuenta de Claude…")
		if err := t.exec(ctx, path, "auth", "login"); err != nil {
			return fmt.Errorf("el inicio de sesión no ha terminado: %w", err)
		}
		if err := (narrate.ClaudeCLI{Command: cfg.Claude}).CheckAuth(ctx); err != nil {
			return err
		}
		t.say("✓ Sesión iniciada.")
		if s.OnClaudeReady != nil {
			go s.OnClaudeReady()
		}
		return nil
	}
	var err error
	switch req.Action {
	case "instalar":
		err = s.startTask("Instalar Claude Code", func(ctx context.Context, t *task) error {
			t.say("Descargando el instalador oficial de Anthropic (%s)…", narrate.InstallCommand())
			var ierr error
			if runtime.GOOS == "windows" {
				ierr = t.exec(ctx, "powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", narrate.InstallCommand())
			} else {
				ierr = t.exec(ctx, "bash", "-c", narrate.InstallCommand())
			}
			if ierr != nil {
				return fmt.Errorf("el instalador ha fallado: %w", ierr)
			}
			t.say("✓ Claude Code instalado.\n")
			return login(ctx, t)
		})
	case "sesion":
		err = s.startTask("Iniciar sesión en Claude", login)
	default:
		writeErr(w, http.StatusBadRequest, "acción desconocida")
		return
	}
	if err != nil {
		writeErr(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, s.setupState(r.Context()))
}

// apiSetupInput manda una línea (el código de inicio de sesión) al trabajo en marcha.
func (s *Server) apiSetupInput(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "usa POST")
		return
	}
	var req struct {
		Text string `json:"texto"`
	}
	if err := readBody(r, &req); err != nil || strings.TrimSpace(req.Text) == "" {
		writeErr(w, http.StatusBadRequest, "petición no válida")
		return
	}
	s.taskMu.Lock()
	t := s.task
	s.taskMu.Unlock()
	if t == nil {
		writeErr(w, http.StatusConflict, "no hay nada esperando")
		return
	}
	t.mu.Lock()
	in := t.stdin
	t.mu.Unlock()
	if in == nil {
		writeErr(w, http.StatusConflict, "no hay nada esperando")
		return
	}
	bw := bufio.NewWriter(in)
	bw.WriteString(strings.TrimSpace(req.Text) + "\n")
	if err := bw.Flush(); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	t.say("(código enviado)")
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) apiSetupTask(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{"tarea": s.currentTask()})
}

var reGitHub = regexp.MustCompile(`^https://github\.com/[\w.-]+/[\w.-]+?(\.git)?/?$`)

func (s *Server) apiSetupGitHub(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "usa POST")
		return
	}
	var req struct {
		URL    string `json:"url"`
		Remove bool   `json:"quitar"`
	}
	if err := readBody(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "petición no válida")
		return
	}
	home := filepath.Dir(s.ConfigPath)
	if req.Remove {
		if err := s.saveConfig(func(c *app.Config) { c.Publish, c.Repo = false, home }); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if s.PrepareWeb != nil {
			s.PrepareWeb(home)
		}
		writeJSON(w, s.setupState(r.Context()))
		return
	}
	url := strings.TrimSpace(req.URL)
	if !reGitHub.MatchString(url) {
		writeErr(w, http.StatusBadRequest, "Esa dirección no parece de un repositorio de GitHub (por ejemplo https://github.com/TuUsuario/mi-cronica).")
		return
	}
	if _, err := exec.LookPath("git"); err != nil {
		writeErr(w, http.StatusBadRequest, "Falta Git en este equipo.")
		return
	}
	err := s.startTask("Conectar con GitHub", func(ctx context.Context, t *task) error {
		dir := filepath.Join(home, "web")
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			t.say("Ya tenía una copia; la actualizo…")
			t.exec(ctx, "git", "-C", dir, "remote", "set-url", "origin", url)
			if err := t.exec(ctx, "git", "-C", dir, "pull", "--rebase"); err != nil {
				t.say("(no he podido actualizarla, sigo con la que hay)")
			}
		} else {
			t.say("Descargando %s…", url)
			if err := t.exec(ctx, "git", "clone", url, dir); err != nil {
				return fmt.Errorf("no he podido descargarlo (¿la dirección es correcta?): %w", err)
			}
		}
		t.say("\nComprobando que puedes subir cambios. Si es la primera vez, se abrirá una ventana de GitHub para que entres: Crónica no ve tu contraseña…")
		pushErr := t.exec(ctx, "git", "-C", dir, "push", "--dry-run")
		if s.PrepareWeb != nil {
			if err := s.PrepareWeb(dir); err != nil {
				t.say("No he podido preparar la web: %v", err)
			}
		}
		if err := s.saveConfig(func(c *app.Config) { c.Repo, c.Publish = dir, true }); err != nil {
			return err
		}
		if pushErr != nil {
			return fmt.Errorf("he descargado el repositorio pero no puedo subir cambios: revisa que es tuyo y que has entrado en GitHub. La crónica se guarda igualmente y se publicará cuando lo arregles")
		}
		t.say("✓ GitHub listo: cada relato nuevo se publicará solo.")
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, s.setupState(r.Context()))
}

func (s *Server) apiSetupAutostart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "usa POST")
		return
	}
	var req struct {
		On bool `json:"activo"`
	}
	if err := readBody(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "petición no válida")
		return
	}
	var err error
	if req.On {
		cfg := s.Runner.Config()
		if _, serr := os.Stat(s.ConfigPath); serr != nil {
			serr = app.SaveConfig(s.ConfigPath, cfg)
			err = serr
		}
		if err == nil {
			_, err = system.Install(system.Service{Exe: s.Exe, Config: s.ConfigPath, WorkDir: cfg.Repo, LogFile: system.DefaultLogFile()})
		}
	} else {
		err = system.RemoveAutostart()
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, s.setupState(r.Context()))
}
