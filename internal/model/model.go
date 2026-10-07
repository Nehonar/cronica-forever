// Package model define los datos que registra el addon y los convierte desde SavedVariables.
package model

import (
	"fmt"
	"os"
	"sort"

	"github.com/Nehonar/cronica-forever/internal/luasv"
)

// Tipos de evento que escribe el addon.
const (
	EvLogin        = "login"
	EvLogout       = "logout"
	EvLevel        = "level"
	EvZone         = "zone"
	EvQuestAccept  = "quest_accept"
	EvQuestTurnin  = "quest_turnin"
	EvRep          = "rep"
	EvStanding     = "standing"
	EvEquip        = "equip"
	EvDeath        = "death"
	EvQuestAbandon = "quest_abandon"
)

// Event es una cosa que ha pasado en el juego.
type Event struct {
	T          int64  `json:"t"`
	Type       string `json:"type"`
	ID         int64  `json:"id,omitempty"`
	Title      string `json:"title,omitempty"`
	NPC        string `json:"npc,omitempty"`
	Zone       string `json:"zone,omitempty"`
	Subzone    string `json:"subzone,omitempty"`
	Level      int    `json:"level,omitempty"`
	Text       string `json:"text,omitempty"`
	Objectives string `json:"objectives,omitempty"`
	Reward     string `json:"reward,omitempty"`
	Item       string `json:"item,omitempty"`
	Quality    int    `json:"quality,omitempty"`
	Slot       string `json:"slot,omitempty"`
	// Datos de la pieza de equipo (solo en eventos «equip»).
	ItemType    string         `json:"itemType,omitempty"`    // p. ej. «Arma», «Armadura»
	ItemSubType string         `json:"itemSubType,omitempty"` // p. ej. «Espadas de una mano», «Placas»
	Armor       int            `json:"armor,omitempty"`
	Stats       map[string]int `json:"stats,omitempty"` // claves de la API: ITEM_MOD_STRENGTH_SHORT…
	// Tiempo jugado (segundos) en el momento del evento; solo en «level».
	Played int64 `json:"played,omitempty"`
	// Reputación (eventos «rep» y «standing»).
	Faction    string `json:"faction,omitempty"`
	Amount     int    `json:"amount,omitempty"`     // reputación ganada
	Standing   string `json:"standing,omitempty"`   // rango alcanzado: «Amistoso»…
	StandingID int    `json:"standingID,omitempty"` // 5 Amistoso, 6 Honorable, 7 Venerado, 8 Exaltado
}

// Character es un personaje tal como lo guarda el addon.
type Character struct {
	Key       string  `json:"key"`
	Name      string  `json:"name"`
	Surname   string  `json:"surname,omitempty"` // apellido (WoW Forever)
	Realm     string  `json:"realm"`
	Race      string  `json:"race"`
	Class     string  `json:"class"`
	ClassFile string  `json:"classFile"`
	Level     int     `json:"level"`
	Played    int64   `json:"played"`
	Events    []Event `json:"-"`
}

// LoadSavedVariables lee el archivo Cronica.lua del addon.
func LoadSavedVariables(path string) ([]Character, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseSavedVariables(string(b))
}

// ParseSavedVariables convierte el contenido de Cronica.lua en personajes.
func ParseSavedVariables(src string) ([]Character, error) {
	vars, err := luasv.Parse(src)
	if err != nil {
		return nil, err
	}
	db, ok := vars["CronicaDB"].(*luasv.Table)
	if !ok {
		return nil, fmt.Errorf("el archivo no contiene CronicaDB")
	}
	chars := db.Table("characters")
	if chars == nil {
		return nil, nil
	}
	keys := make([]string, 0, len(chars.Hash))
	for k := range chars.Hash {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out []Character
	for _, k := range keys {
		ct, ok := chars.Hash[k].(*luasv.Table)
		if !ok {
			continue
		}
		c := Character{
			Key:       k,
			Name:      ct.String("name"),
			Surname:   ct.String("surname"),
			Realm:     ct.String("realm"),
			Race:      ct.String("race"),
			Class:     ct.String("class"),
			ClassFile: ct.String("classFile"),
			Level:     int(ct.Int("level")),
			Played:    ct.Int("played"),
		}
		if evs := ct.Table("events"); evs != nil {
			for _, raw := range evs.Array {
				et, ok := raw.(*luasv.Table)
				if !ok {
					continue
				}
				c.Events = append(c.Events, Event{
					T:           et.Int("t"),
					Type:        et.String("type"),
					ID:          et.Int("id"),
					Title:       et.String("title"),
					NPC:         et.String("npc"),
					Zone:        et.String("zone"),
					Subzone:     et.String("subzone"),
					Level:       int(et.Int("level")),
					Text:        et.String("text"),
					Objectives:  et.String("objectives"),
					Reward:      et.String("reward"),
					Item:        et.String("item"),
					Quality:     int(et.Int("quality")),
					Slot:        et.String("slot"),
					ItemType:    et.String("itemType"),
					ItemSubType: et.String("itemSubType"),
					Armor:       int(et.Int("armor")),
					Stats:       statsOf(et.Table("stats")),
					Played:      et.Int("played"),
					Faction:     et.String("faction"),
					Amount:      int(et.Int("amount")),
					Standing:    et.String("standing"),
					StandingID:  int(et.Int("standingID")),
				})
			}
		}
		sort.SliceStable(c.Events, func(i, j int) bool { return c.Events[i].T < c.Events[j].T })
		out = append(out, c)
	}
	return out, nil
}

func statsOf(t *luasv.Table) map[string]int {
	if t == nil || len(t.Hash) == 0 {
		return nil
	}
	out := map[string]int{}
	for k, v := range t.Hash {
		if f, ok := v.(float64); ok && f != 0 {
			out[k] = int(f)
		}
	}
	return out
}

// FullName es el nombre con apellido (en Forever, lo único que es único).
func (c Character) FullName() string {
	if c.Surname == "" {
		return c.Name
	}
	return c.Name + " " + c.Surname
}
