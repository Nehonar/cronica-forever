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

func TestWarnsWhenClaudeIsLoggedOut(t *testing.T) {
	dir := t.TempDir()
	fakeClaude := filepath.Join(dir, "claude")
	// Simula «claude auth status» sin sesión y «claude -p» fallando.
	os.WriteFile(fakeClaude, []byte("#!/bin/sh\nif [ \"$1\" = auth ]; then echo '{\"loggedIn\": false}'; exit 1; fi\necho 'Not logged in' >&2; exit 1\n"), 0o755)
	var notices []string
	addon := filepath.Join(dir, "CronicaTextos.lua")
	r := &Runner{
		Cfg:      Config{SavedVariables: "../../samples/Cronica.lua", Repo: dir, AddonTexts: addon},
		Narrator: narrate.ClaudeCLI{Command: fakeClaude},
		Out:      io.Discard,
		Notify:   func(_, m string) { notices = append(notices, m) },
	}
	if err := r.CheckClaude(context.Background()); err == nil {
		t.Fatal("debería detectar que no hay sesión")
	}
	res, err := r.Process(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.NewStories != 0 || res.Failed == 0 {
		t.Fatalf("no debería narrar nada: %+v", res)
	}
	if len(notices) != 1 {
		t.Fatalf("un solo aviso por hora, no %d", len(notices))
	}
	b, _ := os.ReadFile(addon)
	if !strings.Contains(string(b), `["ok"] = false`) || !strings.Contains(string(b), "claude") {
		t.Fatalf("el addon debe recibir el estado de error:\n%s", b)
	}
	// Lo no narrado queda para la próxima pasada.
	doc, _ := store.Paths{Repo: dir}.LoadDoc("Nehonar-Demo")
	if len(doc.Stories) != 0 {
		t.Fatal("no debe guardar relatos fallidos")
	}
}
