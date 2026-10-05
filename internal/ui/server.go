// Package ui sirve en local (solo en este equipo) la web de la crónica y el
// chat con el cronista para crear el trasfondo de los personajes.
package ui

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Nehonar/cronica-forever/internal/app"
	"github.com/Nehonar/cronica-forever/internal/interview"
	"github.com/Nehonar/cronica-forever/internal/model"
	"github.com/Nehonar/cronica-forever/internal/narrate"
	"github.com/Nehonar/cronica-forever/internal/store"
)

//go:embed web/*
var webFiles embed.FS

// Server es el servidor local de Crónica.
type Server struct {
	Runner   *app.Runner
	Narrator narrate.Narrator
	Port     int

	// Para la página «Preparar Crónica»:
	ConfigPath    string // cronica.json
	Version       string // versión del programa
	Exe           string // ruta del programa (para el arranque automático)
	AddonFiles    fs.FS  // archivos del addon (carpeta addon/Cronica)
	PrepareWeb    func(repo string) error
	OnChange      func() // la configuración o Claude han cambiado
	OnClaudeReady func() // Claude acaba de quedar con la sesión iniciada
	Fake          bool   // modo de prueba: no comprueba Claude

	addr   string
	srv    *http.Server
	taskMu sync.Mutex
	task   *task
}

// URL devuelve la dirección base del servidor una vez arrancado.
func (s *Server) URL() string { return "http://" + s.addr + "/" }

// Start arranca el servidor en 127.0.0.1 (no es accesible desde otros equipos).
func (s *Server) Start() error {
	port := s.Port
	if port == 0 {
		port = 8737
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		ln, err = net.Listen("tcp", "127.0.0.1:0") // puerto ocupado: cualquiera libre
		if err != nil {
			return err
		}
	}
	s.addr = ln.Addr().String()
	s.srv = &http.Server{Handler: s.Handler(), ReadHeaderTimeout: 10 * time.Second}
	go s.srv.Serve(ln)
	return nil
}

// Stop cierra el servidor.
func (s *Server) Stop() {
	if s.srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		s.srv.Shutdown(ctx)
	}
}

// Handler monta las rutas.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	sub, _ := fs.Sub(webFiles, "web")
	files := http.FileServer(http.FS(sub))
	for _, page := range []string{"/personaje", "/misiones", "/preparar"} {
		page := page
		mux.Handle(page+"/", files)
		mux.HandleFunc(page, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, page+"/"+query(r), http.StatusFound)
		})
	}
	mux.HandleFunc("/api/misiones", s.apiQuests)
	mux.HandleFunc("/api/personajes", s.apiCharacters)
	mux.HandleFunc("/api/entrevista", s.apiInterview)
	mux.HandleFunc("/api/guardar", s.apiSave)
	mux.HandleFunc("/api/hola", s.apiHello)
	mux.HandleFunc("/api/preparar/estado", s.apiSetupState)
	mux.HandleFunc("/api/preparar/juego", s.apiSetupGame)
	mux.HandleFunc("/api/preparar/claude", s.apiSetupClaude)
	mux.HandleFunc("/api/preparar/codigo", s.apiSetupInput)
	mux.HandleFunc("/api/preparar/tarea", s.apiSetupTask)
	mux.HandleFunc("/api/preparar/github", s.apiSetupGitHub)
	mux.HandleFunc("/api/preparar/arranque", s.apiSetupAutostart)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.FileServer(http.Dir(filepath.Join(s.Runner.Config().Repo, "docs"))).ServeHTTP(w, r)
	})
	return s.guard(mux)
}

// Hello lo usa una segunda copia del programa para saber que Crónica ya está abierta.
type Hello struct {
	App  string `json:"app"`
	Page string `json:"pagina"` // qué abrir si vuelves a hacer doble clic
}

func (s *Server) apiHello(w http.ResponseWriter, r *http.Request) {
	h := Hello{App: "cronica", Page: "/"}
	if cfg := s.Runner.Config(); cfg.WoW == "" && cfg.SavedVariables == "" {
		h.Page = "/preparar/"
	}
	writeJSON(w, h)
}

func query(r *http.Request) string {
	if r.URL.RawQuery == "" {
		return ""
	}
	return "?" + r.URL.RawQuery
}

// guard impide que otras webs abiertas en el navegador usen este servidor.
func (s *Server) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		if host != "127.0.0.1" && host != "localhost" {
			http.Error(w, "solo accesible desde este equipo", http.StatusForbidden)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") && r.Method == http.MethodPost {
			if r.Header.Get("X-Cronica") != "1" || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
				http.Error(w, "petición no válida", http.StatusForbidden)
				return
			}
		}
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// CharacterView es un personaje para la página del chat.
type CharacterView struct {
	app.CharacterInfo
	New   bool     `json:"new"`
	Title string   `json:"title,omitempty"` // nombre de la ficha, si existe
	Zones []string `json:"zones,omitempty"`
}

