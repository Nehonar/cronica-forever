package narrate

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// QuestInfo es una misión recién aceptada.
type QuestInfo struct {
	Title      string
	Giver      string
	Zone       string
	Subzone    string
	Text       string
	Objectives string
}

// Encargo narra el momento de recibir una misión, para leerlo en la app al
// aceptarla. Devuelve un título y un texto breve; los objetivos se muestran
// aparte, tal cual.
func Encargo(ctx context.Context, n Narrator, sheet Sheet, q QuestInfo) (title, text string, err error) {
	var b strings.Builder
	writeSheet(&b, sheet)
	fmt.Fprintf(&b, "\nTAREA: el personaje acaba de aceptar esta misión. Cuenta el momento en que se la encargan, como el comienzo de una escena: quién se la pide, qué le cuenta y por qué importa. Entre 90 y 140 palabras.\n")
	b.WriteString("- Fiel al texto de la misión: no añadas hechos, lugares ni personajes que no estén ahí, y no adelantes cómo acabará.\n")
	b.WriteString("- Que al terminar de leerlo quede claro qué tiene que hacer y adónde ir, según el texto (sin cifras de objetivos: esas se muestran aparte).\n")
	b.WriteString("- Puedes incluir una o dos frases del PNJ en estilo directo, adaptadas del texto de la misión.\n\n")
	fmt.Fprintf(&b, "MISIÓN: «%s»\n", q.Title)
	if q.Giver != "" {
		fmt.Fprintf(&b, "La da: %s\n", q.Giver)
	}
	if q.Zone != "" {
		place := q.Zone
		if q.Subzone != "" {
			place = q.Subzone + ", " + q.Zone
		}
		fmt.Fprintf(&b, "Dónde está el personaje: %s\n", place)
	}
	if q.Text != "" {
		fmt.Fprintf(&b, "Texto de la misión: %s\n", oneLine(q.Text))
	}
	if q.Objectives != "" {
		fmt.Fprintf(&b, "Objetivos: %s\n", oneLine(q.Objectives))
	}
	b.WriteString("\nFORMATO DE RESPUESTA:\nTÍTULO: <título, máximo 6 palabras>\n\n<texto>")
	raw, err := n.Generate(ctx, systemPrompt, b.String())
	if err != nil {
		return "", "", err
	}
	st, err := Parse(raw)
	if err != nil {
		return "", "", err
	}
	if st.Title == "" {
		st.Title = q.Title
	}
	return st.Title, st.Text, nil
}

// PhraseCategories son los momentos en los que el personaje puede decir algo.
var PhraseCategories = map[string]string{
	"nivel":    "acaba de crecer en poder (sube de nivel)",
	"muerte":   "acaba de caer derrotado y se levanta",
	"zona":     "llega por primera vez a una tierra nueva",
	"aceptar":  "acepta un encargo",
	"entregar": "termina un encargo",
	"equipo":   "se pone una pieza de equipo nueva y valiosa; usa {objeto} para el nombre de la pieza",
	"descanso": "descansa en una posada o una ciudad",
}

var reJSON = regexp.MustCompile("(?s)\\{.*\\}")

// Phrases genera la baraja de frases del personaje: frases cortas, en primera
// persona y con su voz, para cada momento.
func Phrases(ctx context.Context, n Narrator, sheet Sheet) (map[string][]string, error) {
	var b strings.Builder
	writeSheet(&b, sheet)
	b.WriteString("\nTAREA: escribe frases cortas que este personaje dice o piensa en voz alta en distintos momentos de su aventura. Se mostrarán en el juego y quizá las lean otros jugadores, así que:\n")
	b.WriteString("- Primera persona, con su voz y su carácter. Máximo 90 caracteres cada una.\n")
	b.WriteString("- Variadas: que no empiecen igual ni repitan ideas. Algunas serias, alguna con humor si encaja con él.\n")
	b.WriteString("- Sin mecánicas de juego (nada de niveles, experiencia, puntos, misiones como palabra técnica) y sin nombrar lugares o personajes concretos del juego, porque se dirán en cualquier sitio.\n")
	b.WriteString("- Pueden aludir a su pasado y a sus hilos abiertos, sin destripar nada.\n\n")
	b.WriteString("Momentos (12 frases para cada uno):\n")
	for _, cat := range []string{"nivel", "muerte", "zona", "aceptar", "entregar", "equipo", "descanso"} {
		fmt.Fprintf(&b, "- %s: %s\n", cat, PhraseCategories[cat])
	}
	b.WriteString("\nResponde SOLO con un objeto JSON cuyas claves son esos momentos y cuyos valores son listas de frases, sin texto antes ni después.")
	raw, err := n.Generate(ctx, systemPrompt, b.String())
	if err != nil {
		return nil, err
	}
	m := reJSON.FindString(raw)
	if m == "" {
		return nil, fmt.Errorf("la respuesta no contiene las frases")
	}
	var out map[string][]string
	if err := json.Unmarshal([]byte(m), &out); err != nil {
		return nil, fmt.Errorf("frases mal formadas: %w", err)
	}
	clean := map[string][]string{}
	for cat := range PhraseCategories {
		for _, p := range out[cat] {
			p = strings.TrimSpace(strings.Trim(p, "«»\""))
			if p != "" && len([]rune(p)) <= 140 {
				clean[cat] = append(clean[cat], p)
			}
		}
	}
	if len(clean) < 4 {
		return nil, fmt.Errorf("faltan frases en la respuesta")
	}
	return clean, nil
}

// writeSheet escribe la ficha del personaje al principio de un prompt.
func writeSheet(b *strings.Builder, sheet Sheet) {
	fmt.Fprintf(b, "PERSONAJE: %s", sheet.Name)
	if sheet.Epithet != "" {
		fmt.Fprintf(b, ", «%s»", sheet.Epithet)
	}
	fmt.Fprintf(b, ". %s %s.\n", orDefault(sheet.Race, "Humano"), strings.ToLower(orDefault(sheet.Class, "aventurero")))
	if sheet.Voice != "" {
		fmt.Fprintf(b, "VOZ: %s\n", sheet.Voice)
	}
	if sheet.Motto != "" {
		fmt.Fprintf(b, "LEMA: %s\n", sheet.Motto)
	}
	if sheet.Backstory != "" {
		fmt.Fprintf(b, "\nTRASFONDO:\n%s\n", sheet.Backstory)
	}
	if len(sheet.Threads) > 0 {
		fmt.Fprintf(b, "\nHILOS ABIERTOS:\n- %s\n", strings.Join(sheet.Threads, "\n- "))
	}
}
