package narrate

import (
	"context"
	"strings"
	"testing"

	"github.com/Nehonar/cronica-forever/internal/group"
)

func TestParse(t *testing.T) {
	raw := "**TÍTULO:** «Lo que asoma bajo la abadía»\n\nPrimer párrafo.\n\nSegundo.\n\nRESUMEN: Limpia la Cresta del Eco y parte hacia Villadorada."
	st, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if st.Title != "Lo que asoma bajo la abadía" {
		t.Errorf("título = %q", st.Title)
	}
	if st.Text != "Primer párrafo.\n\nSegundo." {
		t.Errorf("texto = %q", st.Text)
	}
	if !strings.HasPrefix(st.Summary, "Limpia") {
		t.Errorf("resumen = %q", st.Summary)
	}
}

func TestPromptHasNoMechanicsLeak(t *testing.T) {
	g := group.Group{Kind: group.Chain, Zone: "Bosque de Elwynn", LevelFrom: 1, LevelTo: 5,
		Quests: []group.Quest{{ID: 1, Title: "A", Giver: "X", Text: "t"}, {ID: 2, Title: "B", Giver: "X"}}}
	_, p := BuildPrompt(Sheet{Name: "Nehonar"}, g, nil)
	if strings.Contains(p, "nivel 1") || strings.Contains(p, "nivel 5") {
		t.Error("el prompt no debe dar números de nivel")
	}
	if !strings.Contains(p, "«A»") || !strings.Contains(p, "«B»") {
		t.Error("faltan misiones en el prompt")
	}
}

func TestNarrateWithFake(t *testing.T) {
	g := group.Group{Kind: group.Loose, Zone: "Elwynn", Quests: []group.Quest{{ID: 1, Title: "A"}}}
	st, err := Narrate(context.Background(), Fake{}, Sheet{Name: "N"}, g, nil)
	if err != nil || st.Title != "Relato de prueba" || st.Summary == "" {
		t.Fatalf("%+v %v", st, err)
	}
}
