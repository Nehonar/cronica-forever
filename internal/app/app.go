// Package app une las piezas: lee el addon, agrupa, narra, guarda y publica.
package app

import (
	"github.com/Nehonar/cronica-forever/internal/system"

	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Nehonar/cronica-forever/internal/group"
	"github.com/Nehonar/cronica-forever/internal/model"
	"github.com/Nehonar/cronica-forever/internal/narrate"
	"github.com/Nehonar/cronica-forever/internal/store"
	"github.com/Nehonar/cronica-forever/internal/wow"
)

// Config es el contenido de cronica.json.
type Config struct {
	// Ruta a WTF/Account/<CUENTA>/SavedVariables/Cronica.lua. Si se deja vacía y
	// está «wow», se usa el Cronica.lua más reciente de cualquier cuenta.
	SavedVariables string `json:"savedvariables"`
	// Carpeta de la versión del juego (p. ej. C:\...\World of Warcraft\_forever_).
	WoW string `json:"wow,omitempty"`
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
	c.SavedVariables, c.Repo, c.AddonTexts, c.WoW = abs(c.SavedVariables), abs(c.Repo), abs(c.AddonTexts), abs(c.WoW)
	if c.Repo == "" {
		c.Repo = base
	}
	return c, nil
}

// SaveConfig guarda la configuración en path.
func SaveConfig(path string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

// DefaultConfig es la configuración de partida, con los datos en home.
func DefaultConfig(home string) Config {
	return Config{Claude: "claude", MaxPerRun: 12, ChainWindow: 90, Repo: home}
}

// DefaultHome es la carpeta de datos de Crónica: %APPDATA%\Cronica en Windows,
// ~/.config/Cronica en Linux.
func DefaultHome() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "Cronica"
	}
	return filepath.Join(dir, "Cronica")
}

// ResolveConfig decide qué archivo de configuración usar: el indicado, el
// cronica.json de la carpeta actual si existe (modo desarrollo) o el de la
// carpeta de datos.
func ResolveConfig(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if _, err := os.Stat("cronica.json"); err == nil {
		return "cronica.json"
	}
	return filepath.Join(DefaultHome(), "cronica.json")
}

// Runner ejecuta el proceso completo.
type Runner struct {
	Cfg      Config
	Narrator narrate.Narrator
	Out      io.Writer
	Now      func() time.Time
	// Notify muestra un aviso en el escritorio (nil = sin avisos).
	Notify func(title, message string)
	// OnResult se llama tras cada pasada (la bandeja lo usa para refrescar su menú).
	OnResult func(Result, error)
	// OnPublish se llama tras cada intento de publicar en GitHub.
	OnPublish func()
	// Inform muestra un aviso discreto (misiones nuevas). nil = sin avisos.
	Inform func(title, message string)

	mu         sync.Mutex
	cfgMu      sync.RWMutex
	pubMu      sync.Mutex
	pub        PublishState
	pubNotice  time.Time
	status     store.Status
	lastNotice time.Time
	announced  map[string]bool
}

// Config devuelve una copia de la configuración en uso.
func (r *Runner) Config() Config {
	r.cfgMu.RLock()
	defer r.cfgMu.RUnlock()
	return r.Cfg
}

// UpdateConfig cambia la configuración en uso y devuelve la nueva.
func (r *Runner) UpdateConfig(f func(*Config)) Config {
	r.cfgMu.Lock()
	defer r.cfgMu.Unlock()
	f(&r.Cfg)
	return r.Cfg
}

// Status devuelve el último estado conocido del cronista.
func (r *Runner) Status() store.Status { return r.status }

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
		msg := "El cronista no puede escribir: " + err.Error() + ". Haz clic en el icono de Crónica → «Configuración…» para arreglarlo."
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
	// Personajes sin trasfondo: su progreso se registra pero no se narra hasta que lo tengan.
	NeedsBackstory []CharacterInfo
	LastTitle      string
	NewEncargos    int
}

