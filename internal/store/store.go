// Package store guarda la crónica: JSON para la web y un archivo Lua para el addon.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
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
	Name    string `json:"name"`
	Quality int    `json:"quality"`
	Slot    string `json:"slot,omitempty"`
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
	Start     int64      `json:"start"`
	End       int64      `json:"end"`
	Created   string     `json:"created"`
}

// LevelUp marca cuándo se alcanzó un nivel.
type LevelUp struct {
	Level int   `json:"level"`
	T     int64 `json:"t"`
}

// Stats son los números del personaje.
type Stats struct {
	Level      int       `json:"level"`
	Played     int64     `json:"played"`
	QuestsDone int       `json:"questsDone"`
	Deaths     int       `json:"deaths"`
	Zones      []string  `json:"zones"`
	LevelUps   []LevelUp `json:"levelUps"`
	FirstSeen  int64     `json:"firstSeen,omitempty"`
	LastSeen   int64     `json:"lastSeen,omitempty"`
}

// Pending es lo registrado que aún no se ha narrado.
type Pending struct {
	Loose      []string   `json:"loose"`
	OpenChains [][]string `json:"openChains"`
}

// Doc es todo lo que la web sabe de un personaje.
type Doc struct {
	Key       string        `json:"key"`
	Character narrate.Sheet `json:"character"`
	Level     int           `json:"level"`
	Realm     string        `json:"realm,omitempty"`
	Updated   string        `json:"updated"`
	Stories   []Story       `json:"stories"`
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
	for _, it := range g.Items {
		s.Items = append(s.Items, ItemRef{Name: it.Item, Quality: it.Quality, Slot: it.Slot})
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

// LoadDoc lee el documento de un personaje; si no existe devuelve uno vacío.
func (p Paths) LoadDoc(key string) (*Doc, error) {
	d := &Doc{Key: key}
	b, err := os.ReadFile(filepath.Join(p.DataDir(), key+".json"))
	if errors.Is(err, fs.ErrNotExist) {
		return d, nil
	}
	if err != nil {
		return nil, err
	}
	return d, json.Unmarshal(b, d)
}

// SaveDoc escribe el documento de un personaje.
func (p Paths) SaveDoc(d *Doc) error {
	d.Updated = time.Now().UTC().Format(time.RFC3339)
	return writeJSON(filepath.Join(p.DataDir(), d.Key+".json"), d)
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
	Updated string `json:"updated"`
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
	return writeJSON(filepath.Join(p.DataDir(), "index.json"), map[string]any{
		"updated":    time.Now().UTC().Format(time.RFC3339),
		"characters": idx,
	})
}

// WriteAddonTexts escribe CronicaTextos.lua, que el addon carga al iniciar o con /reload.
// Incluye los últimos `limit` relatos de cada personaje.
func WriteAddonTexts(path string, docs []*Doc, limit int) error {
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
	src := "-- Generado por Crónica. No editar a mano.\nCronicaTextos = " + luasv.Encode(all, 0) + "\n"
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
