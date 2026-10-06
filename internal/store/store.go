// Package store guarda la crónica: JSON para la web y un archivo Lua para el addon.
package store

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Nehonar/cronica-forever/internal/group"
	"github.com/Nehonar/cronica-forever/internal/luasv"
	"github.com/Nehonar/cronica-forever/internal/narrate"
)

// QuestRef es la referencia a una misión dentro de un relato.
type QuestRef struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Giver string `json:"giver,omitempty"`
}

// ItemRef es una pieza de equipo dentro de un relato.
type ItemRef struct {
	Name    string         `json:"name"`
	Quality int            `json:"quality"`
	Slot    string         `json:"slot,omitempty"`
	SubType string         `json:"subType,omitempty"`
	Armor   int            `json:"armor,omitempty"`
	Stats   map[string]int `json:"stats,omitempty"`
}

// Story es un relato guardado.
type Story struct {
	ID        string     `json:"id"`
	Kind      group.Kind `json:"kind"`
	Zone      string     `json:"zone"`
	Subzones  []string   `json:"subzones,omitempty"`
	Title     string     `json:"title"`
	Text      string     `json:"text"`
	Summary   string     `json:"summary"`
	Quests    []QuestRef `json:"quests,omitempty"`
	Items     []ItemRef  `json:"items,omitempty"`
	LevelFrom int        `json:"levelFrom,omitempty"`
	LevelTo   int        `json:"levelTo,omitempty"`
	Deaths    int        `json:"deaths,omitempty"`
	Faction   string     `json:"faction,omitempty"`  // hitos de reputación
	Standing  string     `json:"standing,omitempty"` // rango alcanzado
	Start     int64      `json:"start,omitempty"`
	End       int64      `json:"end,omitempty"`
	Created   string     `json:"created,omitempty"`
}

// LevelUp marca cuándo se alcanzó un nivel.
type LevelUp struct {
	Level  int   `json:"level"`
	T      int64 `json:"t,omitempty"`      // fecha (solo en la copia privada)
	Played int64 `json:"played,omitempty"` // tiempo jugado total al alcanzarlo
}

// Stats son los números del personaje.
type Stats struct {
	Level      int       `json:"level"`
	Played     int64     `json:"played"`
	QuestsDone int       `json:"questsDone"`
	Deaths     int       `json:"deaths"`
	Zones      []string  `json:"zones"`
	LevelUps   []LevelUp `json:"levelUps"`
	// Rango actual con cada facción (solo las que han subido a Amistoso o más).
	Reputation []RepStanding `json:"reputation,omitempty"`
	FirstSeen  int64         `json:"firstSeen,omitempty"`
	LastSeen   int64         `json:"lastSeen,omitempty"`
}

// RepStanding es el rango con una facción.
type RepStanding struct {
	Faction    string `json:"faction"`
	Standing   string `json:"standing"`
	StandingID int    `json:"standingID"`
}

// Pending es lo registrado que aún no se ha narrado.
type Pending struct {
	Loose      []string   `json:"loose"`
	OpenChains [][]string `json:"openChains"`
	// Relatos que esperan a que el personaje tenga trasfondo.
	AwaitingBackstory int `json:"awaitingBackstory,omitempty"`
}

// Encargo es una misión aceptada, narrada al momento para leerla en la app.
type Encargo struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	Giver      string `json:"giver,omitempty"`
	Zone       string `json:"zone,omitempty"`
	Subzone    string `json:"subzone,omitempty"`
	Objectives string `json:"objectives,omitempty"`
	Original   string `json:"original,omitempty"` // texto original de la misión
	Heading    string `json:"heading,omitempty"`  // título narrativo
	Text       string `json:"text,omitempty"`     // narración
	AcceptT    int64  `json:"acceptT,omitempty"`
	State      string `json:"state"` // activa, entregada, abandonada
	Narrated   string `json:"narrated,omitempty"`
}

// Doc es todo lo que la web sabe de un personaje.
type Doc struct {
	Key       string        `json:"key"`
	Character narrate.Sheet `json:"character"`
	Level     int           `json:"level"`
	Realm     string        `json:"realm,omitempty"`
	Updated   string        `json:"updated,omitempty"`
	Stories   []Story       `json:"stories"`
	Encargos  []Encargo     `json:"encargos,omitempty"`
	Pending   Pending       `json:"pending"`
	Stats     Stats         `json:"stats"`
}

// Has indica si ya existe un relato con ese id.
func (d *Doc) Has(id string) bool {
	for _, s := range d.Stories {
		if s.ID == id {
			return true
		}
	}
	return false
}

