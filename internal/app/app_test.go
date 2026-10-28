package app

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Nehonar/cronica-forever/internal/narrate"
	"github.com/Nehonar/cronica-forever/internal/store"
	"github.com/Nehonar/cronica-forever/internal/voz"
)

// withDemoSheet copia la ficha de demostración al repositorio de prueba.
func withDemoSheet(t *testing.T, repo string) {
	t.Helper()
	b, err := os.ReadFile("../../samples/Tobias-Demo.json")
	if err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Join(repo, "personajes"), 0o755)
	os.WriteFile(filepath.Join(repo, "personajes", "Tobias-Demo.json"), b, 0o644)
}

func TestNewCharacterWaitsForBackstory(t *testing.T) {
	repo := t.TempDir()
	var notices int
	r := &Runner{
		Cfg:      Config{SavedVariables: "../../samples/Cronica.lua", Repo: repo},
		Narrator: narrate.Fake{},
		Out:      io.Discard,
		Notify:   func(_, _ string) { notices++ },
	}
	res, err := r.Process(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.NewStories != 0 || len(res.NeedsBackstory) != 1 || res.NeedsBackstory[0].Waiting != 1 {
		t.Fatalf("sin trasfondo no se narra: %+v", res)
	}
	r.Process(context.Background())
	if notices != 1 {
		t.Fatalf("debe avisar una sola vez, no %d", notices)
	}
	doc, _ := store.Paths{Repo: repo}.LoadDoc("Tobias-Demo")
	if doc.Pending.AwaitingBackstory != 1 || doc.Stats.QuestsDone == 0 {
		t.Fatalf("el progreso debe guardarse: %+v", doc.Pending)
	}
	// Al crear la historia, se narra todo lo pendiente.
	withDemoSheet(t, repo)
	res, _ = r.Process(context.Background())
	if res.NewStories != 1 {
		t.Fatalf("con trasfondo se narra lo pendiente: %+v", res)
	}
}

func TestProcessIsIncremental(t *testing.T) {
	repo := t.TempDir()
	withDemoSheet(t, repo)
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
	if res.NewStories != 1 {
		t.Fatalf("primera pasada: %d relatos", res.NewStories)
	}
	res, err = r.Process(context.Background())
	if err != nil || res.NewStories != 0 {
		t.Fatalf("segunda pasada no debería narrar nada: %+v %v", res, err)
	}
	doc, err := store.Paths{Repo: repo}.LoadDoc("Tobias-Demo")
	if err != nil || len(doc.Chapters) != 1 || doc.Current == nil || doc.Current.Quests != 5 || doc.Current.Points != 3 || doc.Current.Threshold != 8 || doc.Stats.QuestsDone != 12 || doc.Stats.Deaths != 1 {
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
	os.WriteFile(filepath.Join(p.SheetsDir(), "Tobias-Demo.json"), []byte(`{"key":"Tobias-Demo","name":"Tobias Brenn","backstory":"Hijo del molinero."}`), 0o644)
	s, created, err := p.LoadSheet("Tobias-Reino", "Tobias", "Humano", "Guerrero")
	if err != nil || !created || s.Backstory != "Hijo del molinero." || s.Key != "Tobias-Reino" {
		t.Fatalf("%+v %v %v", s, created, err)
	}
}

func TestWarnsWhenClaudeIsLoggedOut(t *testing.T) {
	dir := t.TempDir()
	fakeClaude := filepath.Join(dir, "claude")
	// Simula «claude auth status» sin sesión y «claude -p» fallando.
	os.WriteFile(fakeClaude, []byte("#!/bin/sh\nif [ \"$1\" = auth ]; then echo '{\"loggedIn\": false}'; exit 1; fi\necho 'Not logged in' >&2; exit 1\n"), 0o755)
	var notices []string
	withDemoSheet(t, dir)
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
	if !strings.Contains(string(b), `["ok"] = false`) || !strings.Contains(string(b), "Claude") {
		t.Fatalf("el addon debe recibir el estado de error:\n%s", b)
	}
	// Lo no narrado queda para la próxima pasada.
	doc, _ := store.Paths{Repo: dir}.LoadDoc("Tobias-Demo")
	if len(doc.Chapters) != 0 {
		t.Fatal("no debe guardar capítulos fallidos")
	}
}

func TestStoriesGetAVoice(t *testing.T) {
	var calls int
	azure := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Ocp-Apim-Subscription-Key") != "clave" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		calls++
		w.Write([]byte(strings.Repeat("\xff\xfb", 300)))
	}))
	defer azure.Close()
	repo := t.TempDir()
	withDemoSheet(t, repo)
	r := &Runner{
		Cfg:      Config{SavedVariables: "../../samples/Cronica.lua", Repo: repo},
		Narrator: narrate.Fake{},
		Out:      io.Discard,
		Voice:    voz.Client{Base: azure.URL},
	}
	// Sin voz configurada: la historia se escribe, sin audio.
	if res, err := r.Process(context.Background()); err != nil || res.NewStories != 1 || res.NewAudio != 0 {
		t.Fatalf("%+v %v", res, err)
	}
	// Con la voz puesta, se dobla lo que faltaba en la siguiente pasada.
	r.UpdateConfig(func(c *Config) { c.Voz = voz.Config{Active: true, Region: "westeurope", Key: "clave"} })
	res, err := r.Process(context.Background())
	if err != nil || res.NewAudio != 1 || calls != 1 {
		t.Fatalf("%+v %v (llamadas %d)", res, err, calls)
	}
	p := store.Paths{Repo: repo}
	doc, _ := p.LoadDoc("Tobias-Demo")
	ch := doc.Chapters[0]
	if ch.ID != "historia-1" || ch.Kind != "historia" || ch.Audio != "audio/Tobias-Demo/historia-1.mp3" || ch.AudioVoice != voz.DefaultVoice {
		t.Fatalf("historia: %+v", ch)
	}
	if fi, err := os.Stat(filepath.Join(repo, "docs", ch.Audio)); err != nil || fi.Size() != 600 {
		t.Fatalf("audio: %v", err)
	}
	// La copia pública no lleva horas.
	b, _ := os.ReadFile(filepath.Join(p.DataDir(), "Tobias-Demo.json"))
	if strings.Contains(string(b), "writtenTo") || strings.Contains(string(b), `"end"`) {
		t.Fatalf("la web no debe saber cuándo juegas:\n%s", b)
	}
	// No se vuelve a doblar ni a escribir.
	if res, _ := r.Process(context.Background()); res.NewAudio != 0 || res.NewStories != 0 || calls != 1 {
		t.Fatalf("%+v", res)
	}
	// Si se borra la historia, no se vuelve a escribir y su audio desaparece.
	if err := p.RemoveStory("Tobias-Demo", "historia-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repo, "docs", ch.Audio)); !os.IsNotExist(err) {
		t.Fatal("el audio debería borrarse")
	}
	if res, _ := r.Process(context.Background()); res.NewStories != 0 {
		t.Fatalf("no debe reescribir lo borrado: %+v", res)
	}
}

func TestTemplateSheetBySurname(t *testing.T) {
	p := store.Paths{Repo: t.TempDir()}
	os.MkdirAll(p.SheetsDir(), 0o755)
	os.WriteFile(filepath.Join(p.SheetsDir(), "NehonarBanthalas-Plantilla.json"), []byte(`{"key":"NehonarBanthalas-Plantilla","name":"Nehonar Banthalas","backstory":"De Anvilas."}`), 0o644)
	s, created, err := p.LoadSheet("NehonarBanthalas-Reino", "Nehonar Banthalas", "Cielonato", "Guerrero")
	if err != nil || !created || s.Backstory != "De Anvilas." || s.Key != "NehonarBanthalas-Reino" {
		t.Fatalf("%+v %v %v", s, created, err)
	}
	if n, _ := p.DeleteAll(1); n != 1 {
		t.Fatalf("la plantilla no se borra al empezar de cero (borrados %d)", n)
	}
	if _, err := os.Stat(filepath.Join(p.SheetsDir(), "NehonarBanthalas-Plantilla.json")); err != nil {
		t.Fatal(err)
	}
}
