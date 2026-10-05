// Package app une las piezas: lee el addon, agrupa, narra, guarda y publica.
package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Nehonar/cronica-forever/internal/group"
	"github.com/Nehonar/cronica-forever/internal/model"
	"github.com/Nehonar/cronica-forever/internal/narrate"
	"github.com/Nehonar/cronica-forever/internal/store"
)

// Config es el contenido de cronica.json.
type Config struct {
	// Ruta a WTF/Account/<CUENTA>/SavedVariables/Cronica.lua
	SavedVariables string `json:"savedvariables"`
	// Carpeta del repositorio cronica-forever (donde están docs/ y personajes/).
	Repo string `json:"repo"`
	// Ruta a Interface/AddOns/Cronica/CronicaTextos.lua (opcional hasta que exista el addon).
	AddonTexts string `json:"addon_textos"`
	// Comando de Claude Code. Por defecto "claude".
	Claude string `json:"claude"`
	// Modelo opcional (por ejemplo "sonnet"). Vacío = el de tu cuenta por defecto.
	Model string `json:"modelo"`
	// Subir los cambios a GitHub tras cada proceso.
	Publish bool `json:"publicar"`
	// Máximo de relatos nuevos por pasada (para no gastar de golpe).
	MaxPerRun int `json:"max_por_pasada"`
	// Segundos de margen para detectar cadenas.
	ChainWindow int64 `json:"ventana_cadena_seg"`
}

// LoadConfig lee la configuración. Las rutas relativas se resuelven respecto al archivo.
func LoadConfig(path string) (Config, error) {
	var c Config
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return c, fmt.Errorf("%s: %w", path, err)
	}
	base := filepath.Dir(path)
	abs := func(p string) string {
		if p == "" || filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(base, p)
	}
	c.SavedVariables, c.Repo, c.AddonTexts = abs(c.SavedVariables), abs(c.Repo), abs(c.AddonTexts)
	return c, nil
}

// Runner ejecuta el proceso completo.
type Runner struct {
	Cfg      Config
	Narrator narrate.Narrator
	Out      io.Writer
	Now      func() time.Time
	// Notify muestra un aviso en el escritorio (nil = sin avisos).
	Notify func(title, message string)

	status     store.Status
	lastNotice time.Time
}

// warn avisa en el escritorio como mucho una vez cada hora por el mismo motivo.
func (r *Runner) warn(message string) {
	r.status = store.Status{OK: false, Message: message, T: time.Now().Unix()}
	if r.Notify == nil || (time.Since(r.lastNotice) < time.Hour && !r.lastNotice.IsZero()) {
		return
	}
	r.lastNotice = time.Now()
	r.Notify("Crónica", message)
}

// CheckClaude comprueba que Claude Code tiene sesión y, si no, avisa.
func (r *Runner) CheckClaude(ctx context.Context) error {
	ac, ok := r.Narrator.(narrate.AuthChecker)
	if !ok {
		return nil
	}
	if err := ac.CheckAuth(ctx); err != nil {
		msg := "El cronista no puede escribir: " + err.Error() + ". Abre una terminal, ejecuta «claude» e inicia sesión."
		r.logf("%s", msg)
		r.warn(msg)
		return err
	}
	r.status = store.Status{OK: true, T: time.Now().Unix()}
	r.lastNotice = time.Time{}
	r.logf("Claude Code tiene la sesión iniciada.")
	return nil
}

func (r *Runner) logf(format string, a ...any) {
	fmt.Fprintf(r.Out, time.Now().Format("15:04:05")+"  "+format+"\n", a...)
}

// Result resume una pasada.
type Result struct {
	NewStories int
	Failed     int
	Pending    int
}

// Process hace una pasada completa sobre el archivo del addon.
func (r *Runner) Process(ctx context.Context) (Result, error) {
	var res Result
	chars, err := model.LoadSavedVariables(r.Cfg.SavedVariables)
	if err != nil {
		return res, fmt.Errorf("no puedo leer %s: %w", r.Cfg.SavedVariables, err)
	}
	paths := store.Paths{Repo: r.Cfg.Repo}
	opt := group.DefaultOptions()
	if r.Cfg.ChainWindow > 0 {
		opt.ChainWindow = r.Cfg.ChainWindow
	}
	max := r.Cfg.MaxPerRun
	if max <= 0 {
		max = 12
	}
	now := time.Now
	if r.Now != nil {
		now = r.Now
	}

	var docs []*store.Doc
	for _, c := range chars {
		sheet, created, err := paths.LoadSheet(c.Key, c.Name, c.Race, c.Class)
		if err != nil {
			return res, err
		}
		if created {
			r.logf("Personaje nuevo: %s. He creado personajes/%s.json para que escribas su trasfondo.", c.Name, c.Key)
		}
		doc, err := paths.LoadDoc(c.Key)
		if err != nil {
			return res, err
		}
		doc.Character, doc.Level, doc.Realm = sheet, c.Level, c.Realm

		built := group.Build(c.Events, now().Unix(), opt)
		for _, g := range built.Groups {
			if doc.Has(g.ID) {
				continue
			}
			if res.NewStories >= max {
				res.Pending++
				continue
			}
			r.logf("%s · narrando %s (%s)…", c.Name, describe(g), g.Zone)
			st, err := narrate.Narrate(ctx, r.Narrator, sheet, g, doc.Previous(5))
			if err != nil {
				res.Failed++
				r.logf("  ✗ %v", err)
				if strings.Contains(err.Error(), "no encuentro el comando") {
					r.warn("El cronista no puede escribir: " + err.Error())
					return res, err
				}
				// Si falla por la sesión, se avisa; lo no narrado se reintenta en la próxima pasada.
				if r.CheckClaude(ctx) != nil {
					break
				}
				continue
			}
			r.status = store.Status{OK: true, T: time.Now().Unix()}
			doc.Add(g, st)
			res.NewStories++
			r.logf("  ✓ «%s»", st.Title)
			if err := paths.SaveDoc(doc); err != nil { // guardar tras cada relato
				return res, err
			}
		}
		doc.Pending = pending(built)
		doc.Stats = stats(c)
		res.Pending += len(built.PendingLoose) + len(built.OpenChains)
		if err := paths.SaveDoc(doc); err != nil {
			return res, err
		}
		docs = append(docs, doc)
	}
	if err := paths.SaveIndex(); err != nil {
		return res, err
	}
	if r.Cfg.AddonTexts != "" {
		if err := store.WriteAddonTexts(r.Cfg.AddonTexts, docs, 40, r.status); err != nil {
			r.logf("No he podido escribir los textos del addon: %v", err)
		}
	}
	if r.Cfg.Publish && res.NewStories > 0 {
		if err := Publish(r.Cfg.Repo, fmt.Sprintf("Crónica: %d relato(s) nuevo(s)", res.NewStories), r.Out); err != nil {
			r.logf("No he podido publicar: %v", err)
		}
	}
	return res, nil
}

