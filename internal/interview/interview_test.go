package interview

import (
	"strings"
	"testing"
)

func TestParseReplyWithSheet(t *testing.T) {
	raw := "Aquí tienes su historia. Puedes guardarla o pedirme cambios.\n\nFICHA:\n```json\n{\"name\":\"Brom Piedrahonda\",\"epithet\":\"El de la mina\",\"voice\":\"Seco.\",\"backstory\":\"Uno.\\n\\nDos.\",\"threads\":[\"a\",\"b\",\"c\"],\"motto\":\"Más hondo\"}\n```"
	r := ParseReply(raw)
	if r.Sheet == nil || r.Sheet.Name != "Brom Piedrahonda" || len(r.Sheet.Threads) != 3 {
		t.Fatalf("ficha: %+v", r.Sheet)
	}
	if strings.Contains(r.Text, "```") || !strings.HasPrefix(r.Text, "Aquí tienes") {
		t.Fatalf("texto: %q", r.Text)
	}
}

func TestParseReplyQuestion(t *testing.T) {
	r := ParseReply("Cronista: ¿De dónde viene tu enano?")
	if r.Sheet != nil || r.Text != "¿De dónde viene tu enano?" {
		t.Fatalf("%+v", r)
	}
}

func TestPromptIncludesFactsAndHistory(t *testing.T) {
	p := Prompt(Facts{Name: "Brom", Race: "Enano", Class: "Cazador", Zones: []string{"Dun Morogh"}, QuestsDone: 4},
		[]Message{{Role: "cronista", Content: "¿Cómo se le conoce?"}, {Role: "jugador", Content: "Brom Piedrahonda"}})
	for _, want := range []string{"Enano", "Cazador", "Dun Morogh", "Jugador: Brom Piedrahonda", "Cronista: ¿Cómo"} {
		if !strings.Contains(p, want) {
			t.Errorf("falta %q", want)
		}
	}
}
