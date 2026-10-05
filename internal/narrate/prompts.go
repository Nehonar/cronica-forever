package narrate

import (
	"fmt"
	"strings"

	"github.com/Nehonar/cronica-forever/internal/group"
)

const systemPrompt = `Eres el cronista de un personaje de World of Warcraft: Forever (el Azeroth de WoW Classic). Conviertes lo que el jugador ha hecho en el juego en relatos breves de su historia personal.

Reglas que nunca rompes:
- Español de España. Tercera persona y tiempo pasado. Prosa clara, con ritmo, sin adornos vacíos.
- Fidelidad: los hechos (qué encargo, quién lo dio, dónde, qué se consiguió) salen SOLO de los datos que te dan. Los textos de las misiones son la fuente de la verdad. No inventes personajes con nombre, combates concretos, muertes ni giros que no estén en los datos. Sí puedes añadir ambiente, sensaciones, pensamientos del personaje y algún diálogo breve con los PNJ que aparecen.
- Sin mecánicas de juego: nada de niveles, experiencia, puntos, talentos, objetivos numéricos ni interfaz. Subir de nivel es crecer; una muerte es una derrota de la que se levanta; una pieza de equipo es un objeto con peso en la historia.
- Sin destripes: no adelantes nada que no esté en los datos ni insinúes lo que vendrá en misiones futuras.
- Respeta la voz y el trasfondo del personaje.
- Responde exactamente en el formato pedido, sin nada antes ni después.`

// BuildPrompt devuelve el prompt de sistema y el de usuario para un grupo.
func BuildPrompt(sheet Sheet, g group.Group, prev []Previous) (string, string) {
	var b strings.Builder
	fmt.Fprintf(&b, "PERSONAJE: %s", sheet.Name)
	if sheet.Epithet != "" {
		fmt.Fprintf(&b, ", «%s»", sheet.Epithet)
	}
	fmt.Fprintf(&b, ". %s %s.\n", orDefault(sheet.Race, "Humano"), strings.ToLower(orDefault(sheet.Class, "aventurero")))
	if sheet.Voice != "" {
		fmt.Fprintf(&b, "VOZ: %s\n", sheet.Voice)
	}
	if sheet.Motto != "" {
		fmt.Fprintf(&b, "LEMA: %s\n", sheet.Motto)
	}
	if sheet.Backstory != "" {
		fmt.Fprintf(&b, "\nTRASFONDO:\n%s\n", sheet.Backstory)
	}
	if len(sheet.Threads) > 0 {
		fmt.Fprintf(&b, "\nHILOS ABIERTOS (úsalos solo si encajan de forma natural, no los resuelvas):\n- %s\n", strings.Join(sheet.Threads, "\n- "))
	}
	if len(prev) > 0 {
		b.WriteString("\nRELATOS ANTERIORES (para continuidad; no los repitas):\n")
		for _, p := range prev {
			fmt.Fprintf(&b, "- %s: %s\n", p.Title, p.Summary)
		}
	}

	b.WriteString("\n")
	ctx := groupContext(g)
	switch g.Kind {
	case group.Chain:
		fmt.Fprintf(&b, "TAREA: escribe el relato de una cadena de misiones completa, en orden, como una sola historia con principio y final. Entre 250 y 380 palabras.\n%s\nMISIONES DE LA CADENA:\n", ctx)
		writeQuests(&b, g.Quests)
	case group.Loose:
		fmt.Fprintf(&b, "TAREA: estas misiones no tienen relación entre sí. Escribe un único relato breve que las una con un hilo temático (lo que dicen de la zona, de su gente o del propio personaje), sin forzar una trama común que no existe. Entre 180 y 280 palabras.\n%s\nMISIONES:\n", ctx)
		writeQuests(&b, g.Quests)
	case group.Equip:
		it := g.Items[0]
		fmt.Fprintf(&b, "TAREA: el personaje se acaba de poner por primera vez esta pieza de equipo. Escribe un momento breve e íntimo sobre ello (qué significa para él, cómo la siente). Entre 50 y 90 palabras. No inventes cómo la consiguió.\n%s\nPIEZA: %s (una pieza %s; se lleva en: %s). La calidad describe lo valiosa que es, no su color.\n", ctx, it.Item, quality(it.Quality), slotName(it.Slot))
	}
	b.WriteString(`
FORMATO DE RESPUESTA:
TÍTULO: <título, máximo 7 palabras>

<texto, párrafos separados por una línea en blanco>

RESUMEN: <una frase que resuma el relato para recordarlo más adelante>`)
	return systemPrompt, b.String()
}

func groupContext(g group.Group) string {
	var parts []string
	if g.Zone != "" {
		z := "Zona: " + g.Zone
		if len(g.Subzones) > 0 {
			z += " (" + strings.Join(g.Subzones, ", ") + ")"
		}
		parts = append(parts, z)
	}
	if g.LevelTo > g.LevelFrom && g.Kind != group.Equip {
		parts = append(parts, "Durante este tramo el personaje creció y se hizo más fuerte (no lo cuentes como niveles).")
	}
	if g.Deaths == 1 {
		parts = append(parts, "Durante este tramo cayó derrotado una vez y se levantó.")
	} else if g.Deaths > 1 {
		parts = append(parts, fmt.Sprintf("Durante este tramo cayó derrotado %d veces y siguió adelante.", g.Deaths))
	}
	if g.Kind != group.Equip {
		for _, it := range g.Items {
			parts = append(parts, fmt.Sprintf("Durante este tramo empezó a usar: %s (%s).", it.Item, quality(it.Quality)))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "CONTEXTO:\n- " + strings.Join(parts, "\n- ") + "\n"
}

func writeQuests(b *strings.Builder, qs []group.Quest) {
	for i, q := range qs {
		fmt.Fprintf(b, "\n%d. «%s»\n", i+1, q.Title)
		if q.Giver != "" {
			fmt.Fprintf(b, "   La da: %s\n", q.Giver)
		}
		if q.TurninNPC != "" && q.TurninNPC != q.Giver {
			fmt.Fprintf(b, "   Se entrega a: %s\n", q.TurninNPC)
		}
		if q.Subzone != "" {
			fmt.Fprintf(b, "   Lugar: %s\n", q.Subzone)
		}
		if q.Text != "" {
			fmt.Fprintf(b, "   Texto de la misión: %s\n", oneLine(q.Text))
		}
		if q.Objectives != "" {
			fmt.Fprintf(b, "   Encargo: %s\n", oneLine(q.Objectives))
		}
		if q.Reward != "" {
			fmt.Fprintf(b, "   Al entregarla le dicen: %s\n", oneLine(q.Reward))
		}
	}
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

func quality(q int) string {
	switch q {
	case 3:
		return "de buena factura, poco común"
	case 4:
		return "excepcional, de las que pocos llevan"
	case 5:
		return "legendaria"
	}
	return "corriente"
}

func slotName(s string) string {
	names := map[string]string{
		"HEAD": "cabeza", "NECK": "cuello", "SHOULDER": "hombros", "CHEST": "pecho", "WAIST": "cintura",
		"LEGS": "piernas", "FEET": "pies", "WRIST": "muñecas", "HANDS": "manos", "FINGER": "dedo",
		"TRINKET": "abalorio", "BACK": "espalda", "MAINHAND": "mano derecha", "OFFHAND": "mano izquierda",
		"SHIELD": "escudo", "RANGED": "a distancia", "TWOHAND": "arma de dos manos",
	}
	if n, ok := names[strings.ToUpper(s)]; ok {
		return n
	}
	return strings.ToLower(s)
}
