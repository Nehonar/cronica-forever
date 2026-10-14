package group

import (
	"testing"

	"github.com/Nehonar/cronica-forever/internal/model"
)

// q crea la aceptación y la entrega de una misión en una zona.
func qev(id int64, zone string, t int64, level int) []model.Event {
	return []model.Event{
		{Type: model.EvQuestAccept, ID: id, Title: "M" + string(rune('A'+id%26)), NPC: "PNJ", Zone: zone, T: t, Level: level},
		{Type: model.EvQuestTurnin, ID: id, NPC: "Otro", Zone: zone, T: t + 10, Level: level},
	}
}

func TestChaptersByZone(t *testing.T) {
	var evs []model.Event
	tm := int64(1000)
	add := func(id int64, zone string) {
		evs = append(evs, qev(id, zone, tm, 6)...)
		tm += 100
	}
	for i := int64(1); i <= 5; i++ {
		add(i, "Elwynn")
	}
	add(6, "Ventormenta") // visita corta: se queda en Elwynn
	add(7, "Elwynn")
	for i := int64(8); i <= 11; i++ {
		add(i, "Páramos de Poniente")
	}
	res := Chapters(evs, DefaultChapterOptions())
	if len(res.Closed) != 1 {
		t.Fatalf("capítulos cerrados = %d, quiero 1", len(res.Closed))
	}
	c := res.Closed[0]
	if c.Zone != "Elwynn" || len(c.Quests) != 7 || !c.FirstVisit {
		t.Fatalf("capítulo 1 = %s con %d misiones (primera vez %v)", c.Zone, len(c.Quests), c.FirstVisit)
	}
	if res.Open == nil || res.Open.Zone != "Páramos de Poniente" || len(res.Open.Quests) != 4 {
		t.Fatalf("capítulo en curso mal: %+v", res.Open)
	}
}

func TestChaptersSplitLong(t *testing.T) {
	var evs []model.Event
	for i := int64(1); i <= 30; i++ {
		evs = append(evs, qev(i, "Elwynn", 1000+i*100, 8)...)
	}
	opt := DefaultChapterOptions()
	opt.Max = 12
	res := Chapters(evs, opt)
	if len(res.Closed) != 2 || res.Closed[1].Part != 2 || res.Closed[1].FirstVisit {
		t.Fatalf("esperaba dos capítulos cerrados de Elwynn, el segundo «parte 2»: %d", len(res.Closed))
	}
	if res.Open == nil || len(res.Open.Quests) != 6 {
		t.Fatalf("el resto debería seguir abierto")
	}
}

func TestChaptersScenesAndContext(t *testing.T) {
	evs := qev(1, "Elwynn", 1000, 9)
	evs = append(evs,
		model.Event{Type: model.EvLevel, Level: 10, T: 1050},
		model.Event{Type: model.EvDeath, T: 1060, Zone: "Elwynn"},
		model.Event{Type: model.EvEquip, Item: "Espada", Quality: 3, T: 1070},
		model.Event{Type: model.EvStanding, Faction: "Ventormenta", Standing: "Amistoso", StandingID: 5, T: 1080},
	)
	for i := int64(2); i <= 4; i++ {
		evs = append(evs, qev(i, "Loch Modan", 2000+i*100, 10)...)
	}
	res := Chapters(evs, DefaultChapterOptions())
	if len(res.Closed) != 1 {
		t.Fatalf("cerrados = %d", len(res.Closed))
	}
	c := res.Closed[0]
	if c.Deaths != 1 || len(c.Items) != 1 || len(c.Scenes) != 2 || c.LevelTo != 10 {
		t.Fatalf("contexto mal: muertes %d, equipo %d, escenas %+v, nivel %d", c.Deaths, len(c.Items), c.Scenes, c.LevelTo)
	}
}

func TestChaptersRemoved(t *testing.T) {
	var evs []model.Event
	for i := int64(1); i <= 3; i++ {
		evs = append(evs, qev(i, "Elwynn", 1000+i*100, 5)...)
	}
	opt := DefaultChapterOptions()
	opt.Removed = map[int64]bool{2: true}
	res := Chapters(evs, opt)
	if res.Open == nil || len(res.Open.Quests) != 2 {
		t.Fatalf("la misión quitada no debería contar")
	}
}
