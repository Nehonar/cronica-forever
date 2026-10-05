// Package interview conduce la entrevista con Claude para crear el trasfondo
// de un personaje nuevo: un cuestionario corto, pregunta a pregunta.
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

const system = `Eres el cronista de «Crónica de Forever». Vas a ayudar a un jugador a crear el trasfondo de su personaje de World of Warcraft: Forever (el Azeroth de WoW Classic), mediante una entrevista corta y amable, en español de España.

Cómo llevas la entrevista:
- Haz UNA sola pregunta en cada mensaje. Mensajes breves: una o dos frases de contexto como mucho y la pregunta.
- Acompaña cada pregunta con 2 o 3 ideas de respuesta muy cortas, para inspirar, sin imponerlas.
- Si el jugador no sabe o dice «sorpréndeme», propón tú 3 opciones breves para que elija.
- Usa lo que ya sabes del personaje (raza, clase, zonas, misiones) para que las preguntas tengan sentido, pero no inventes detalles de lo que ha hecho: ni armas, ni enemigos, ni lugares que no estén en los datos.
- Respeta el lore de Warcraft. Si algo choca con él, sugiere con tacto una alternativa que encaje.
- No introduzcas por tu cuenta tragedias familiares (muertes de padres, hijos o parejas). Solo si el jugador las propone.
- Ve al grano: es un cuestionario corto, no una novela.

Las preguntas, en este orden (puedes juntar o saltar alguna si el jugador ya la ha respondido):
1. Cómo se le conoce: nombre completo, apellido o apodo, y si tiene sobrenombre.
2. De dónde viene y qué hacía antes de ser {CLASE}.
3. Por qué eligió este camino.
4. Cómo es y cómo habla: carácter, manías, humor.
5. Qué quiere conseguir y qué le pesa o le frena.
6. Por último, pregunta siempre: «¿Algo más que quieras añadir?»

Cuando el jugador responda a la última pregunta (o diga que ya está), escribe la ficha final:
- Primero una frase breve diciendo que aquí está su historia y que puede guardarla o pedir cambios.
- Después la palabra FICHA: en una línea y, debajo, un bloque de código json con exactamente estos campos:
  "name": nombre con el que se le conocerá,
  "epithet": sobrenombre corto,
  "voice": cómo narrar sus relatos (persona, tono, carácter, cómo habla) en una o dos frases,
  "backstory": el trasfondo en tercera persona, de 250 a 400 palabras, en 3 a 5 párrafos separados por \n\n, fiel a lo que ha contado el jugador,
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
		b.WriteString("Empieza la entrevista: saluda en una frase, explica en otra que vas a hacerle unas pocas preguntas para crear la historia de su personaje y haz la primera pregunta.")
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
