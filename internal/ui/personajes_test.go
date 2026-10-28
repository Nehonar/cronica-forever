package ui

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Nehonar/cronica-forever/internal/app"
	"github.com/Nehonar/cronica-forever/internal/narrate"
	"github.com/Nehonar/cronica-forever/internal/store"
)

func TestDeletedCharactersAreNotListed(t *testing.T) {
	repo := t.TempDir()
	src, _ := os.ReadFile("../../samples/Cronica.lua")
	sv := filepath.Join(repo, "Cronica.lua")
	os.WriteFile(sv, []byte(strings.ReplaceAll(string(src), "Tobias-Demo", "Tobias-Reino")), 0o644)
	r := &app.Runner{Cfg: app.Config{SavedVariables: sv, Repo: repo}, Narrator: narrate.Fake{}, Out: io.Discard}
	s := &Server{Runner: r}
	list, _, err := s.characters()
	if err != nil || len(list) != 1 {
		t.Fatalf("antes de borrar: %d %v", len(list), err)
	}
	if _, err := (store.Paths{Repo: repo}).DeleteAll(4102444800); err != nil {
		t.Fatal(err)
	}
	if list, _, _ = s.characters(); len(list) != 0 {
		t.Fatalf("un personaje borrado y sin jugar no debe salir: %+v", list)
	}
}
