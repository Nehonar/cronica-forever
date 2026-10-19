package group

import (
	"fmt"

	"github.com/Nehonar/cronica-forever/internal/model"
)

// Story es el tipo de pieza de la crónica desde la 0.6: una historia corta que
// se escribe cuando el personaje ha vivido bastantes momentos clave (subir de
// nivel, terminar una mazmorra o una cadena, ponerse una pieza buena…). Las
// misiones entregadas solo dan el contexto: su texto ya lo narra Spoken.
//
// El addon lleva la misma cuenta (Cronica.lua, «Historias») para avisar en el
// juego: si cambias los pesos aquí, cámbialos también allí.
const Story Kind = "historia"

// Pesos de cada momento.
const (
	PtsLevel      = 1 // subir de nivel
	PtsLevelTen   = 3 // nivel múltiplo de 10 (momento gordo)
	PtsDungeon    = 4 // mazmorra con algún jefe derrotado (momento gordo)
	PtsChain      = 2 // cadena de 3 o más misiones terminada
	PtsBlue       = 1 // ponerse por primera vez una pieza azul
	PtsEpic       = 3 // ponerse por primera vez una pieza morada o mejor (momento gordo)
	PtsStanding   = 2 // subir de rango con una facción (Amistoso o más)
	PtsNewZone    = 1 // pisar una zona por primera vez
	QuestsPerPt   = 5 // cada 5 misiones entregadas, 1 punto
	minChainScene = 3
)

// StoryOptions controla cuándo se cierra una historia.
type StoryOptions struct {
	// Threshold: puntos con los que se escribe la historia.
	Threshold int
	// BigMin: con un momento gordo, basta con estos puntos.
	BigMin int
	// ChainWindow: segundos entre entregar y aceptar la siguiente para enlazar cadenas.
	ChainWindow int64
	// Since: solo cuenta lo que pasó después (lo anterior ya está contado en
	// capítulos o historias escritas). Lo de antes sirve para saber qué zonas
	// ya conocía y cómo iban las cadenas.
	Since int64
	// Now: hora actual, para dar por terminada una cadena si ya no llega la siguiente.
	Now int64
}

// DefaultStoryOptions son los valores por defecto.
func DefaultStoryOptions() StoryOptions {
	return StoryOptions{Threshold: 8, BigMin: 4, ChainWindow: 90}
}

// StoryResult es la salida de Stories.
type StoryResult struct {
	Closed []Group // historias listas para escribir, en orden
	Open   *Group  // lo que lleva acumulado la siguiente (nil si nada)
}

type pendingTurnin struct {
	t      int64
	npc    string
	id     int64
	title  string
	length int
	linked bool
}

