package group

import (
	"testing"

	"github.com/Nehonar/cronica-forever/internal/model"
)

func load(t *testing.T) []model.Character {
	t.Helper()
	cs, err := model.LoadSavedVariables("../../samples/Cronica.lua")
	if err != nil {
		t.Fatal(err)
	}
	return cs
}

func TestSampleGroups(t *testing.T) {
	c := load(t)[0]
	res := Build(c.Events, 1791200000, DefaultOptions())

	var chains, loose, equip []Group
	for _, g := range res.Groups {
		switch g.Kind {
		case Chain:
			chains = append(chains, g)
		case Loose:
			loose = append(loose, g)
		case Equip:
			equip = append(equip, g)
		}
	}
	if len(chains) != 1 || chains[0].ID != "cadena-783-7-15-21-54" {
		t.Fatalf("cadenas = %+v", chains)
	}
	if chains[0].Deaths != 1 || chains[0].LevelFrom != 1 || chains[0].LevelTo != 5 {
		t.Errorf("contexto de la cadena: muertes=%d niveles=%d-%d", chains[0].Deaths, chains[0].LevelFrom, chains[0].LevelTo)
	}
	if len(chains[0].Items) != 1 {
		t.Errorf("la pieza azul equipada durante la cadena debería figurar en su contexto")
	}
	if len(loose) != 1 || loose[0].ID != "sueltas-33-18-60-88" {
		t.Fatalf("sueltas = %+v", loose)
	}
	if loose[0].Deaths != 0 || len(loose[0].Items) != 0 {
		t.Errorf("la muerte y la espada ya cuentan en la cadena; no deben repetirse en las sueltas")
	}
	if len(equip) != 1 || equip[0].Items[0].Item != "Espada corta del vigía" {
		t.Fatalf("equipo = %+v", equip)
	}
	if len(res.PendingLoose) != 0 || len(res.OpenChains) != 0 {
		t.Errorf("no debería quedar nada pendiente")
	}
}

func TestChainNotSettledYet(t *testing.T) {
	c := load(t)[0]
	// Cortamos el registro justo tras entregar «Informe a Villadorada» (t=1791106700)
	// y usamos como "ahora" ese mismo instante: aún podría llegar otra misión de Dughan.
	var evs []model.Event
	for _, e := range c.Events {
		if e.T <= 1791106700 {
			evs = append(evs, e)
		}
	}
	res := Build(evs, 1791106700, DefaultOptions())
	for _, g := range res.Groups {
		if g.Kind == Chain {
			t.Fatalf("la cadena no debería cerrarse todavía")
		}
	}
	if len(res.OpenChains) != 1 {
		t.Fatalf("debería haber una cadena abierta")
	}
}

func TestLooseFlushOnZoneChange(t *testing.T) {
	mk := func(id int64, t int64, zone string) []model.Event {
		return []model.Event{
			{T: t, Type: model.EvQuestAccept, ID: id, Title: "Q", NPC: "A" + zone, Zone: zone},
			{T: t + 10, Type: model.EvQuestTurnin, ID: id, NPC: "B" + zone, Zone: zone},
		}
	}
	var evs []model.Event
	evs = append(evs, mk(1, 1000, "Elwynn")...)
	evs = append(evs, mk(2, 2000, "Elwynn")...)
	evs = append(evs, mk(3, 3000, "Páramos")...)
	res := Build(evs, 10000, DefaultOptions())
	if len(res.Groups) != 1 || len(res.Groups[0].Quests) != 2 {
		t.Fatalf("al cambiar de zona se cierra el grupo de Elwynn: %+v", res.Groups)
	}
	if len(res.PendingLoose) != 1 || res.PendingLoose[0].ID != 3 {
		t.Fatalf("la de Páramos queda pendiente")
	}
}
