package group

import (
	"testing"

	"github.com/Nehonar/cronica-forever/internal/model"
)

func ev(t int64, typ string, f func(*model.Event)) model.Event {
	e := model.Event{T: t, Type: typ, Zone: "Elwynn"}
	if f != nil {
		f(&e)
	}
	return e
}

func quest(t int64, id int64, npc string) []model.Event {
	return []model.Event{
		ev(t, model.EvQuestAccept, func(e *model.Event) { e.ID, e.NPC, e.Title = id, npc, "Misión" }),
		ev(t+100, model.EvQuestTurnin, func(e *model.Event) { e.ID, e.NPC = id, npc }),
	}
}

func TestStoriesThreshold(t *testing.T) {
	var evs []model.Event
	// 10 misiones sueltas (2 puntos) + 6 niveles (6 puntos) = 8 → historia.
	for i := int64(0); i < 10; i++ {
		evs = append(evs, quest(1000+i*1000, i+1, "PNJ"+string(rune('A'+i)))...)
	}
	for l := 2; l <= 7; l++ {
		lv := l
		evs = append(evs, ev(20000+int64(l)*10, model.EvLevel, func(e *model.Event) { e.Level = lv }))
	}
	r := Stories(evs, StoryOptions{Now: 1 << 40})
	if len(r.Closed) != 1 {
		t.Fatalf("historias cerradas: %d", len(r.Closed))
	}
	g := r.Closed[0]
	if g.Points != 8 || len(g.Quests) != 10 || g.LevelTo != 7 || g.Zone != "Elwynn" {
		t.Fatalf("historia: %+v", g)
	}
	if r.Open != nil {
		t.Fatalf("no debería quedar nada abierto: %+v", r.Open)
	}
}

func TestStoriesBigMoment(t *testing.T) {
	evs := []model.Event{
		ev(10, model.EvLevel, func(e *model.Event) { e.Level = 8 }),
		ev(20, model.EvLevel, func(e *model.Event) { e.Level = 9 }),
		// Mazmorra: 2 + 4 = 6 ≥ 4 con momento gordo → historia.
		ev(30, model.EvDungeon, func(e *model.Event) {
			e.Instance, e.Bosses, e.Title, e.Final = "Minas de la Muerte", 5, "Edwin VanCleef", true
		}),
		ev(40, model.EvEquip, func(e *model.Event) { e.Item, e.Quality = "Espada", 3 }),
	}
	r := Stories(evs, StoryOptions{Now: 1 << 40})
	if len(r.Closed) != 1 || r.Closed[0].Points != 6 || r.Closed[0].Scenes[2].Kind != "mazmorra" {
		t.Fatalf("%+v", r.Closed)
	}
	if r.Open == nil || r.Open.Points != 1 || len(r.Open.Items) != 1 {
		t.Fatalf("abierta: %+v", r.Open)
	}
	// Un momento gordo con pocos puntos no basta: nivel 10 (3) sin nada más.
	r = Stories([]model.Event{ev(10, model.EvLevel, func(e *model.Event) { e.Level = 10 })}, StoryOptions{})
	if len(r.Closed) != 0 || r.Open.Points != 3 {
		t.Fatalf("%+v", r)
	}
}

func TestStoriesChainAndSince(t *testing.T) {
	var evs []model.Event
	// Cadena de 3 con el mismo PNJ: entregar y aceptar la siguiente enseguida.
	evs = append(evs,
		ev(100, model.EvQuestAccept, func(e *model.Event) { e.ID, e.NPC, e.Title = 1, "Gryan", "Uno" }),
		ev(200, model.EvQuestTurnin, func(e *model.Event) { e.ID, e.NPC = 1, "Gryan" }),
		ev(210, model.EvQuestAccept, func(e *model.Event) { e.ID, e.NPC, e.Title = 2, "Gryan", "Dos" }),
		ev(300, model.EvQuestTurnin, func(e *model.Event) { e.ID, e.NPC = 2, "Gryan" }),
		ev(310, model.EvQuestAccept, func(e *model.Event) { e.ID, e.NPC, e.Title = 3, "Gryan", "Tres" }),
		ev(400, model.EvQuestTurnin, func(e *model.Event) { e.ID, e.NPC = 3, "Gryan" }),
	)
	// Aún dentro del margen: la cadena no está cerrada.
	r := Stories(evs, StoryOptions{Now: 450})
	if r.Open == nil || r.Open.Points != 0 {
		t.Fatalf("antes del margen: %+v", r.Open)
	}
	r = Stories(evs, StoryOptions{Now: 1000})
	if r.Open == nil || r.Open.Points != PtsChain || r.Open.Scenes[0].Title != "Tres" || r.Open.Scenes[0].Detail != "3 misiones encadenadas" {
		t.Fatalf("cadena: %+v", r.Open)
	}
	// Zona ya vista antes del corte: no cuenta como nueva.
	evs = []model.Event{
		ev(10, model.EvZone, func(e *model.Event) { e.Zone = "Elwynn" }),
		ev(20, model.EvZone, func(e *model.Event) { e.Zone = "Páramos de Poniente" }),
		ev(30, model.EvZone, func(e *model.Event) { e.Zone = "Elwynn" }),
		ev(40, model.EvZone, func(e *model.Event) { e.Zone = "Montañas Crestagrana" }),
	}
	r = Stories(evs, StoryOptions{Since: 25})
	if r.Open == nil || r.Open.Points != 1 || r.Open.Zones[0] != "Montañas Crestagrana" {
		t.Fatalf("zonas: %+v", r.Open)
	}
}
