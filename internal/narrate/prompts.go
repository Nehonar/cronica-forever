package narrate

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Nehonar/cronica-forever/internal/group"
	"github.com/Nehonar/cronica-forever/internal/model"
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
		fmt.Fprintf(&b, "TAREA: el personaje se acaba de poner por primera vez esta pieza de equipo. Escribe un momento breve e íntimo sobre ello: qué significa para él y, sobre todo, cómo la siente en el cuerpo al llevarla. Entre 50 y 90 palabras. No inventes cómo la consiguió.\n%s\n%s", ctx, itemLine(it))
	case group.Rep:
		st := g.Standing
		fmt.Fprintf(&b, "TAREA: el personaje acaba de ganarse un nuevo grado de confianza con una facción de Azeroth. Escribe un momento breve sobre ello, entre 90 y 150 palabras: quién es esa gente (usa el lore de Warcraft y la descripción oficial, sin inventar hechos que contradigan el canon), qué significa para él que ahora le vean así y cómo se lo hacen notar (un saludo distinto, una puerta que se abre, una mirada). No digas «reputación» ni nombres de rangos como si fueran números; tradúcelo a cómo le tratan.\n%s\nFACCIÓN: %s\nAHORA LE CONSIDERAN: %s (%s)\n", ctx, st.Faction, strings.ToLower(st.Standing), standingFeel(st.StandingID))
		if strings.TrimSpace(st.Text) != "" {
			fmt.Fprintf(&b, "DESCRIPCIÓN OFICIAL DE LA FACCIÓN (en el juego): %s\n", strings.TrimSpace(st.Text))
		}
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
	if g.LevelTo > g.LevelFrom && g.Kind != group.Equip && g.Kind != group.Rep {
		parts = append(parts, "Durante este tramo el personaje creció y se hizo más fuerte (no lo cuentes como niveles).")
	}
	if g.Deaths == 1 {
		parts = append(parts, "Durante este tramo cayó derrotado una vez y se levantó.")
	} else if g.Deaths > 1 {
		parts = append(parts, fmt.Sprintf("Durante este tramo cayó derrotado %d veces y siguió adelante.", g.Deaths))
	}
	if len(g.RepGains) > 0 {
		var fs []string
		for f, n := range g.RepGains {
			fs = append(fs, f+" ("+repAmount(n)+")")
		}
		sort.Strings(fs)
		parts = append(parts, "Con lo que hizo en este tramo se ganó el aprecio de: "+strings.Join(fs, ", ")+". Puedes reflejarlo con naturalidad (cómo le miran o le tratan), sin cifras.")
	}
	if g.Kind != group.Equip {
		for _, it := range g.Items {
			parts = append(parts, "Durante este tramo empezó a usar una pieza nueva. "+itemLine(it))
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

// itemLine describe una pieza para el prompt: tipo, calidad y lo que se nota al llevarla.
func itemLine(it model.Event) string {
	var b strings.Builder
	fmt.Fprintf(&b, "PIEZA: %s", it.Item)
	kind := it.ItemSubType
	if kind == "" {
		kind = it.ItemType
	}
	if kind != "" {
		fmt.Fprintf(&b, " (%s)", strings.ToLower(kind))
	}
	fmt.Fprintf(&b, "; una pieza %s; se lleva en: %s. La calidad describe lo valiosa que es, no su color.\n", quality(it.Quality), slotName(it.Slot))
	if feel := itemFeel(it); len(feel) > 0 {
		b.WriteString("Lo que se nota al llevarla (úsalo como sensaciones del personaje, sin cifras ni nombres de atributos):\n- " + strings.Join(feel, "\n- ") + "\n")
	}
	return b.String()
}

var statFeel = []struct{ key, feel string }{
	{"STRENGTH", "fuerza: golpea con más peso, se siente más poderoso"},
	{"STAMINA", "aguante: la nota robusta, se siente más difícil de tumbar"},
	{"AGILITY", "agilidad: se mueve más ligero y rápido"},
	{"INTELLECT", "intelecto: la mente más clara, la Luz o la magia responden mejor"},
	{"SPIRIT", "espíritu: más sereno, se recupera antes del cansancio"},
	{"DEFENSE", "defensa: para y esquiva mejor los golpes"},
	{"ATTACK_POWER", "potencia de ataque: cada golpe pesa más"},
	{"CRIT", "golpes certeros: encuentra los puntos débiles"},
	{"HIT", "precisión: falla menos"},
	{"SPELL_POWER", "poder de los hechizos: la Luz brota con más fuerza"},
	{"BLOCK", "bloqueo: el escudo aguanta mejor"},
	{"DODGE", "esquiva: se aparta a tiempo"},
	{"PARRY", "parada: desvía los golpes con el arma"},
	{"HEALTH_REGEN", "recuperación: las heridas cierran antes"},
	{"MANA_REGEN", "maná: la Luz no se le agota tan pronto"},
}

func itemFeel(it model.Event) []string {
	var out []string
	used := map[string]bool{}
	for _, sf := range statFeel {
		for k, v := range it.Stats {
			if used[k] || v <= 0 || !strings.Contains(strings.ToUpper(k), sf.key) {
				continue
			}
			used[k] = true
			f := sf.feel
			if v >= 6 {
				f += " (de forma muy notable)"
			}
			out = append(out, f)
		}
	}
	if it.Armor > 0 {
		out = append(out, "protección: la armadura detiene mejor los golpes")
	}
	return out
}

// repAmount traduce la reputación ganada a palabras.
func repAmount(n int) string {
	switch {
	case n >= 1000:
		return "mucho"
	case n >= 300:
		return "bastante"
	default:
		return "un poco"
	}
}

// standingFeel explica cada rango sin hablar de mecánicas.
func standingFeel(id int) string {
	switch id {
	case 5:
		return "ya no es un desconocido: le reciben con simpatía"
	case 6:
		return "le respetan y se fían de él"
	case 7:
		return "le veneran: es uno de los suyos, de los que más aprecian"
	case 8:
		return "le ensalzan: su nombre se pronuncia con orgullo entre ellos"
	}
	return "le tienen en mejor estima"
}