// CharacterInfo describe un personaje para la bandeja y la entrevista.
type CharacterInfo struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	Race    string `json:"race"`
	Class   string `json:"class"`
	Level   int    `json:"level"`
	Waiting int    `json:"waiting"` // relatos que esperan a tener trasfondo
}

// Process hace una pasada completa sobre el archivo del addon.
func (r *Runner) Process(ctx context.Context) (res Result, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer func() {
		if r.OnResult != nil {
			r.OnResult(res, err)
		}
	}()
	return r.process(ctx)
}

// SVPath es el archivo del addon que se lee: el configurado o, si no hay, el
// más reciente de cualquier cuenta de la carpeta del juego.
func (r *Runner) SVPath() string {
	c := r.Config()
	if c.SavedVariables != "" || c.WoW == "" {
		return c.SavedVariables
	}
	return wow.LatestSavedVariables(c.WoW)
}

func (r *Runner) process(ctx context.Context) (Result, error) {
	var res Result
	sv := r.SVPath()
	if sv == "" {
		return res, fmt.Errorf("aún no hay datos del addon: entra al juego con el addon activado y haz /reload o sal")
	}
	chars, err := model.LoadSavedVariables(sv)
	if err != nil {
		return res, fmt.Errorf("no puedo leer %s: %w", sv, err)
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
			r.logf("Personaje nuevo: %s (personajes/%s.json).", c.Name, c.Key)
		}
		doc, err := paths.LoadDoc(c.Key)
		if err != nil {
			return res, err
		}
		doc.Character, doc.Level, doc.Realm = sheet, c.Level, c.Realm

		// Misiones aceptadas: se narran al momento para leerlas en la app.
		todo := doc.SyncEncargos(questMaps(c.Events))
		if sheet.Backstory != "" {
			var titles []string
			for _, i := range todo {
				if res.NewEncargos >= 10 {
					break
				}
				e := &doc.Encargos[i]
				r.logf("%s · nueva misión: «%s»…", c.Name, e.Title)
				h, txt, err := narrate.Encargo(ctx, r.Narrator, sheet, narrate.QuestInfo{
					Title: e.Title, Giver: e.Giver, Zone: e.Zone, Subzone: e.Subzone, Text: e.Original, Objectives: e.Objectives,
				})
				if err != nil {
					r.logf("  ✗ %v", err)
					if r.CheckClaude(ctx) != nil {
						break
					}
					continue
				}
				e.Heading, e.Text, e.Narrated = h, txt, time.Now().UTC().Format(time.RFC3339)
				res.NewEncargos++
				titles = append(titles, e.Title)
				r.logf("  ✓ «%s»", h)
				if err := paths.SaveDoc(doc); err != nil {
					return res, err
				}
			}
			if len(titles) > 0 && r.Inform != nil {
				msg := "«" + titles[0] + "»"
				if len(titles) > 1 {
					msg = fmt.Sprintf("%s y %d más", msg, len(titles)-1)
				}
				r.Inform("Crónica: nueva misión", msg+". Léela en Misiones.")
			}
			if _, ok := paths.LoadPhrases(c.Key); !ok {
				r.logf("%s · escribiendo sus frases…", c.Name)
				if ph, err := narrate.Phrases(ctx, r.Narrator, sheet); err != nil {
					r.logf("  ✗ frases: %v", err)
				} else if err := paths.SavePhrases(c.Key, ph); err != nil {
					r.logf("  ✗ frases: %v", err)
				} else {
					r.logf("  ✓ frases listas")
				}
			}
		}

		copt := opt
		copt.Narrated = map[int64]bool{}
		for _, st := range doc.Stories {
			for _, q := range st.Quests {
				copt.Narrated[q.ID] = true
			}
		}
		built := group.Build(c.Events, now().Unix(), copt)
		if sheet.Backstory == "" {
			// Sin trasfondo no se narra: se guarda el progreso y se espera a que lo crees.
			waiting := 0
			for _, g := range built.Groups {
				if !doc.Has(g.ID) {
					waiting++
				}
			}
			res.NeedsBackstory = append(res.NeedsBackstory, CharacterInfo{Key: c.Key, Name: c.Name, Race: c.Race, Class: c.Class, Level: c.Level, Waiting: waiting})
			if r.announced == nil {
				r.announced = map[string]bool{}
			}
			if !r.announced[c.Key] {
				r.announced[c.Key] = true
				r.logf("%s aún no tiene trasfondo: guardo su progreso (%d relato(s) en espera) hasta que crees su historia.", c.Name, waiting)
				if r.Notify != nil {
					r.Notify("Crónica: personaje nuevo", c.Name+" ("+strings.ToLower(c.Race)+" "+strings.ToLower(c.Class)+") aún no tiene historia. Ábrela desde el icono de Crónica para crearla.")
				}
			}
			doc.Pending = pending(built)
			doc.Pending.AwaitingBackstory = waiting
			doc.Stats = stats(c)
			if err := paths.SaveDoc(doc); err != nil {
				return res, err
			}
			docs = append(docs, doc)
			continue
		}
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
			res.LastTitle = st.Title
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
		phrases := map[string]store.Phrases{}
		for _, d := range docs {
			if ph, ok := paths.LoadPhrases(d.Key); ok {
				phrases[d.Key] = ph
			}
		}
		if err := store.WriteAddonTexts(r.Cfg.AddonTexts, docs, 40, r.status, phrases); err != nil {
			r.logf("No he podido escribir los textos del addon: %v", err)
		}
	}
	msg := "Crónica: misiones y personajes al día"
	if res.NewStories > 0 {
		msg = fmt.Sprintf("Crónica: %d relato(s) nuevo(s)", res.NewStories)
	}
	r.PublishNow(msg, nil)
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

