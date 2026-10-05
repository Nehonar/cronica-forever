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

func TestProcessIsIncremental(t *testing.T) {
	repo := t.TempDir()
	addon := filepath.Join(repo, "addon", "CronicaTextos.lua")
	r := &Runner{
		Cfg:      Config{SavedVariables: "../../samples/Cronica.lua", Repo: repo, AddonTexts: addon},
		Narrator: narrate.Fake{},
		Out:      io.Discard,
	}
	res, err := r.Process(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.NewStories != 3 {
		t.Fatalf("primera pasada: %d relatos", res.NewStories)
	}
	res, err = r.Process(context.Background())
	if err != nil || res.NewStories != 0 {
		t.Fatalf("segunda pasada no debería narrar nada: %+v %v", res, err)
	}
	doc, err := store.Paths{Repo: repo}.LoadDoc("Nehonar-Demo")
	if err != nil || len(doc.Stories) != 3 || doc.Stats.QuestsDone != 9 || doc.Stats.Deaths != 1 {
		t.Fatalf("documento: %+v %v", doc.Stats, err)
	}
	b, err := os.ReadFile(addon)
	if err != nil || !strings.Contains(string(b), "CronicaTextos = {") {
		t.Fatalf("textos del addon: %v", err)
	}
}

func TestNewCharacterInheritsSheetByName(t *testing.T) {
	repo := t.TempDir()
	p := store.Paths{Repo: repo}
	os.MkdirAll(p.SheetsDir(), 0o755)
	os.WriteFile(filepath.Join(p.SheetsDir(), "Nehonar-Demo.json"), []byte(`{"key":"Nehonar-Demo","name":"Nehonar Lionhart","backstory":"Hijo del puerto."}`), 0o644)
	s, created, err := p.LoadSheet("Nehonar-Reino", "Nehonar", "Humano", "Paladín")
	if err != nil || !created || s.Backstory != "Hijo del puerto." || s.Key != "Nehonar-Reino" {
		t.Fatalf("%+v %v %v", s, created, err)
	}
}
