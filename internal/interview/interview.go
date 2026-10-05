// Package interview conduce la entrevista con Claude para crear el trasfondo
// de un personaje nuevo: un cuestionario corto que se contesta de una vez.
package interview

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/Nehonar/cronica-forever/internal/narrate"
)

// Message es un turno de la conversación.
type Message struct {
	Role    string `json:"role"` // "cronista" o "jugador"
	Content string `json:"content"`
}

// Facts es lo que ya se sabe del personaje por el addon.
type Facts struct {
	Key        string
	Name       string
	Race       string
	Class      string
	Level      int
	Zones      []string
	QuestsDone int
	Quests     []string // algunos títulos de misiones ya hechas
}

// Reply es la respuesta del cronista; Sheet llega cuando la entrevista termina.
type Reply struct {
	Text  string         `json:"text"`
	Sheet *narrate.Sheet `json:"sheet,omitempty"`
}

const system = `Eres el cronista de «Crónica de Forever». Vas a ayudar a un jugador a crear el trasfondo de su personaje de World of Warcraft: Forever (el Azeroth de WoW Classic), en español de España.

El jugador ha contestado de una vez a un cuestionario corto (de dónde viene, cómo es, qué busca y algo más si quiere). Con sus respuestas escribe directamente la ficha, sin hacerle más preguntas.

Reglas:
- Sé fiel a lo que ha contado. Si deja alguna respuesta en blanco o dice «sorpréndeme», complétala tú con algo sencillo y coherente con su raza y su clase {CLASE}.
- Usa lo que ya sabes del personaje (raza, clase, zonas, misiones), pero no inventes hechos de lo que ha vivido en el juego: ni armas, ni enemigos, ni lugares que no estén en los datos.
- Respeta el lore de Warcraft. Si algo choca con él, ajústalo con tacto y dilo en una frase.
- No introduzcas por tu cuenta tragedias familiares (muertes de padres, hijos o parejas). Solo si el jugador las propone.

Formato de tu respuesta:
- Primero una o dos frases breves diciendo que aquí está su historia y que puede guardarla o pedir cambios.
- Después la palabra FICHA: en una línea y, debajo, un bloque de código json con exactamente estos campos:
  "name": nombre con el que se le conocerá,
  "epithet": sobrenombre corto,
  "voice": cómo narrar sus relatos (persona, tono, carácter, cómo habla) en una o dos frases,
  "backstory": el trasfondo en tercera persona, de 250 a 400 palabras, en 3 a 5 párrafos separados por \n\n,
  "threads": lista de 3 o 4 hilos abiertos para el futuro (frases cortas),
  "motto": un lema breve.
Si después el jugador pide cambios, aplícalos y vuelve a escribir la ficha completa con el mismo formato.`

// Turn genera la siguiente respuesta del cronista.
func Turn(ctx context.Context, n narrate.Narrator, f Facts, history []Message) (Reply, error) {
	sys := strings.ReplaceAll(system, "{CLASE}", strings.ToLower(orDefault(f.Class, "aventurero")))
	raw, err := n.Generate(ctx, sys, Prompt(f, history))
	if err != nil {
		return Reply{}, err
	}
	return ParseReply(raw), nil
}

// Prompt construye el mensaje con los datos del personaje y la conversación.
func Prompt(f Facts, history []Message) string {
	var b strings.Builder
	fmt.Fprintf(&b, "LO QUE YA SABEMOS DEL PERSONAJE (por el juego):\n- Nombre en el juego: %s\n- Raza: %s\n- Clase: %s\n", f.Name, orDefault(f.Race, "desconocida"), orDefault(f.Class, "desconocida"))
	if f.Level > 0 {
		fmt.Fprintf(&b, "- Nivel actual: %d (no lo menciones como número en la historia)\n", f.Level)
	}
	if len(f.Zones) > 0 {
		fmt.Fprintf(&b, "- Zonas que ha recorrido: %s\n", strings.Join(f.Zones, ", "))
	}
	if f.QuestsDone > 0 {
		fmt.Fprintf(&b, "- Misiones completadas: %d", f.QuestsDone)
		if len(f.Quests) > 0 {
			fmt.Fprintf(&b, " (por ejemplo: %s)", strings.Join(f.Quests, "; "))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	if len(history) == 0 {
		b.WriteString("El jugador aún no ha contestado nada: inventa tú un trasfondo sencillo y coherente y escribe la ficha.")
		return b.String()
	}
	b.WriteString("CONVERSACIÓN HASTA AHORA:\n")
	for _, m := range history {
		who := "Cronista"
		if m.Role == "jugador" {
			who = "Jugador"
		}
		fmt.Fprintf(&b, "\n%s: %s\n", who, strings.TrimSpace(m.Content))
	}
	b.WriteString("\nResponde ahora como Cronista al último mensaje del jugador, siguiendo las instrucciones. Escribe solo tu mensaje, sin la etiqueta «Cronista:».")
	return b.String()
}

var reSheet = regexp.MustCompile("(?s)FICHA:?\\s*\\**\\s*```(?:json)?\\s*(\\{.*?\\})\\s*```")
var reBareJSON = regexp.MustCompile("(?s)```json\\s*(\\{.*?\\})\\s*```")

// ParseReply separa el texto visible de la ficha final, si la hay.
func ParseReply(raw string) Reply {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, "\r", ""))
	raw = strings.TrimPrefix(raw, "Cronista:")
	m := reSheet.FindStringSubmatchIndex(raw)
	if m == nil {
		m = reBareJSON.FindStringSubmatchIndex(raw)
	}
	if m == nil {
		return Reply{Text: strings.TrimSpace(raw)}
	}
	var sh narrate.Sheet
	if err := json.Unmarshal([]byte(raw[m[2]:m[3]]), &sh); err != nil || sh.Backstory == "" {
		return Reply{Text: strings.TrimSpace(raw)}
	}
	text := strings.TrimSpace(raw[:m[0]] + raw[m[1]:])
	return Reply{Text: text, Sheet: &sh}
}

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}