// Previous devuelve los n últimos relatos resumidos, para continuidad.
func (d *Doc) Previous(n int) []narrate.Previous {
	var out []narrate.Previous
	start := len(d.Stories) - n
	if start < 0 {
		start = 0
	}
	for _, s := range d.Stories[start:] {
		if s.Kind == group.Equip {
			continue
		}
		out = append(out, narrate.Previous{Title: s.Title, Summary: s.Summary})
	}
	return out
}

// Add añade un relato a partir de su grupo y su texto, manteniendo el orden temporal.
func (d *Doc) Add(g group.Group, st narrate.Story) {
	s := Story{
		ID: g.ID, Kind: g.Kind, Zone: g.Zone, Subzones: g.Subzones,
		Title: st.Title, Text: st.Text, Summary: st.Summary,
		LevelFrom: g.LevelFrom, LevelTo: g.LevelTo, Deaths: g.Deaths,
		Start: g.Start, End: g.End, Created: time.Now().UTC().Format(time.RFC3339),
	}
	for _, q := range g.Quests {
		s.Quests = append(s.Quests, QuestRef{ID: q.ID, Title: q.Title, Giver: q.Giver})
	}
	if g.Standing != nil {
		s.Faction, s.Standing = g.Standing.Faction, g.Standing.Standing
	}
	for _, it := range g.Items {
		s.Items = append(s.Items, ItemRef{Name: it.Item, Quality: it.Quality, Slot: it.Slot, SubType: it.ItemSubType, Armor: it.Armor, Stats: it.Stats})
	}
	d.Stories = append(d.Stories, s)
	sort.SliceStable(d.Stories, func(i, j int) bool { return d.Stories[i].End < d.Stories[j].End })
}

// Paths agrupa las rutas del repositorio.
type Paths struct {
	Repo string
}

// DataDir es la carpeta de datos de la web.
func (p Paths) DataDir() string { return filepath.Join(p.Repo, "docs", "data") }

// SheetsDir es la carpeta de fichas de personaje.
func (p Paths) SheetsDir() string { return filepath.Join(p.Repo, "personajes") }

// PrivateDir guarda la copia completa de cada personaje, con fechas y horas.
// No se publica nunca: Publish solo sube docs/ y personajes/.
func (p Paths) PrivateDir() string { return filepath.Join(p.Repo, "privado") }

// LoadDoc lee el documento de un personaje; si no existe devuelve uno vacío.
func (p Paths) LoadDoc(key string) (*Doc, error) {
	d := &Doc{Key: key}
	b, err := os.ReadFile(filepath.Join(p.PrivateDir(), key+".json"))
	if errors.Is(err, fs.ErrNotExist) { // versiones anteriores solo tenían la copia pública
		b, err = os.ReadFile(filepath.Join(p.DataDir(), key+".json"))
	}
	if errors.Is(err, fs.ErrNotExist) {
		return d, nil
	}
	if err != nil {
		return nil, err
	}
	return d, json.Unmarshal(b, d)
}

// SaveDoc escribe el documento de un personaje: la copia completa en privado/
// y, para la web, una copia sin fechas ni horas (para que nadie pueda saber
// cuándo juegas).
func (p Paths) SaveDoc(d *Doc) error {
	d.Updated = time.Now().UTC().Format(time.RFC3339)
	if err := writeJSON(filepath.Join(p.PrivateDir(), d.Key+".json"), d); err != nil {
		return err
	}
	return writeJSON(filepath.Join(p.DataDir(), d.Key+".json"), Public(d))
}

// Public es la copia de un documento que se puede publicar: sin fechas, horas
// ni nada que permita saber cuándo se ha jugado.
func Public(d *Doc) *Doc {
	c := *d
	c.Updated = ""
	c.Stories = make([]Story, len(d.Stories))
	for i, s := range d.Stories {
		s.Start, s.End, s.Created = 0, 0, ""
		s.ID = publicID(s.ID)
		c.Stories[i] = s
	}
	c.Encargos = make([]Encargo, len(d.Encargos))
	for i, e := range d.Encargos {
		e.AcceptT, e.Narrated = 0, ""
		c.Encargos[i] = e
	}
	c.Stats.FirstSeen, c.Stats.LastSeen = 0, 0
	c.Stats.LevelUps = make([]LevelUp, len(d.Stats.LevelUps))
	for i, u := range d.Stats.LevelUps {
		u.T = 0
		c.Stats.LevelUps[i] = u
	}
	return &c
}

// publicID cambia el identificador de un relato por otro que no lleve la hora
// dentro (los de equipo la llevaban).
func publicID(id string) string {
	h := sha1.Sum([]byte(id))
	return hex.EncodeToString(h[:6])
}

