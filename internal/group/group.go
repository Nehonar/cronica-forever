// Package group agrupa los eventos del addon en piezas narrables:
// cadenas de misiones, grupos de misiones sueltas e hitos de equipo.
package group

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Nehonar/cronica-forever/internal/model"
)

// Kind es el tipo de pieza narrable.
type Kind string

const (
	Chain Kind = "cadena"
	Loose Kind = "sueltas"
	Equip Kind = "equipo"
)

// Options controla la agrupación.
type Options struct {
	// ChainWindow: segundos tras entregar una misión en los que, si el mismo PNJ
	// te da otra, se consideran parte de la misma cadena.
	ChainWindow int64
	// LooseMax: tamaño máximo de un grupo de misiones sueltas.
	LooseMax int
	// MinQuality: calidad mínima del equipo que genera hito (3 = azul).
	MinQuality int
}

// DefaultOptions son los valores por defecto.
func DefaultOptions() Options { return Options{ChainWindow: 90, LooseMax: 4, MinQuality: 3} }

// Quest reúne lo que se sabe de una misión.
type Quest struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Giver       string `json:"giver,omitempty"`
	TurninNPC   string `json:"turninNpc,omitempty"`
	Zone        string `json:"zone,omitempty"`
	Subzone     string `json:"subzone,omitempty"`
	Text        string `json:"-"`
	Objectives  string `json:"objectives,omitempty"`
	Reward      string `json:"-"`
	AcceptT     int64  `json:"-"`
	TurninT     int64  `json:"-"`
	LevelTurnin int    `json:"-"`
}

// Group es una pieza lista para narrar.
type Group struct {
	ID        string        `json:"id"`
	Kind      Kind          `json:"kind"`
	Zone      string        `json:"zone"`
	Subzones  []string      `json:"subzones,omitempty"`
	Quests    []Quest       `json:"quests,omitempty"`
	Items     []model.Event `json:"items,omitempty"`
	Start     int64         `json:"start"`
	End       int64         `json:"end"`
	LevelFrom int           `json:"levelFrom,omitempty"`
	LevelTo   int           `json:"levelTo,omitempty"`
	Deaths    int           `json:"deaths,omitempty"`
}

// Result es la salida de Build.
type Result struct {
	Groups       []Group
	PendingLoose []Quest   // misiones sueltas que esperan a completar grupo
	OpenChains   [][]Quest // cadenas empezadas y no terminadas
}

// Build agrupa los eventos de un personaje. now es la hora actual (Unix):
// sirve para saber si ya ha pasado la ventana tras la última entrega.
func Build(events []model.Event, now int64, opt Options) Result {
	if opt.ChainWindow <= 0 {
		opt.ChainWindow = 90
	}
	if opt.LooseMax <= 0 {
		opt.LooseMax = 4
	}
	if opt.MinQuality <= 0 {
		opt.MinQuality = 3
	}
	horizon := now
	for _, e := range events {
		if e.T > horizon {
			horizon = e.T
		}
	}

	quests := map[int64]*Quest{}
	var order []int64
	get := func(id int64) *Quest {
		q, ok := quests[id]
		if !ok {
			q = &Quest{ID: id}
			quests[id] = q
			order = append(order, id)
		}
		return q
	}
	type accept struct {
		t   int64
		npc string
		id  int64
	}
	var accepts []accept
	for _, e := range events {
		switch e.Type {
		case model.EvQuestAccept:
			q := get(e.ID)
			q.Title, q.Giver, q.Zone, q.Subzone = e.Title, e.NPC, e.Zone, e.Subzone
			q.Text, q.Objectives, q.AcceptT = e.Text, e.Objectives, e.T
			q.TurninT, q.TurninNPC = 0, "" // si se repite, cuenta la última vez
			accepts = append(accepts, accept{e.T, e.NPC, e.ID})
		case model.EvQuestTurnin:
			q := get(e.ID)
			if q.Title == "" {
				q.Title = e.Title
			}
			if q.Zone == "" {
				q.Zone, q.Subzone = e.Zone, e.Subzone
			}
			q.TurninNPC, q.Reward, q.TurninT, q.LevelTurnin = e.NPC, e.Reward, e.T, e.Level
		}
	}

	// Enlaces de cadena: entregas a X y X te da otra misión enseguida.
	next := map[int64]int64{}
	prev := map[int64]int64{}
	for _, id := range order {
		q := quests[id]
		if q.TurninT == 0 || q.TurninNPC == "" {
			continue
		}
		for _, a := range accepts {
			if a.id == id || a.npc != q.TurninNPC {
				continue
			}
			if a.t >= q.TurninT && a.t-q.TurninT <= opt.ChainWindow {
				if _, taken := prev[a.id]; !taken {
					next[id] = a.id
					prev[a.id] = id
				}
				break
			}
		}
	}
	settled := func(q *Quest) bool { return q.TurninT > 0 && horizon-q.TurninT > opt.ChainWindow }

	var res Result
	// Cadenas
	for _, id := range order {
		if _, hasPrev := prev[id]; hasPrev {
			continue
		}
		if _, hasNext := next[id]; !hasNext {
			continue
		}
		var chain []Quest
		complete := true
		for cur, ok := id, true; ok; cur, ok = next[cur] {
			q := quests[cur]
			chain = append(chain, *q)
			if q.TurninT == 0 {
				complete = false
			}
		}
		last := quests[chain[len(chain)-1].ID]
		if complete && settled(last) {
			res.Groups = append(res.Groups, makeGroup(Chain, chain, events))
		} else {
			res.OpenChains = append(res.OpenChains, chain)
		}
	}

	// Sueltas, en orden de entrega
	var loose []*Quest
	for _, id := range order {
		_, p := prev[id]
		_, n := next[id]
		q := quests[id]
		if !p && !n && settled(q) {
			loose = append(loose, q)
		}
	}
	sort.SliceStable(loose, func(i, j int) bool { return loose[i].TurninT < loose[j].TurninT })
	var buf []Quest
	flush := func() {
		if len(buf) > 0 {
			res.Groups = append(res.Groups, makeGroup(Loose, buf, events))
			buf = nil
		}
	}
	for _, q := range loose {
		if len(buf) > 0 && q.Zone != buf[0].Zone {
			flush()
		}
		buf = append(buf, *q)
		if len(buf) >= opt.LooseMax {
			flush()
		}
	}
	res.PendingLoose = buf
	attachContext(res.Groups, events, opt.MinQuality)
	// Las misiones aceptadas y aún no entregadas tampoco se narran todavía.

	// Hitos de equipo: la primera vez que te pones cada pieza azul o mejor.
	seen := map[string]bool{}
	for _, e := range events {
		if e.Type != model.EvEquip || e.Quality < opt.MinQuality || e.Item == "" || seen[e.Item] {
			continue
		}
		seen[e.Item] = true
		g := Group{
			ID:        fmt.Sprintf("equipo-%d-%s", e.T, slug(e.Item)),
			Kind:      Equip,
			Zone:      e.Zone,
			Items:     []model.Event{e},
			Start:     e.T,
			End:       e.T,
			LevelFrom: e.Level,
			LevelTo:   e.Level,
		}
		if e.Subzone != "" {
			g.Subzones = []string{e.Subzone}
		}
		res.Groups = append(res.Groups, g)
	}

	sort.SliceStable(res.Groups, func(i, j int) bool { return res.Groups[i].End < res.Groups[j].End })
	return res
}

