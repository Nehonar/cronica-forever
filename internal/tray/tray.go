// Package tray muestra Crónica en la bandeja del sistema.
package tray

import (
	_ "embed"
	"fmt"
	"runtime"
	"sync"

	"fyne.io/systray"
)

//go:embed icono.png
var iconPNG []byte

//go:embed icono.ico
var iconICO []byte

//go:embed icono-aviso.png
var alertPNG []byte

//go:embed icono-aviso.ico
var alertICO []byte

func icon(alert bool) []byte {
	if runtime.GOOS == "windows" {
		if alert {
			return alertICO
		}
		return iconICO
	}
	if alert {
		return alertPNG
	}
	return iconPNG
}

// Character es un personaje que aparece en el menú.
type Character struct {
	Key   string
	Label string
}

// State es lo que muestra el menú.
type State struct {
	Status      string      // línea de estado («Vigilando», «Narrando…»)
	ClaudeOK    bool        // Claude tiene sesión
	LastStory   string      // último relato escrito
	NewChars    []Character // personajes sin trasfondo
	PendingText string      // «3 relatos pendientes», etc.
	GitHub      string      // estado de la publicación («» = no se publica)
	GitHubError bool
}

// Actions son las respuestas a los clics del menú.
type Actions struct {
	OpenChronicle func()
	OpenQuests    func()
	OpenCharacter func(key string) // key vacío = página de personajes
	NarrateNow    func()
	FixClaude     func()
	Settings      func()
	Quit          func()
}

// App es el icono de la bandeja.
type App struct {
	act Actions

	mu        sync.Mutex
	ready     bool
	pending   *State
	status    *systray.MenuItem
	claude    *systray.MenuItem
	github    *systray.MenuItem
	last      *systray.MenuItem
	pendingMI *systray.MenuItem
	newHeader *systray.MenuItem
	charItems []*systray.MenuItem
	charKeys  []string
}

const maxChars = 8

// Run muestra el icono y bloquea hasta que se elige «Salir». onReady se llama
// cuando el icono ya está listo.
func Run(act Actions, onReady func(*App)) {
	a := &App{act: act}
	systray.Run(func() {
		a.build()
		if onReady != nil {
			go onReady(a)
		}
	}, func() {})
}

func (a *App) build() {
	systray.SetIcon(icon(false))
	systray.SetTitle("")
	systray.SetTooltip("Crónica de Forever")

	a.status = systray.AddMenuItem("Crónica: arrancando…", "")
	a.status.Disable()
	a.last = systray.AddMenuItem("", "")
	a.last.Disable()
	a.last.Hide()
	a.pendingMI = systray.AddMenuItem("", "")
	a.pendingMI.Disable()
	a.pendingMI.Hide()
	systray.AddSeparator()

	a.newHeader = systray.AddMenuItem("Personajes nuevos sin historia:", "")
	a.newHeader.Disable()
	a.newHeader.Hide()
	for i := 0; i < maxChars; i++ {
		mi := systray.AddMenuItem("", "Crear su historia con el cronista")
		mi.Hide()
		a.charItems = append(a.charItems, mi)
		a.charKeys = append(a.charKeys, "")
		go func(i int, mi *systray.MenuItem) {
			for range mi.ClickedCh {
				a.mu.Lock()
				key := a.charKeys[i]
				a.mu.Unlock()
				if a.act.OpenCharacter != nil {
					a.act.OpenCharacter(key)
				}
			}
		}(i, mi)
	}

	quests := systray.AddMenuItem("Misiones en curso", "Tus misiones contadas por el cronista, al momento")
	chron := systray.AddMenuItem("Abrir mi crónica", "La web con tus relatos")
	chars := systray.AddMenuItem("Personajes e historias", "Crear o reescribir la historia de un personaje")
	now := systray.AddMenuItem("Narrar ahora", "Procesar ya lo que haya guardado el juego")
	a.claude = systray.AddMenuItem("Claude: comprobando…", "")
	a.github = systray.AddMenuItem("", "Ver el estado de la publicación")
	a.github.Hide()
	settings := systray.AddMenuItem("Configuración…", "Carpeta del juego, Claude, GitHub y arranque")
	systray.AddSeparator()
	quit := systray.AddMenuItem("Salir", "Cerrar Crónica (dejará de narrar hasta que vuelvas a abrirla)")

	click := func(mi *systray.MenuItem, f func()) {
		go func() {
			for range mi.ClickedCh {
				if f != nil {
					f()
				}
			}
		}()
	}
	click(quests, a.act.OpenQuests)
	click(chron, a.act.OpenChronicle)
	click(chars, func() {
		if a.act.OpenCharacter != nil {
			a.act.OpenCharacter("")
		}
	})
	click(now, a.act.NarrateNow)
	click(a.claude, a.act.FixClaude)
	click(settings, a.act.Settings)
	click(a.github, a.act.Settings)
	click(quit, func() {
		if a.act.Quit != nil {
			a.act.Quit()
		}
		systray.Quit()
	})

	a.mu.Lock()
	a.ready = true
	p := a.pending
	a.mu.Unlock()
	if p != nil {
		a.Update(*p)
	}
}

// Update refresca el menú y el icono.
func (a *App) Update(s State) {
	a.mu.Lock()
	if !a.ready {
		a.pending = &s
		a.mu.Unlock()
		return
	}
	a.mu.Unlock()

	a.status.SetTitle("Crónica: " + s.Status)
	if s.LastStory != "" {
		a.last.SetTitle("Último relato: «" + s.LastStory + "»")
		a.last.Show()
	}
	if s.PendingText != "" {
		a.pendingMI.SetTitle(s.PendingText)
		a.pendingMI.Show()
	} else {
		a.pendingMI.Hide()
	}
	if s.GitHub != "" {
		prefix := "GitHub: "
		if s.GitHubError {
			prefix = "⚠ GitHub: "
		}
		a.github.SetTitle(prefix + s.GitHub)
		a.github.Show()
	} else {
		a.github.Hide()
	}
	if s.ClaudeOK {
		a.claude.SetTitle("Claude: sesión iniciada ✓")
		a.claude.SetTooltip("Todo en orden")
	} else {
		a.claude.SetTitle("⚠ Claude: iniciar sesión…")
		a.claude.SetTooltip("Abrir la configuración para instalar Claude Code o iniciar sesión")
	}

	a.mu.Lock()
	for i, mi := range a.charItems {
		if i < len(s.NewChars) {
			a.charKeys[i] = s.NewChars[i].Key
			mi.SetTitle("✦ " + s.NewChars[i].Label + " — crear su historia")
			mi.Show()
		} else {
			a.charKeys[i] = ""
			mi.Hide()
		}
	}
	a.mu.Unlock()
	if len(s.NewChars) > 0 {
		a.newHeader.Show()
	} else {
		a.newHeader.Hide()
	}

	alert := !s.ClaudeOK || len(s.NewChars) > 0
	systray.SetIcon(icon(alert))
	tip := "Crónica de Forever"
	if len(s.NewChars) > 0 {
		tip += fmt.Sprintf(" — %d personaje(s) esperan su historia", len(s.NewChars))
	} else if !s.ClaudeOK {
		tip += " — Claude necesita que inicies sesión"
	}
	systray.SetTooltip(tip)
}

// Quit cierra el icono de la bandeja.
func (a *App) Quit() { systray.Quit() }