// Publish hace commit y push de docs/ y personajes/. También sube los
// commits que se quedaron sin subir en un intento anterior.
func Publish(repo, message string, out io.Writer) error {
	git := func(args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		system.Hide(cmd)
		b, err := cmd.CombinedOutput()
		return strings.TrimSpace(string(b)), err
	}
	add := []string{"add"}
	for _, d := range []string{"docs", "personajes"} {
		if _, err := os.Stat(filepath.Join(repo, d)); err == nil {
			add = append(add, d)
		}
	}
	if len(add) > 1 {
		if o, err := git(add...); err != nil {
			return fmt.Errorf("git add: %s", o)
		}
	}
	if _, err := git("diff", "--cached", "--quiet"); err != nil {
		args := []string{"commit", "-m", message}
		// Git recién instalado no sabe quién eres: firmar como el dueño del repositorio.
		if email, _ := git("config", "user.email"); email == "" {
			owner := "cronica"
			if u, err := git("config", "--get", "remote.origin.url"); err == nil {
				if m := reOwner.FindStringSubmatch(u); m != nil {
					owner = m[1]
				}
			}
			args = append([]string{"-c", "user.name=" + owner, "-c", "user.email=" + owner + "@users.noreply.github.com"}, args...)
		}
		if o, err := git(args...); err != nil {
			return fmt.Errorf("no he podido guardar el cambio (commit): %s", o)
		}
	}
	if ahead, err := git("rev-list", "--count", "@{u}..HEAD"); err == nil && ahead == "0" {
		return nil // nada que subir
	}
	if _, err := git("push"); err != nil {
		if o, err := git("pull", "--rebase"); err != nil {
			git("rebase", "--abort")
			return fmt.Errorf("no he podido traer los cambios de GitHub: %s", o)
		}
		if o, err := git("push"); err != nil {
			return fmt.Errorf("GitHub no acepta la subida: %s", o)
		}
	}
	fmt.Fprintf(out, "%s  Publicado en GitHub.\n", time.Now().Format("15:04:05"))
	return nil
}

