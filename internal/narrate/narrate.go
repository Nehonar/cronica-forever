// Package narrate convierte grupos de eventos en texto narrativo usando Claude.
package narrate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/Nehonar/cronica-forever/internal/group"
)

// Sheet es la ficha de trasfondo de un personaje (personajes/<clave>.json).
type Sheet struct {
	Key       string   `json:"key"`
	Name      string   `json:"name"`
	Epithet   string   `json:"epithet"`
	Race      string   `json:"race"`
	Class     string   `json:"class"`
	Voice     string   `json:"voice"`
	Backstory string   `json:"backstory"`
	Threads   []string `json:"threads"`
	Motto     string   `json:"motto"`
}

// Previous es un relato anterior, resumido, para dar continuidad.
type Previous struct {
	Title   string
	Summary string
}

// Story es el resultado de narrar un grupo.
type Story struct {
	Title   string `json:"title"`
	Text    string `json:"text"`
	Summary string `json:"summary"`
}

// Narrator genera texto a partir de un prompt.
type Narrator interface {
	Generate(ctx context.Context, system, prompt string) (string, error)
}

// ClaudeCLI llama a Claude Code en modo no interactivo (`claude -p`),
// usando la sesión iniciada en este equipo.
type ClaudeCLI struct {
	Command string        // ruta o nombre del ejecutable; por defecto "claude"
	Model   string        // opcional: modelo concreto
	Timeout time.Duration // por defecto 3 minutos
}

// Generate envía el prompt por la entrada estándar y devuelve la respuesta.
func (c ClaudeCLI) Generate(ctx context.Context, system, prompt string) (string, error) {
	cmdName, _ := FindClaude(c.Command)
	timeout := c.Timeout
	if timeout == 0 {
		timeout = 3 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	// El prompt de sistema va en un archivo temporal: en Windows, «claude» suele ser
	// un .cmd y los argumentos con saltos de línea o comillas se estropean.
	sp, err := os.CreateTemp("", "cronica-sistema-*.txt")
	if err != nil {
		return "", err
	}
	defer os.Remove(sp.Name())
	if _, err := sp.WriteString(system); err != nil {
		sp.Close()
		return "", err
	}
	sp.Close()
	// Sin herramientas ni servidores MCP: solo texto.
	args := []string{"-p", "--output-format", "text", "--tools=", "--strict-mcp-config", "--system-prompt-file", sp.Name()}
	if c.Model != "" {
		args = append(args, "--model", c.Model)
	}
	cmd := exec.CommandContext(ctx, cmdName, args...)
	cmd.Dir = os.TempDir() // sin el contexto de ningún proyecto
	cmd.Stdin = strings.NewReader(prompt)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("Claude no ha respondido en %s", timeout)
		}
		var nf *exec.Error
		if errors.As(err, &nf) {
			return "", fmt.Errorf("no encuentro el comando %q: ¿está instalado Claude Code y has iniciado sesión?", cmdName)
		}
		return "", fmt.Errorf("claude: %v: %s", err, strings.TrimSpace(errb.String()))
	}
	return strings.TrimSpace(out.String()), nil
}

// Fake devuelve un texto fijo; sirve para probar sin gastar uso.
type Fake struct{}

// Generate implementa Narrator.
func (Fake) Generate(_ context.Context, _ string, prompt string) (string, error) {
	first := strings.SplitN(prompt, "\n", 2)[0]
	return "TÍTULO: Relato de prueba\n\nTexto de prueba generado sin IA.\n\nSegundo párrafo de prueba.\n\nRESUMEN: Prueba. " + first, nil
}

var (
	reTitle   = regexp.MustCompile(`(?mi)^\s*\**T[ÍI]TULO:?\**:?\s*(.+)$`)
	reSummary = regexp.MustCompile(`(?mi)^\s*\**RESUMEN:?\**:?\s*`)
)

// Parse separa título, texto y resumen de la respuesta del modelo.
func Parse(raw string) (Story, error) {
	s := strings.ReplaceAll(raw, "\r", "")
	var st Story
	if m := reTitle.FindStringSubmatchIndex(s); m != nil {
		st.Title = strings.Trim(s[m[2]:m[3]], " \t\"«»*")
		s = s[:m[0]] + s[m[1]:]
	}
	if loc := reSummary.FindStringIndex(s); loc != nil {
		st.Summary = strings.TrimSpace(s[loc[1]:])
		s = s[:loc[0]]
	}
	st.Text = strings.TrimSpace(s)
	if st.Text == "" {
		return st, fmt.Errorf("la respuesta no contiene texto")
	}
	return st, nil
}

// Narrate construye el prompt de un grupo, llama al narrador y devuelve el relato.
func Narrate(ctx context.Context, n Narrator, sheet Sheet, g group.Group, prev []Previous) (Story, error) {
	system, prompt := BuildPrompt(sheet, g, prev)
	raw, err := n.Generate(ctx, system, prompt)
	if err != nil {
		return Story{}, err
	}
	st, err := Parse(raw)
	if err != nil {
		return Story{}, err
	}
	if st.Title == "" {
		st.Title = defaultTitle(g)
	}
	return st, nil
}

func defaultTitle(g group.Group) string {
	switch g.Kind {
	case group.Equip:
		return g.Items[0].Item
	case group.Chain:
		return g.Quests[len(g.Quests)-1].Title
	}
	return "Encargos en " + g.Zone
}