// LoadSheet lee la ficha de un personaje. Si no existe, crea una ficha básica
// marcada como nueva para que el jugador escriba su trasfondo.
func (p Paths) LoadSheet(key, name, race, class string) (narrate.Sheet, bool, error) {
	path := filepath.Join(p.SheetsDir(), key+".json")
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		s := narrate.Sheet{Key: key, Name: name, Race: race, Class: class}
		// Si ya hay una ficha con trasfondo para un personaje del mismo nombre
		// (por ejemplo, la de demostración), se parte de ella.
		if base, ok := p.findSheetByName(name); ok {
			base.Key = key
			s = base
		}
		return s, true, writeJSON(path, s)
	}
	if err != nil {
		return narrate.Sheet{}, false, err
	}
	var s narrate.Sheet
	if err := json.Unmarshal(b, &s); err != nil {
		return s, false, fmt.Errorf("%s: %w", path, err)
	}
	if s.Key == "" {
		s.Key = key
	}
	return s, false, nil
}

func (p Paths) findSheetByName(name string) (narrate.Sheet, bool) {
	if name == "" {
		return narrate.Sheet{}, false
	}
	files, _ := filepath.Glob(filepath.Join(p.SheetsDir(), name+"-*.json"))
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var s narrate.Sheet
		if json.Unmarshal(b, &s) == nil && s.Backstory != "" {
			return s, true
		}
	}
	return narrate.Sheet{}, false
}

// IndexEntry es un personaje en el índice de la web.
type IndexEntry struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	Epithet string `json:"epithet,omitempty"`
	Race    string `json:"race"`
	Class   string `json:"class"`
	Level   int    `json:"level"`
	Stories int    `json:"stories"`
	Updated string `json:"-"` // solo para ordenar; no se publica
	New     bool   `json:"new,omitempty"`
}

// SaveIndex reescribe docs/data/index.json con todos los personajes que haya.
func (p Paths) SaveIndex() error {
	entries, _ := os.ReadDir(p.DataDir())
	var idx []IndexEntry
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" || e.Name() == "index.json" {
			continue
		}
		key := e.Name()[:len(e.Name())-5]
		d, err := p.LoadDoc(key)
		if err != nil {
			continue
		}
		idx = append(idx, IndexEntry{
			Key: d.Key, Name: d.Character.Name, Epithet: d.Character.Epithet,
			Race: d.Character.Race, Class: d.Character.Class, Level: d.Level,
			Stories: len(d.Stories), Updated: d.Updated, New: d.Character.Backstory == "",
		})
	}
	sort.Slice(idx, func(i, j int) bool { return idx[i].Updated > idx[j].Updated })
	// «updated» no es una hora: es una huella del contenido, para que la web sepa
	// si hay algo nuevo sin revelar cuándo se ha jugado.
	h := sha1.New()
	for _, e := range idx {
		b, _ := os.ReadFile(filepath.Join(p.DataDir(), e.Key+".json"))
		h.Write(b)
	}
	return writeJSON(filepath.Join(p.DataDir(), "index.json"), map[string]any{
		"updated":    hex.EncodeToString(h.Sum(nil))[:16],
		"characters": idx,
	})
}

// WriteAddonTexts escribe CronicaTextos.lua, que el addon carga al iniciar o con /reload.
// Incluye los últimos `limit` relatos de cada personaje.
// Status es el estado del cronista que el addon puede mostrar en el juego.
type Status struct {
	OK      bool
	Message string
	T       int64
}

func WriteAddonTexts(path string, docs []*Doc, limit int, st Status, phrases map[string]Phrases) error {
	all := map[string]any{}
	for _, d := range docs {
		start := len(d.Stories) - limit
		if start < 0 {
			start = 0
		}
		var list []any
		for _, s := range d.Stories[start:] {
			list = append(list, map[string]any{
				"id": s.ID, "kind": string(s.Kind), "zone": s.Zone,
				"title": s.Title, "text": s.Text, "t": s.End,
			})
		}
		all[d.Key] = list
	}
	frases := map[string]any{}
	for k, ph := range phrases {
		cats := map[string]any{}
		for cat, list := range ph {
			arr := make([]any, len(list))
			for i, x := range list {
				arr[i] = x
			}
			cats[cat] = arr
		}
		frases[k] = cats
	}
	estado := map[string]any{"ok": st.OK, "mensaje": st.Message, "t": st.T}
	src := "-- Generado por Crónica. No editar a mano.\nCronicaTextos = " + luasv.Encode(all, 0) + "\n" +
		"CronicaFrases = " + luasv.Encode(frases, 0) + "\n" +
		"CronicaEstado = " + luasv.Encode(estado, 0) + "\n"
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(src), 0o644)
}

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// SaveSheet guarda la ficha de un personaje (personajes/<clave>.json).
func (p Paths) SaveSheet(s narrate.Sheet) error {
	if s.Key == "" {
		return fmt.Errorf("la ficha no tiene clave de personaje")
	}
	return writeJSON(filepath.Join(p.SheetsDir(), s.Key+".json"), s)
}

