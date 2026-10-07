package app

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/Nehonar/cronica-forever/internal/narrate"
	"github.com/Nehonar/cronica-forever/internal/store"
)

// Antes el addon usaba «Nombre-Reino»; con el apellido, la crónica pasa al personaje
// con apellido y los relatos que mezclaban dos personajes se quitan.
func TestLegacyCharacterMerges(t *testing.T) {
	repo := t.TempDir()
	paths := store.Paths{Repo: repo}
	sheet := narrate.Sheet{Key: "Nehonar-Reino", Name: "Nehonar Lionhart", Race: "Humano", Class: "Paladín", Backstory: "Picaba piedra."}
	if err := paths.SaveSheet(sheet); err != nil {
		t.Fatal(err)
	}
	doc := &store.Doc{Key: "Nehonar-Reino", Character: sheet, Stories: []store.Story{
		{ID: "mezcla", Title: "Mezcla", LevelFrom: 10, LevelTo: 3, Quests: []store.QuestRef{{ID: 7}}},
		{ID: "bueno", Title: "Bueno", LevelFrom: 13, LevelTo: 14, Quests: []store.QuestRef{{ID: 8}}},
	}}
	if err := paths.SaveDoc(doc); err != nil {
		t.Fatal(err)
	}
	sv := filepath.Join(repo, "Cronica.lua")
	os.WriteFile(sv, []byte(`CronicaDB = { characters = {
  ["Nehonar-Reino"] = { name = "Nehonar", race = "Humano", class = "Paladín", level = 16, events = {} },
  ["NehonarLionheart-Reino"] = { name = "Nehonar", surname = "Lionheart", race = "Humano", class = "Paladín", level = 16, events = {} },
  ["NehonarOtro-Reino"] = { name = "Nehonar", surname = "Otro", race = "Humano", class = "Paladín", level = 5, events = {} },
} }`), 0o644)
	r := &Runner{Cfg: Config{SavedVariables: sv, Repo: repo}, Narrator: narrate.Fake{}, Out: io.Discard}
	if _, err := r.Process(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, _ := paths.LoadDoc("NehonarLionheart-Reino")
	if len(got.Stories) != 1 || got.Stories[0].ID != "bueno" || got.Character.Backstory == "" {
		t.Fatalf("la crónica debe pasar al de más nivel, sin el relato mezclado: %+v", got)
	}
	if paths.MergedInto("Nehonar-Reino") != "NehonarLionheart-Reino" {
		t.Fatal("la clave antigua debe quedar marcada")
	}
	if _, err := os.Stat(filepath.Join(repo, "docs", "data", "Nehonar-Reino.json")); err == nil {
		t.Fatal("la crónica antigua ya no debe publicarse")
	}
	other, _ := paths.LoadDoc("NehonarOtro-Reino")
	if other.Character.Backstory != "" {
		t.Fatal("el otro Nehonar es otro personaje: sin historia")
	}
	// Quitar un relato a mano.
	if err := paths.RemoveStory("NehonarLionheart-Reino", "bueno"); err != nil {
		t.Fatal(err)
	}
	got, _ = paths.LoadDoc("NehonarLionheart-Reino")
	if len(got.Stories) != 0 || len(got.Removed) != 2 {
		t.Fatalf("relatos=%d quitadas=%v", len(got.Stories), got.Removed)
	}
}