// Stories reparte lo jugado en historias por puntos.
func Stories(events []model.Event, opt StoryOptions) StoryResult {
	if opt.Threshold <= 0 {
		opt.Threshold = 8
	}
	if opt.BigMin <= 0 {
		opt.BigMin = 4
	}
	if opt.ChainWindow <= 0 {
		opt.ChainWindow = 90
	}

	var res StoryResult
	var cur *Group
	var questsIn int
	seenZone := map[string]bool{}
	chainLen := map[int64]int{}
	accepted := map[int64]model.Event{}
	level := 0
	var last *pendingTurnin
	var lastT int64

	start := func(t int64) {
		if cur == nil {
			cur = &Group{Kind: Story, Start: t, LevelFrom: level, LevelTo: level}
			questsIn = 0
		}
	}
	addZone := func(z string) {
		if z == "" || cur == nil {
			return
		}
		for _, x := range cur.Zones {
			if x == z {
				return
			}
		}
		cur.Zones = append(cur.Zones, z)
	}
	// score suma puntos y cierra la historia si toca.
	score := func(t int64, pts int, big bool, sc *Scene) {
		start(t)
		cur.Points += pts
		if sc != nil {
			cur.Scenes = append(cur.Scenes, *sc)
		}
		if cur.Points >= opt.Threshold || (big && cur.Points >= opt.BigMin) {
			cur.End = t
			finish(cur)
			res.Closed = append(res.Closed, *cur)
			cur = nil
		}
	}
	// endChain da por terminada la cadena de la última entrega si nadie la siguió.
	endChain := func() {
		if last == nil {
			return
		}
		p := last
		last = nil
		if p.linked || p.length < minChainScene || p.t <= opt.Since {
			return
		}
		score(p.t, PtsChain, false, &Scene{Kind: "cadena", Title: p.title, Detail: fmt.Sprintf("%d misiones encadenadas", p.length)})
	}

	for _, e := range events {
		if last != nil && e.T > last.t+opt.ChainWindow {
			endChain()
		}
		if e.T > lastT {
			lastT = e.T
		}
		live := e.T > opt.Since
		switch e.Type {
		case model.EvQuestAccept:
			accepted[e.ID] = e
			if last != nil && !last.linked && e.NPC != "" && e.NPC == last.npc && e.T-last.t <= opt.ChainWindow {
				last.linked = true
				chainLen[e.ID] = last.length + 1
			}
		case model.EvQuestTurnin:
			if last != nil && !last.linked {
				endChain()
			}
			n := chainLen[e.ID]
			if n == 0 {
				n = 1
			}
			title := e.Title
			if title == "" {
				title = accepted[e.ID].Title
			}
			last = &pendingTurnin{t: e.T, npc: e.NPC, id: e.ID, title: title, length: n}
			if !live {
				continue
			}
			start(e.T)
			q := Quest{ID: e.ID, Title: title, TurninNPC: e.NPC, Reward: e.Reward, TurninT: e.T, LevelTurnin: e.Level}
			if a, ok := accepted[e.ID]; ok {
				q.Giver, q.Zone, q.Subzone, q.Objectives, q.Text, q.AcceptT = a.NPC, a.Zone, a.Subzone, a.Objectives, a.Text, a.T
			} else {
				q.Zone, q.Subzone = e.Zone, e.Subzone
			}
			cur.Quests = append(cur.Quests, q)
			addZone(q.Zone)
			questsIn++
			if questsIn%QuestsPerPt == 0 {
				score(e.T, 1, false, nil)
			}
		case model.EvZone:
			first := e.Zone != "" && !seenZone[e.Zone]
			seenZone[e.Zone] = true
			if live && first {
				start(e.T)
				addZone(e.Zone)
				score(e.T, PtsNewZone, false, &Scene{Kind: "zona", Title: e.Zone})
			}
		case model.EvLevel:
			if !live || e.Level <= 0 {
				if e.Level > 0 {
					level = e.Level
				}
				continue
			}
			start(e.T)
			level = e.Level
			cur.LevelTo = e.Level
			if e.Level%10 == 0 {
				score(e.T, PtsLevelTen, true, &Scene{Kind: "nivel", Title: fmt.Sprintf("Nivel %d", e.Level)})
			} else {
				score(e.T, PtsLevel, false, &Scene{Kind: "nivel", Title: fmt.Sprintf("Nivel %d", e.Level)})
			}
		case model.EvDungeon:
			if !live || e.Bosses <= 0 {
				continue
			}
			start(e.T)
			name := e.Instance
			if name == "" {
				name = e.Zone
			}
			detail := fmt.Sprintf("%d jefe(s) derrotado(s)", e.Bosses)
			if e.Final {
				detail = "terminada"
			}
			if e.Title != "" {
				detail += "; el último, " + e.Title
			}
			score(e.T, PtsDungeon, true, &Scene{Kind: "mazmorra", Title: name, Detail: detail})
		case model.EvEquip:
			if !live || e.Quality < 3 {
				continue
			}
			start(e.T)
			cur.Items = append(cur.Items, e)
			if e.Quality >= 4 {
				score(e.T, PtsEpic, true, &Scene{Kind: "equipo", Title: e.Item, Detail: "pieza excepcional"})
			} else {
				score(e.T, PtsBlue, false, &Scene{Kind: "equipo", Title: e.Item})
			}
		case model.EvStanding:
			if !live || e.Faction == "" || e.StandingID < 5 {
				continue
			}
			start(e.T)
			score(e.T, PtsStanding, false, &Scene{Kind: "reputacion", Title: e.Faction, Detail: e.Standing})
		case model.EvRep:
			if live && cur != nil && e.Faction != "" && e.Amount > 0 {
				if cur.RepGains == nil {
					cur.RepGains = map[string]int{}
				}
				cur.RepGains[e.Faction] += e.Amount
			}
		case model.EvDeath:
			if live && cur != nil {
				cur.Deaths++
			}
		}
	}
	// ¿Ha pasado ya el margen para que la última entrega tuviera continuación?
	if last != nil && opt.Now > last.t+opt.ChainWindow {
		endChain()
	}
	if cur != nil {
		finish(cur)
		if cur.End == 0 {
			cur.End = lastT
		}
		res.Open = cur
	}
	return res
}

// finish rellena la zona principal y su id.
func finish(g *Group) {
	count := map[string]int{}
	best := ""
	for _, q := range g.Quests {
		if q.Zone == "" {
			continue
		}
		count[q.Zone]++
		if count[q.Zone] > count[best] {
			best = q.Zone
		}
	}
	if best == "" && len(g.Zones) > 0 {
		best = g.Zones[len(g.Zones)-1]
	}
	g.Zone = best
	g.Subzones = nil
	subs := map[string]bool{}
	for _, q := range g.Quests {
		if q.Zone == best && q.Subzone != "" && !subs[q.Subzone] {
			subs[q.Subzone] = true
			g.Subzones = append(g.Subzones, q.Subzone)
		}
	}
	if g.LevelFrom == 0 {
		g.LevelFrom = g.LevelTo
	}
	g.ID = fmt.Sprintf("historia-%d", g.Start)
}
