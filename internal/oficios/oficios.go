// Package oficios convierte lo que guarda el addon sobre bolsas, banco y
// profesiones en el JSON que usa la app «Banco y oficios» (docs/banco/).
// Solo cantidades, niveles y nombres: nada de fechas ni horas.
package oficios

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/Nehonar/cronica-forever/internal/luasv"
)

// Stack es un objeto y cuántos hay.
type Stack struct {
	ID int `json:"id"`
	N  int `json:"n"`
}

// Skill es el nivel de una profesión.
type Skill struct {
	Level int `json:"nivel"`
	Max   int `json:"max"`
}

// Mat es un material de una receta.
type Mat struct {
	ID   int    `json:"id,omitempty"`
	Name string `json:"nombre,omitempty"`
	N    int    `json:"n"`
}

// Known es una receta que el personaje sabe hacer, con su color de ahora.
type Known struct {
	ID    int    `json:"id,omitempty"`
	Name  string `json:"nombre"`
	Color string `json:"color,omitempty"`
	Mats  []Mat  `json:"mats,omitempty"`
}

// Teach es algo que enseña un instructor.
type Teach struct {
	ID    int    `json:"id,omitempty"`
	Name  string `json:"nombre"`
	Rank  string `json:"rango,omitempty"`
	State string `json:"estado,omitempty"` // available, unavailable, used
	Level int    `json:"nivel,omitempty"`
	Price int    `json:"precio,omitempty"` // en cobre
}

// Data es lo que la app sabe de un personaje.
type Data struct {
	Key       string             `json:"key"`
	Name      string             `json:"nombre"`
	Class     string             `json:"clase,omitempty"`
	Level     int                `json:"nivel,omitempty"`
	Money     int                `json:"dinero,omitempty"`
	Bags      []Stack            `json:"bolsas"`
	Bank      []Stack            `json:"banco"`
	BankSeen  bool               `json:"bancoVisto"`
	Skills    map[string]Skill   `json:"profesiones"`
	Known     map[string][]Known `json:"recetas"`
	Trainer   map[string][]Teach `json:"instructor"`
	HasSkills bool               `json:"-"`
}

// each recorre una tabla Lua con claves numéricas (en Array o en Hash).
func each(t *luasv.Table, f func(k int, v any)) {
	if t == nil {
		return
	}
	for i, v := range t.Array {
		if v != nil {
			f(i+1, v)
		}
	}
	for k, v := range t.Hash {
		if n, err := strconv.Atoi(k); err == nil {
			f(n, v)
		}
	}
}

func list(t *luasv.Table) []*luasv.Table {
	var out []*luasv.Table
	each(t, func(_ int, v any) {
		if x, ok := v.(*luasv.Table); ok {
			out = append(out, x)
		}
	})
	return out
}

