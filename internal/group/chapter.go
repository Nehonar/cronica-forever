package group

import (
	"fmt"
	"sort"

	"github.com/Nehonar/cronica-forever/internal/model"
)

// Chapter es el tipo de pieza de la crónica por capítulos: todo lo vivido en una
// zona, de que llega a que se marcha. Las cadenas, el equipo, la reputación y los
// niveles van dentro como escenas, no como relatos aparte.
const Chapter Kind = "capitulo"

// Scene es un momento destacado dentro de un capítulo (una línea, sin relato propio).
type Scene struct {
	Kind   string `json:"kind"`             // cadena, nivel, reputacion
	Title  string `json:"title"`            // lo que se muestra en grande
	Detail string `json:"detail,omitempty"` // una línea de contexto
}

// ChapterOptions controla dónde se corta un capítulo.
type ChapterOptions struct {
	// Switch: misiones seguidas entregadas en otra zona para dar por hecho que el
	// personaje se ha ido (las visitas cortas, por ejemplo a la ciudad, no cortan).
	Switch int
	// Max: misiones como mucho en un capítulo; si se pasa, se parte en dos.
	Max int
	// Removed: misiones quitadas a mano, que no se cuentan.
	Removed map[int64]bool
	// MinChain: cadenas de al menos esta longitud salen como escena.
	MinChain int
	// MinQuality: calidad mínima del equipo que se apunta en el capítulo.
	MinQuality int
	// ChainWindow: segundos entre entregar y aceptar la siguiente para enlazar cadenas.
	ChainWindow int64
}

// DefaultChapterOptions son los valores por defecto.
func DefaultChapterOptions() ChapterOptions {
	return ChapterOptions{Switch: 3, Max: 25, MinChain: 3, MinQuality: 3, ChainWindow: 90}
}

// ChapterResult es la salida de Chapters.
type ChapterResult struct {
	Closed []Group // capítulos terminados, listos para narrar, en orden
	Open   *Group  // el capítulo en curso (aún no se narra); nil si no hay misiones
}

