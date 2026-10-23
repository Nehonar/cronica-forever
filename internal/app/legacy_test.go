package app

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
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

func TestDeleteAllStartsFromScratch(t *testing.T) {
	repo := t.TempDir()
	// La muestra con otra clave: el personaje «-Demo» no se borra nunca.
	src, _ := os.ReadFile("../../samples/Cronica.lua")
	sv := filepath.Join(repo, "Cronica.lua")
	os.WriteFile(sv, []byte(strings.ReplaceAll(string(src), "Tobias-Demo", "Tobias-Prueba")), 0o644)
	sheet, _ := os.ReadFile("../../samples/Tobias-Demo.json")
	os.MkdirAll(filepath.Join(repo, "personajes"), 0o755)
	os.WriteFile(filepath.Join(repo, "personajes", "Tobias-Prueba.json"), []byte(strings.ReplaceAll(string(sheet), "Tobias-Demo", "Tobias-Prueba")), 0o644)
	os.WriteFile(filepath.Join(repo, "personajes", "Tobias-Demo.json"), sheet, 0o644)
	r := &Runner{Cfg: Config{SavedVariables: sv, Repo: repo}, Narrator: narrate.Fake{}, Out: io.Discard}
	if res, err := r.Process(context.Background()); err != nil || res.NewStories == 0 {
		t.Fatalf("%+v %v", res, err)
	}
	paths := store.Paths{Repo: repo}
	if n, err := paths.DeleteAll(4102444800); err != nil || n != 1 { // «ahora» en el futuro
		t.Fatalf("n=%d err=%v", n, err)
	}
	res, err := r.Process(context.Background())
	if err != nil || res.NewStories != 0 || len(res.NeedsBackstory) != 0 {
		t.Fatalf("tras borrar todo no se vuelve a narrar ni a pedir historia: %+v %v", res, err)
	}
	// Sin jugar desde el borrado, el personaje no vuelve a aparecer.
	if _, err := os.Stat(filepath.Join(paths.DataDir(), "Tobias-Prueba.json")); !os.IsNotExist(err) {
		t.Fatal("el personaje borrado no debe volver a aparecer hasta que se juegue")
	}
	// Lo que una versión anterior volvió a crear vacío se quita en la siguiente pasada.
	os.WriteFile(filepath.Join(paths.DataDir(), "Tobias-Prueba.json"), []byte(`{"key":"Tobias-Prueba"}`), 0o644)
	os.WriteFile(filepath.Join(paths.SheetsDir(), "Tobias-Prueba.json"), []byte(`{"key":"Tobias-Prueba","name":"Tobias"}`), 0o644)
	r.Process(context.Background())
	if _, err := os.Stat(filepath.Join(paths.DataDir(), "Tobias-Prueba.json")); !os.IsNotExist(err) {
		t.Fatal("debe quitarse la página que se volvió a crear")
	}
	if _, err := os.Stat(filepath.Join(paths.SheetsDir(), "Tobias-Demo.json")); err != nil {
		t.Fatal("la ficha de demostración no se borra")
	}
}