func stacks(t *luasv.Table) []Stack {
	out := []Stack{}
	each(t, func(k int, v any) {
		if n, ok := v.(float64); ok && n > 0 {
			out = append(out, Stack{ID: k, N: int(n)})
		}
	})
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Parse lee la parte «oficios» de un personaje. Devuelve false si el addon aún
// no ha guardado nada.
func Parse(ct *luasv.Table) (Data, bool) {
	o := ct.Table("oficios")
	if o == nil {
		return Data{}, false
	}
	d := Data{
		Money: int(o.Int("dinero")), Bags: stacks(o.Table("bolsas")), Bank: stacks(o.Table("banco")),
		BankSeen: o.Bool("bancoVisto"), Skills: map[string]Skill{}, Known: map[string][]Known{}, Trainer: map[string][]Teach{},
	}
	if p := o.Table("profesiones"); p != nil {
		for name, v := range p.Hash {
			if s, ok := v.(*luasv.Table); ok {
				d.Skills[name] = Skill{Level: int(s.Int("nivel")), Max: int(s.Int("max"))}
			}
		}
	}
	if r := o.Table("recetas"); r != nil {
		for prof, v := range r.Hash {
			pt, ok := v.(*luasv.Table)
			if !ok {
				continue
			}
			var ks []Known
			for _, x := range list(pt.Table("lista")) {
				k := Known{ID: int(x.Int("id")), Name: x.String("nombre"), Color: x.String("color")}
				for _, m := range list(x.Table("mats")) {
					k.Mats = append(k.Mats, Mat{ID: int(m.Int("id")), Name: m.String("nombre"), N: int(m.Int("n"))})
				}
				ks = append(ks, k)
			}
			d.Known[prof] = ks
		}
	}
	if r := o.Table("instructor"); r != nil {
		for prof, v := range r.Hash {
			pt, ok := v.(*luasv.Table)
			if !ok {
				continue
			}
			var ts []Teach
			for _, x := range list(pt) {
				ts = append(ts, Teach{ID: int(x.Int("id")), Name: x.String("nombre"), Rank: x.String("rango"),
					State: x.String("estado"), Level: int(x.Int("nivel")), Price: int(x.Int("precio"))})
			}
			d.Trainer[prof] = ts
		}
	}
	return d, true
}

// Names lee los nombres de objetos en el idioma del juego (CronicaDB.nombres)
// y su calidad (CronicaDB.calidad).
func Names(db *luasv.Table) (map[string]string, map[string]int) {
	names, quality := map[string]string{}, map[string]int{}
	each(db.Table("nombres"), func(k int, v any) {
		if s, ok := v.(string); ok && s != "" {
			names[strconv.Itoa(k)] = s
		}
	})
	each(db.Table("calidad"), func(k int, v any) {
		if n, ok := v.(float64); ok {
			quality[strconv.Itoa(k)] = int(n)
		}
	})
	return names, quality
}

// Index es la lista de personajes con datos de banco y oficios.
type Index struct {
	Characters []IndexEntry      `json:"personajes"`
	Names      map[string]string `json:"nombres"`
	Quality    map[string]int    `json:"calidad"`
}

// IndexEntry es un personaje en el índice.
type IndexEntry struct {
	Key   string `json:"key"`
	Name  string `json:"nombre"`
	Class string `json:"clase,omitempty"`
	Level int    `json:"nivel,omitempty"`
}

// Write guarda docs/oficios/datos/<clave>.json de cada personaje y el índice
// con los nombres. Solo reescribe si algo ha cambiado.
func Write(docs string, chars []Data, names map[string]string, quality map[string]int) error {
	dir := filepath.Join(docs, "oficios", "datos")
	idx := Index{Characters: []IndexEntry{}, Names: names, Quality: quality}
	for _, c := range chars {
		idx.Characters = append(idx.Characters, IndexEntry{Key: c.Key, Name: c.Name, Class: c.Class, Level: c.Level})
		if err := writeIfChanged(filepath.Join(dir, c.Key+".json"), c); err != nil {
			return err
		}
	}
	// Quitar los personajes que ya no están (borrados).
	keep := map[string]bool{"indice.json": true}
	for _, c := range chars {
		keep[c.Key+".json"] = true
	}
	if entries, err := os.ReadDir(dir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && !keep[e.Name()] {
				os.Remove(filepath.Join(dir, e.Name()))
			}
		}
	}
	sort.Slice(idx.Characters, func(i, j int) bool { return idx.Characters[i].Level > idx.Characters[j].Level })
	return writeIfChanged(filepath.Join(dir, "indice.json"), idx)
}

func writeIfChanged(path string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if cur, err := os.ReadFile(path); err == nil && string(cur) == string(b) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// RecipeIDs devuelve los objetos de la tabla de recetas (para que el addon
// pida sus nombres en el idioma del juego).
func RecipeIDs(recetasJSON []byte) []int {
	var t struct {
		Objetos map[string]json.RawMessage `json:"objetos"`
	}
	if json.Unmarshal(recetasJSON, &t) != nil {
		return nil
	}
	var ids []int
	for k := range t.Objetos {
		if n, err := strconv.Atoi(k); err == nil {
			ids = append(ids, n)
		}
	}
	sort.Ints(ids)
	return ids
}