func describe(g group.Group) string {
	switch g.Kind {
	case group.Chain:
		return fmt.Sprintf("cadena de %d misiones", len(g.Quests))
	case group.Loose:
		return fmt.Sprintf("%d misiones sueltas", len(g.Quests))
	case group.Equip:
		return "pieza nueva: " + g.Items[0].Item
	}
	return string(g.Kind)
}

func pending(b group.Result) store.Pending {
	p := store.Pending{Loose: []string{}, OpenChains: [][]string{}}
	for _, q := range b.PendingLoose {
		p.Loose = append(p.Loose, q.Title)
	}
	for _, ch := range b.OpenChains {
		var t []string
		for _, q := range ch {
			t = append(t, q.Title)
		}
		p.OpenChains = append(p.OpenChains, t)
	}
	return p
}

func stats(c model.Character) store.Stats {
	s := store.Stats{Level: c.Level, Played: c.Played, Zones: []string{}, LevelUps: []store.LevelUp{}}
	seen := map[string]bool{}
	for _, e := range c.Events {
		if s.FirstSeen == 0 || e.T < s.FirstSeen {
			s.FirstSeen = e.T
		}
		if e.T > s.LastSeen {
			s.LastSeen = e.T
		}
		switch e.Type {
		case model.EvQuestTurnin:
			s.QuestsDone++
		case model.EvDeath:
			s.Deaths++
		case model.EvLevel:
			s.LevelUps = append(s.LevelUps, store.LevelUp{Level: e.Level, T: e.T})
		}
		if e.Zone != "" && !seen[e.Zone] {
			seen[e.Zone] = true
			s.Zones = append(s.Zones, e.Zone)
		}
	}
	return s
}

// Publish hace commit y push de docs/ y personajes/.
func Publish(repo, message string, out io.Writer) error {
	git := func(args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		b, err := cmd.CombinedOutput()
		return strings.TrimSpace(string(b)), err
	}
	if _, err := git("add", "docs", "personajes"); err != nil {
		return err
	}
	if _, err := git("diff", "--cached", "--quiet"); err == nil {
		return nil // nada que subir
	}
	if o, err := git("commit", "-m", message); err != nil {
		return fmt.Errorf("commit: %s", o)
	}
	if _, err := git("push"); err != nil {
		if o, err := git("pull", "--rebase"); err != nil {
			return fmt.Errorf("pull: %s", o)
		}
		if o, err := git("push"); err != nil {
			return fmt.Errorf("push: %s", o)
		}
	}
	fmt.Fprintf(out, "%s  Publicado en GitHub.\n", time.Now().Format("15:04:05"))
	return nil
}

// Watch vigila el archivo del addon y procesa cada vez que cambia
// (el juego lo escribe al salir o al hacer /reload).
func (r *Runner) Watch(ctx context.Context, every time.Duration) error {
	var last time.Time
	r.logf("Vigilando %s (Ctrl+C para salir)", r.Cfg.SavedVariables)
	r.CheckClaude(ctx)
	for {
		info, err := os.Stat(r.Cfg.SavedVariables)
		switch {
		case errors.Is(err, os.ErrNotExist):
			// aún no existe: el addon no ha guardado nada todavía
		case err != nil:
			r.logf("No puedo leer el archivo: %v", err)
		case info.ModTime().After(last):
			if !last.IsZero() {
				time.Sleep(2 * time.Second) // dejar que el juego termine de escribir
			}
			last = info.ModTime()
			res, err := r.Process(ctx)
			if err != nil {
				r.logf("Error: %v", err)
			} else {
				r.logf("Listo: %d relato(s) nuevo(s), %d pendiente(s).", res.NewStories, res.Pending)
			}
		}
		// Si Claude estaba sin sesión, se vuelve a comprobar cada 10 minutos.
		if !r.status.OK && r.status.T > 0 && time.Since(time.Unix(r.status.T, 0)) > 10*time.Minute {
			r.CheckClaude(ctx)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(every):
		}
	}
}