func makeGroup(kind Kind, qs []Quest, events []model.Event) Group {
	g := Group{Kind: kind, Quests: qs, Zone: qs[0].Zone}
	ids := make([]string, len(qs))
	g.Start, g.End = 1<<62, 0
	subs := map[string]bool{}
	for i, q := range qs {
		ids[i] = fmt.Sprint(q.ID)
		start := q.AcceptT
		if start == 0 {
			start = q.TurninT
		}
		if start < g.Start {
			g.Start = start
		}
		if q.TurninT > g.End {
			g.End = q.TurninT
		}
		if q.Subzone != "" && !subs[q.Subzone] {
			subs[q.Subzone] = true
			g.Subzones = append(g.Subzones, q.Subzone)
		}
	}
	g.ID = string(kind) + "-" + strings.Join(ids, "-")
	for _, e := range events {
		if e.Level > 0 && e.T <= g.Start {
			g.LevelFrom = e.Level
		}
		if e.Level > 0 && e.T <= g.End {
			g.LevelTo = e.Level
		}
	}
	return g
}

// attachContext reparte muertes y equipo nuevo entre los grupos de misiones:
// cada suceso va solo al primer grupo que termina después de él, para que
// no aparezca repetido en dos relatos cuyos tramos se solapan.
func attachContext(groups []Group, events []model.Event, minQuality int) {
	for _, e := range events {
		isDeath := e.Type == model.EvDeath
		isItem := e.Type == model.EvEquip && e.Quality >= minQuality
		if !isDeath && !isItem {
			continue
		}
		best := -1
		for i, g := range groups {
			if g.Start <= e.T && g.End >= e.T && (best < 0 || g.End < groups[best].End) {
				best = i
			}
		}
		if best < 0 {
			continue
		}
		if isDeath {
			groups[best].Deaths++
		} else {
			groups[best].Items = append(groups[best].Items, e)
		}
	}
}

func slug(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case strings.ContainsRune("áàä", r):
			b.WriteRune('a')
		case strings.ContainsRune("éèë", r):
			b.WriteRune('e')
		case strings.ContainsRune("íìï", r):
			b.WriteRune('i')
		case strings.ContainsRune("óòö", r):
			b.WriteRune('o')
		case strings.ContainsRune("úùü", r):
			b.WriteRune('u')
		case r == 'ñ':
			b.WriteRune('n')
		default:
			if b.Len() > 0 && !strings.HasSuffix(b.String(), "-") {
				b.WriteByte('-')
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