// Chapters reparte lo jugado en capítulos por zona.
func Chapters(events []model.Event, opt ChapterOptions) ChapterResult {
	if opt.Switch <= 0 {
		opt.Switch = 3
	}
	if opt.Max <= 0 {
		opt.Max = 25
	}
	if opt.MinChain <= 0 {
		opt.MinChain = 3
	}
	if opt.MinQuality <= 0 {
		opt.MinQuality = 3
	}
	if opt.ChainWindow <= 0 {
		opt.ChainWindow = 90
	}
	done, next, prev := turnedIn(events, opt.ChainWindow)
	var qs []*Quest
	for _, q := range done {
		if !opt.Removed[q.ID] {
			qs = append(qs, q)
		}
	}

	// Cortes: un capítulo sigue en su zona mientras no se entreguen `Switch`
	// misiones seguidas en otra misma zona. Lo de en medio (una visita a la
	// ciudad, por ejemplo) se queda en el capítulo.
	type span struct {
		zone   string
		quests []*Quest
	}
	var spans []span
	var cur *span
	var away []*Quest
	absorb := func() {
		cur.quests = append(cur.quests, away...)
		away = nil
	}
	for _, q := range qs {
		if cur == nil {
			spans = append(spans, span{zone: q.Zone})
			cur = &spans[len(spans)-1]
		}
		if q.Zone == cur.zone || q.Zone == "" {
			absorb()
			cur.quests = append(cur.quests, q)
		} else {
			away = append(away, q)
			// ¿las últimas `Switch` son todas de la misma zona nueva?
			if len(away) >= opt.Switch {
				z := away[len(away)-1].Zone
				run := 0
				for i := len(away) - 1; i >= 0 && away[i].Zone == z; i-- {
					run++
				}
				if run >= opt.Switch {
					moved := away[len(away)-run:]
					away = away[:len(away)-run]
					absorb()
					spans = append(spans, span{zone: z, quests: append([]*Quest(nil), moved...)})
					cur = &spans[len(spans)-1]
				}
			}
		}
		if len(cur.quests) >= opt.Max && len(away) == 0 {
			// Capítulo demasiado largo: se cierra y sigue otro en la misma zona.
			spans = append(spans, span{zone: cur.zone})
			cur = &spans[len(spans)-1]
		}
	}
	if cur != nil {
		absorb()
	}
	// Quitar capítulos vacíos (el que se abre justo al partir uno largo).
	var full []span
	for _, s := range spans {
		if len(s.quests) > 0 {
			full = append(full, s)
		}
	}
	if len(full) == 0 {
		return ChapterResult{}
	}

	// Cada capítulo abarca desde su primera entrega hasta la primera del siguiente.
	bounds := make([]int64, len(full)+1)
	for i, s := range full {
		bounds[i] = s.quests[0].TurninT
	}
	bounds[0] = -1 << 62
	bounds[len(full)] = 1 << 62

	visited := map[string]bool{}
	part := map[string]int{}
	var out []Group
	for i, s := range full {
		g := Group{Kind: Chapter, Zone: s.zone}
		if !visited[s.zone] {
			g.FirstVisit = true
			visited[s.zone] = true
		}
		part[s.zone]++
		g.Part = part[s.zone]
		subs := map[string]bool{}
		g.Start, g.End = 1<<62, 0
		for _, q := range s.quests {
			g.Quests = append(g.Quests, *q)
			if q.Subzone != "" && !subs[q.Subzone] && q.Zone == s.zone {
				subs[q.Subzone] = true
				g.Subzones = append(g.Subzones, q.Subzone)
			}
			st := q.AcceptT
			if st == 0 {
				st = q.TurninT
			}
			if st < g.Start {
				g.Start = st
			}
			if q.TurninT > g.End {
				g.End = q.TurninT
			}
		}
		sort.SliceStable(g.Quests, func(a, b int) bool { return g.Quests[a].TurninT < g.Quests[b].TurninT })
		g.ID = fmt.Sprintf("capitulo-%s-%d", slug(s.zone), s.quests[0].ID)
		from, to := bounds[i], bounds[i+1]
		inside := func(t int64) bool { return t >= from && t < to }

		// Niveles, equipo, reputación y caídas del tramo.
		seenItem := map[string]bool{}
		for _, e := range events {
			if e.Level > 0 && (e.T < from || (g.LevelFrom == 0 && inside(e.T))) {
				g.LevelFrom = e.Level
			}
			if e.Level > 0 && e.T < to {
				g.LevelTo = e.Level
			}
			if !inside(e.T) {
				continue
			}
			switch e.Type {
			case model.EvDeath:
				g.Deaths++
			case model.EvEquip:
				if e.Quality >= opt.MinQuality && e.Item != "" && !seenItem[e.Item] {
					seenItem[e.Item] = true
					g.Items = append(g.Items, e)
				}
			case model.EvRep:
				if e.Faction != "" && e.Amount > 0 {
					if g.RepGains == nil {
						g.RepGains = map[string]int{}
					}
					g.RepGains[e.Faction] += e.Amount
				}
			case model.EvStanding:
				if e.Faction != "" && e.StandingID >= 5 {
					g.Scenes = append(g.Scenes, Scene{Kind: "reputacion", Title: e.Faction, Detail: e.Standing})
				}
			case model.EvLevel:
				if e.Level > 0 && e.Level%10 == 0 {
					g.Scenes = append(g.Scenes, Scene{Kind: "nivel", Title: fmt.Sprintf("Nivel %d", e.Level)})
				}
			}
		}
		if g.LevelFrom == 0 {
			g.LevelFrom = g.LevelTo
		}

		// Cadenas largas completadas dentro del capítulo.
		in := map[int64]bool{}
		for _, q := range s.quests {
			in[q.ID] = true
		}
		for _, q := range s.quests {
			if _, hasPrev := prev[q.ID]; hasPrev {
				continue
			}
			if _, hasNext := next[q.ID]; !hasNext {
				continue
			}
			n, last := 0, q.Title
			for c, ok := q.ID, true; ok && in[c]; c, ok = next[c] {
				n++
				for _, x := range s.quests {
					if x.ID == c {
						last = x.Title
					}
				}
			}
			if n >= opt.MinChain {
				g.Scenes = append(g.Scenes, Scene{Kind: "cadena", Title: last, Detail: fmt.Sprintf("%d misiones encadenadas", n)})
			}
		}
		out = append(out, g)
	}

	res := ChapterResult{Closed: out[:len(out)-1]}
	open := out[len(out)-1]
	res.Open = &open
	return res
}

// turnedIn devuelve las misiones entregadas en orden de entrega y los enlaces de
// cadena (entregas a alguien y te da otra enseguida).
func turnedIn(events []model.Event, window int64) ([]*Quest, map[int64]int64, map[int64]int64) {
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
			q.TurninT, q.TurninNPC = 0, ""
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
			if a.t >= q.TurninT && a.t-q.TurninT <= window {
				if _, taken := prev[a.id]; !taken {
					next[id] = a.id
					prev[a.id] = id
				}
				break
			}
		}
	}
	var done []*Quest
	for _, id := range order {
		if quests[id].TurninT > 0 {
			done = append(done, quests[id])
		}
	}
	sort.SliceStable(done, func(i, j int) bool { return done[i].TurninT < done[j].TurninT })
	return done, next, prev
}