func (s *Server) characters() ([]CharacterView, map[string]model.Character, error) {
	chars, err := model.LoadSavedVariables(s.Runner.SVPath())
	if err != nil {
		return nil, nil, err
	}
	paths := store.Paths{Repo: s.Runner.Config().Repo}
	byKey := map[string]model.Character{}
	var out []CharacterView
	for _, c := range chars {
		byKey[c.Key] = c
		sheet, _, err := paths.LoadSheet(c.Key, c.Name, c.Race, c.Class)
		if err != nil {
			return nil, nil, err
		}
		doc, _ := paths.LoadDoc(c.Key)
		v := CharacterView{
			CharacterInfo: app.CharacterInfo{Key: c.Key, Name: c.Name, Race: c.Race, Class: c.Class, Level: c.Level},
			New:           sheet.Backstory == "",
			Title:         sheet.Name,
		}
		if doc != nil {
			v.Waiting = doc.Pending.AwaitingBackstory
			v.Zones = doc.Stats.Zones
		}
		out = append(out, v)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].New && !out[j].New })
	return out, byKey, nil
}

func (s *Server) apiCharacters(w http.ResponseWriter, r *http.Request) {
	list, _, err := s.characters()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "No puedo leer los datos del addon: "+err.Error())
		return
	}
	writeJSON(w, map[string]any{"characters": list})
}

func (s *Server) facts(key string) (interview.Facts, bool) {
	_, byKey, err := s.characters()
	if err != nil {
		return interview.Facts{}, false
	}
	c, ok := byKey[key]
	if !ok {
		return interview.Facts{}, false
	}
	f := interview.Facts{Key: key, Name: c.Name, Race: c.Race, Class: c.Class, Level: c.Level}
	seenZone := map[string]bool{}
	for _, e := range c.Events {
		if e.Zone != "" && !seenZone[e.Zone] {
			seenZone[e.Zone] = true
			f.Zones = append(f.Zones, e.Zone)
		}
		if e.Type == model.EvQuestTurnin {
			f.QuestsDone++
			if len(f.Quests) < 6 && e.Title != "" {
				f.Quests = append(f.Quests, e.Title)
			}
		}
	}
	return f, true
}

func (s *Server) apiInterview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "usa POST")
		return
	}
	var req struct {
		Key     string              `json:"key"`
		History []interview.Message `json:"history"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10)).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "petición no válida")
		return
	}
	f, ok := s.facts(req.Key)
	if !ok {
		writeErr(w, http.StatusNotFound, "No conozco a ese personaje.")
		return
	}
	reply, err := interview.Turn(r.Context(), s.Narrator, f, req.History)
	if err != nil {
		msg := "El cronista no ha podido responder: " + err.Error()
		if errors.Is(err, narrate.ErrNotLoggedIn) || strings.Contains(err.Error(), "claude") {
			msg += ". Comprueba en el icono de Crónica que Claude tiene la sesión iniciada."
		}
		writeErr(w, http.StatusBadGateway, msg)
		return
	}
	writeJSON(w, reply)
}

func (s *Server) apiSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "usa POST")
		return
	}
	var req struct {
		Key   string        `json:"key"`
		Sheet narrate.Sheet `json:"sheet"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10)).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "petición no válida")
		return
	}
	f, ok := s.facts(req.Key)
	if !ok {
		writeErr(w, http.StatusNotFound, "No conozco a ese personaje.")
		return
	}
	sh := req.Sheet
	sh.Key, sh.Race, sh.Class = req.Key, f.Race, f.Class
	if strings.TrimSpace(sh.Backstory) == "" {
		writeErr(w, http.StatusBadRequest, "La ficha no tiene trasfondo.")
		return
	}
	if sh.Name == "" {
		sh.Name = f.Name
	}
	if err := (store.Paths{Repo: s.Runner.Config().Repo}).SaveSheet(sh); err != nil {
		writeErr(w, http.StatusInternalServerError, "No he podido guardar la ficha: "+err.Error())
		return
	}
	// Narrar ya lo que estaba esperando a esta historia.
	go s.Runner.Process(context.Background())
	writeJSON(w, map[string]any{"ok": true, "key": req.Key})
}

// apiQuests devuelve las misiones aceptadas de un personaje (o del más reciente).
func (s *Server) apiQuests(w http.ResponseWriter, r *http.Request) {
	list, _, err := s.characters()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "No puedo leer los datos del addon: "+err.Error())
		return
	}
	paths := store.Paths{Repo: s.Runner.Config().Repo}
	key := r.URL.Query().Get("p")
	var best *store.Doc
	for _, c := range list {
		d, err := paths.LoadDoc(c.Key)
		if err != nil {
			continue
		}
		if c.Key == key {
			best = d
			break
		}
		if key == "" && (best == nil || latest(d) > latest(best)) {
			best = d
		}
	}
	resp := map[string]any{"characters": list}
	if best != nil {
		name := best.Character.Name
		if name == "" {
			name = best.Key
		}
		resp["key"], resp["name"], resp["encargos"] = best.Key, name, best.Encargos
	}
	writeJSON(w, resp)
}

func latest(d *store.Doc) int64 {
	var t int64
	for _, e := range d.Encargos {
		if e.AcceptT > t {
			t = e.AcceptT
		}
	}
	return t
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