var reOwner = regexp.MustCompile(`github\.com[/:]([\w.-]+)/`)

// PublishState es el resultado de la última publicación en GitHub.
type PublishState struct {
	At  time.Time // última vez que se publicó bien (o que no había nada que subir)
	Err string    // último error, si lo hubo
	T   time.Time // última vez que se intentó
}

// PublishStatus devuelve cómo fue la última publicación.
func (r *Runner) PublishStatus() PublishState {
	r.pubMu.Lock()
	defer r.pubMu.Unlock()
	return r.pub
}

// PublishNow publica ya en GitHub (si está activado) y apunta el resultado.
func (r *Runner) PublishNow(message string, out io.Writer) error {
	cfg := r.Config()
	if !cfg.Publish {
		return nil
	}
	if out == nil {
		out = r.Out
	}
	err := Publish(cfg.Repo, message, out)
	r.pubMu.Lock()
	r.pub.T = time.Now()
	if err != nil {
		r.pub.Err = err.Error()
	} else {
		r.pub.Err, r.pub.At = "", time.Now()
	}
	notice := err != nil && time.Since(r.pubNotice) > time.Hour
	if notice {
		r.pubNotice = time.Now()
	}
	r.pubMu.Unlock()
	if err != nil {
		r.logf("No he podido publicar en GitHub: %v", err)
		if notice && r.Notify != nil {
			r.Notify("Crónica", "No he podido publicar en GitHub. Mira el detalle en el icono de Crónica → Configuración…")
		}
	}
	if r.OnPublish != nil {
		r.OnPublish()
	}
	return err
}

// Watch vigila el archivo del addon y procesa cada vez que cambia
// (el juego lo escribe al salir o al hacer /reload).
func (r *Runner) Watch(ctx context.Context, every time.Duration) error {
	var last time.Time
	var lastPath string
	r.logf("Vigilando los datos del addon (Ctrl+C para salir)")
	r.CheckClaude(ctx)
	for {
		path := r.SVPath()
		if path != lastPath && path != "" {
			r.logf("Archivo del addon: %s", path)
			lastPath, last = path, time.Time{}
		}
		info, err := os.Stat(path)
		switch {
		case path == "" || errors.Is(err, os.ErrNotExist):
			// aún no existe: el addon no ha guardado nada todavía
		case err != nil:
			r.logf("No puedo leer el archivo: %v", err)
		case info.ModTime().After(last):
			if !last.IsZero() {
				time.Sleep(time.Second) // dejar que el juego termine de escribir
			}
			last = info.ModTime()
			res, err := r.Process(ctx)
			if err != nil {
				r.logf("Error: %v", err)
			} else {
				r.logf("Listo: %d relato(s) y %d misión(es) nuevas, %d pendiente(s).", res.NewStories, res.NewEncargos, res.Pending)
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

// questMaps extrae de los eventos las misiones aceptadas, entregadas y abandonadas.
func questMaps(events []model.Event) (accepts, turnins, abandons map[int64]store.Encargo) {
	accepts, turnins, abandons = map[int64]store.Encargo{}, map[int64]store.Encargo{}, map[int64]store.Encargo{}
	lastAccept := map[int64]int64{}
	for _, e := range events {
		switch e.Type {
		case model.EvQuestAccept:
			lastAccept[e.ID] = e.T
			accepts[e.ID] = store.Encargo{ID: e.ID, Title: e.Title, Giver: e.NPC, Zone: e.Zone, Subzone: e.Subzone,
				Objectives: e.Objectives, Original: e.Text, AcceptT: e.T}
		case model.EvQuestTurnin:
			turnins[e.ID] = store.Encargo{ID: e.ID, AcceptT: lastAccept[e.ID]}
		case model.EvQuestAbandon:
			abandons[e.ID] = store.Encargo{ID: e.ID, AcceptT: lastAccept[e.ID]}
		}
	}
	return
}