// SyncEncargos actualiza la lista de misiones aceptadas a partir de los eventos:
// añade las nuevas y marca las entregadas o abandonadas. Devuelve las activas sin narrar.
func (d *Doc) SyncEncargos(accepts, turnins, abandons map[int64]Encargo) []int {
	idx := map[int64]int{}
	for i, e := range d.Encargos {
		idx[e.ID] = i
	}
	for id, a := range accepts {
		if i, ok := idx[id]; ok {
			// Si se volvió a aceptar después de abandonarla, vuelve a estar activa.
			if a.AcceptT > d.Encargos[i].AcceptT {
				d.Encargos[i].AcceptT, d.Encargos[i].State = a.AcceptT, "activa"
			}
			continue
		}
		a.State = "activa"
		d.Encargos = append(d.Encargos, a)
		idx[id] = len(d.Encargos) - 1
	}
	for id, t := range turnins {
		if i, ok := idx[id]; ok && t.AcceptT >= d.Encargos[i].AcceptT {
			d.Encargos[i].State = "entregada"
		}
	}
	for id, t := range abandons {
		if i, ok := idx[id]; ok && t.AcceptT >= d.Encargos[i].AcceptT && d.Encargos[i].State == "activa" {
			d.Encargos[i].State = "abandonada"
		}
	}
	sort.SliceStable(d.Encargos, func(i, j int) bool { return d.Encargos[i].AcceptT > d.Encargos[j].AcceptT })
	var todo []int
	for i, e := range d.Encargos {
		if e.State == "activa" && e.Text == "" {
			todo = append(todo, i)
		}
	}
	return todo
}

// Phrases son las frases que el personaje piensa o dice, por categoría.
type Phrases map[string][]string

// LoadPhrases lee personajes/<clave>.frases.json; ok=false si no existe o está desfasada
// respecto a la ficha (se ha cambiado el trasfondo después).
func (p Paths) LoadPhrases(key string) (Phrases, bool) {
	path := filepath.Join(p.SheetsDir(), key+".frases.json")
	info, err := os.Stat(path)
	if err != nil {
		return nil, false
	}
	if sheet, err := os.Stat(filepath.Join(p.SheetsDir(), key+".json")); err == nil && sheet.ModTime().After(info.ModTime()) {
		return nil, false
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var ph Phrases
	if json.Unmarshal(b, &ph) != nil || len(ph) == 0 {
		return nil, false
	}
	return ph, true
}

// SavePhrases guarda la baraja de frases de un personaje.
func (p Paths) SavePhrases(key string, ph Phrases) error {
	return writeJSON(filepath.Join(p.SheetsDir(), key+".frases.json"), ph)
}

// ---------- Borrar ----------

func (p Paths) deletedFile() string { return filepath.Join(p.PrivateDir(), "borrados.json") }

// DeletedAt devuelve desde cuándo vale la crónica de un personaje (0 = nunca se
// borró). Lo anterior a esa fecha no se vuelve a narrar aunque siga en el juego.
func (p Paths) DeletedAt(key string) int64 {
	var m map[string]int64
	if b, err := os.ReadFile(p.deletedFile()); err == nil {
		json.Unmarshal(b, &m)
	}
	return m[key]
}

// DeleteCharacter borra la crónica de un personaje: sus relatos y misiones y,
// si all, también su historia y sus frases. Lo jugado hasta ahora no se vuelve a
// narrar; si sigues jugándolo, el cronista empieza desde este momento.
func (p Paths) DeleteCharacter(key string, all bool, now int64) error {
	if key == "" || strings.ContainsAny(key, `/\`) || strings.Contains(key, "..") {
		return fmt.Errorf("personaje no válido")
	}
	files := []string{filepath.Join(p.DataDir(), key+".json"), filepath.Join(p.PrivateDir(), key+".json")}
	if all {
		files = append(files, filepath.Join(p.SheetsDir(), key+".json"), filepath.Join(p.SheetsDir(), key+".frases.json"))
	}
	for _, f := range files {
		if err := os.Remove(f); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	m := map[string]int64{}
	if b, err := os.ReadFile(p.deletedFile()); err == nil {
		json.Unmarshal(b, &m)
	}
	m[key] = now
	if err := writeJSON(p.deletedFile(), m); err != nil {
		return err
	}
	return p.SaveIndex()
}
