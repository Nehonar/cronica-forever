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
	"os"
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

	evMu    sync.Mutex
	clients []chan string // páginas abiertas, la última al final
}

// Navigate lleva la última página de Crónica abierta en el navegador a path.
// Devuelve false si no hay ninguna abierta.
func (s *Server) Navigate(path string) bool {
	s.evMu.Lock()
	defer s.evMu.Unlock()
	if len(s.clients) == 0 {
		return false
	}
	b, _ := json.Marshal(map[string]string{"ruta": path})
	select {
	case s.clients[len(s.clients)-1] <- string(b):
		return true
	default:
		return false
	}
}

// apiEvents mantiene abierta una conexión con cada página (Server-Sent Events).
func (s *Server) apiEvents(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "sin streaming", http.StatusInternalServerError)
		return
	}
	ch := make(chan string, 4)
	s.evMu.Lock()
	s.clients = append(s.clients, ch)
	s.evMu.Unlock()
	defer func() {
		s.evMu.Lock()
		for i, c := range s.clients {
			if c == ch {
				s.clients = append(s.clients[:i], s.clients[i+1:]...)
				break
			}
		}
		s.evMu.Unlock()
	}()
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, ": hola\n\n")
	fl.Flush()
	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case msg := <-ch:
			fmt.Fprintf(w, "event: ir\ndata: %s\n\n", msg)
			fl.Flush()
		case <-ping.C:
			fmt.Fprint(w, ": ping\n\n")
			fl.Flush()
		}
	}
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
	mux.Handle("/app/", files)
	mux.HandleFunc("/api/eventos", s.apiEvents)
	mux.HandleFunc("/api/misiones", s.apiQuests)
	mux.HandleFunc("/api/personajes", s.apiCharacters)
	mux.HandleFunc("/api/entrevista", s.apiInterview)
	mux.HandleFunc("/api/guardar", s.apiSave)
	mux.HandleFunc("/api/borrar", s.apiDelete)
	mux.HandleFunc("/api/relatos", s.apiStories)
	mux.HandleFunc("/api/relatos/quitar", s.apiRemoveStory)
	mux.HandleFunc("/api/hola", s.apiHello)
	mux.HandleFunc("/api/preparar/estado", s.apiSetupState)
	mux.HandleFunc("/api/preparar/juego", s.apiSetupGame)
	mux.HandleFunc("/api/preparar/claude", s.apiSetupClaude)
	mux.HandleFunc("/api/preparar/codigo", s.apiSetupInput)
	mux.HandleFunc("/api/preparar/tarea", s.apiSetupTask)
	mux.HandleFunc("/api/preparar/github", s.apiSetupGitHub)
	mux.HandleFunc("/api/preparar/arranque", s.apiSetupAutostart)
	mux.HandleFunc("/api/preparar/publicar", s.apiSetupPublish)
	mux.HandleFunc("/api/preparar/borrar-todo", s.apiDeleteAll)
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
	// OnlyChronicle: está en la crónica pero no en los datos del juego.
	OnlyChronicle bool `json:"solo_cronica,omitempty"`
}

func (s *Server) characters() ([]CharacterView, map[string]model.Character, error) {
	chars, svErr := model.LoadSavedVariables(s.Runner.SVPath())
	paths := store.Paths{Repo: s.Runner.Config().Repo}
	byKey := map[string]model.Character{}
	var out []CharacterView
	for _, c := range chars {
		byKey[c.Key] = c
		if paths.MergedInto(c.Key) != "" {
			continue
		}
		sheet, _, err := paths.LoadSheet(c.Key, c.FullName(), c.Race, c.Class)
		if err != nil {
			return nil, nil, err
		}
		doc, _ := paths.LoadDoc(c.Key)
		v := CharacterView{
			CharacterInfo: app.CharacterInfo{Key: c.Key, Name: c.FullName(), Race: c.Race, Class: c.Class, Level: c.Level},
			New:           sheet.Backstory == "",
			Title:         sheet.Name,
		}
		if doc != nil {
			v.Waiting = doc.Pending.AwaitingBackstory
			v.Zones = doc.Stats.Zones
		}
		out = append(out, v)
	}
	// Personajes que solo están en la crónica (por ejemplo, el de prueba).
	if entries, err := os.ReadDir(paths.DataDir()); err == nil {
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || filepath.Ext(name) != ".json" || name == "index.json" {
				continue
			}
			key := strings.TrimSuffix(name, ".json")
			if _, ok := byKey[key]; ok {
				continue
			}
			doc, err := paths.LoadDoc(key)
			if err != nil {
				continue
			}
			out = append(out, CharacterView{
				CharacterInfo: app.CharacterInfo{Key: key, Name: doc.Character.Name, Race: doc.Character.Race, Class: doc.Character.Class, Level: doc.Level},
				Title:         doc.Character.Name,
				OnlyChronicle: true,
			})
		}
	}
	if len(out) == 0 && svErr != nil {
		return nil, nil, svErr
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
	f := interview.Facts{Key: key, Name: c.FullName(), Race: c.Race, Class: c.Class, Level: c.Level}
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
	// Narrar ya lo que estaba esperando a esta historia y publicarla.
	go func() {
		s.Runner.Process(context.Background())
		s.Runner.PublishNow("Crónica: historia de "+sh.Name, nil)
	}()
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

// apiDelete borra la crónica de un personaje (solo relatos, o todo).
func (s *Server) apiDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "usa POST")
		return
	}
	var req struct {
		Key string `json:"key"`
		All bool   `json:"todo"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&req); err != nil || req.Key == "" {
		writeErr(w, http.StatusBadRequest, "petición no válida")
		return
	}
	paths := store.Paths{Repo: s.Runner.Config().Repo}
	err := s.Runner.Exclusive(func() error { return paths.DeleteCharacter(req.Key, req.All, time.Now().Unix()) })
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "No he podido borrarla: "+err.Error())
		return
	}
	go func() {
		s.Runner.Process(context.Background())
		s.Runner.PublishNow("Crónica: borrada la crónica de "+req.Key, nil)
	}()
	writeJSON(w, map[string]any{"ok": true})
}

// StoryView es un relato en la lista de Personajes.
type StoryView struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Kind      string `json:"kind"`
	Zone      string `json:"zone,omitempty"`
	LevelFrom int    `json:"levelFrom,omitempty"`
	LevelTo   int    `json:"levelTo,omitempty"`
}

func (s *Server) apiStories(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("p")
	doc, err := (store.Paths{Repo: s.Runner.Config().Repo}).LoadDoc(key)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := []StoryView{}
	for _, st := range doc.Stories {
		out = append(out, StoryView{ID: st.ID, Title: st.Title, Kind: string(st.Kind), Zone: st.Zone, LevelFrom: st.LevelFrom, LevelTo: st.LevelTo})
	}
	writeJSON(w, map[string]any{"stories": out})
}

func (s *Server) apiRemoveStory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "usa POST")
		return
	}
	var req struct {
		Key string `json:"key"`
		ID  string `json:"id"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&req); err != nil || req.Key == "" || req.ID == "" {
		writeErr(w, http.StatusBadRequest, "petición no válida")
		return
	}
	paths := store.Paths{Repo: s.Runner.Config().Repo}
	if err := s.Runner.Exclusive(func() error { return paths.RemoveStory(req.Key, req.ID) }); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	go s.Runner.PublishNow("Crónica: relato quitado", nil)
	writeJSON(w, map[string]bool{"ok": true})
}
