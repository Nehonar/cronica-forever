package ui

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Nehonar/cronica-forever/internal/app"
	"github.com/Nehonar/cronica-forever/internal/voz"
)

// Paso «Voz del narrador» de la página Preparar: conectar con Azure (clave y
// región de un recurso de Voz gratuito), elegir voz y probarla. La clave nunca
// vuelve al navegador: la página solo sabe si hay una y sus últimas letras.

// VoiceState es lo que ve la página de la voz.
type VoiceState struct {
	Active  bool        `json:"activa"`
	Region  string      `json:"region"`
	HasKey  bool        `json:"tiene_clave"`
	KeyTail string      `json:"clave_fin,omitempty"`
	Voice   string      `json:"voz"`
	Rate    int         `json:"velocidad"`
	Pitch   int         `json:"tono"`
	Voices  []voz.Voice `json:"voces,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type voiceCache struct {
	mu     sync.Mutex
	sig    string // región + clave con la que se pidió la lista
	voices []voz.Voice
	err    string
}

var voices voiceCache

func keyTail(k string) string {
	k = strings.TrimSpace(k)
	if len(k) <= 4 {
		return ""
	}
	return k[len(k)-4:]
}

// voiceState arma el estado; la lista de voces se pide una vez por clave.
func (s *Server) voiceState(ctx context.Context) VoiceState {
	c := s.Runner.Config().Voz
	d := c.WithDefaults()
	st := VoiceState{Active: c.Active, Region: c.Region, HasKey: c.Key != "", KeyTail: keyTail(c.Key), Voice: d.Voice, Rate: d.Rate, Pitch: d.Pitch}
	if !st.HasKey {
		return st
	}
	sig := c.Region + "\x00" + c.Key
	voices.mu.Lock()
	defer voices.mu.Unlock()
	if voices.sig != sig {
		ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
		defer cancel()
		list, err := s.Runner.Voice.Voices(ctx, c)
		voices.sig, voices.voices, voices.err = sig, list, ""
		if err != nil {
			voices.err = err.Error()
			if !errors.Is(err, voz.ErrKey) {
				voices.sig = "" // error de red: se reintenta la próxima vez
			}
		}
	}
	st.Voices, st.Error = voices.voices, voices.err
	return st
}

func (s *Server) apiSetupVoice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "usa POST")
		return
	}
	var req struct {
		Action string `json:"accion"` // conectar | ajustar | quitar | doblar
		Region string `json:"region"`
		Key    string `json:"clave"`
		Voice  string `json:"voz"`
		Rate   *int   `json:"velocidad"`
		Pitch  *int   `json:"tono"`
		Active *bool  `json:"activa"`
	}
	if err := readBody(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "petición no válida")
		return
	}
	switch req.Action {
	case "conectar":
		c := s.Runner.Config().Voz
		c.Region = strings.ToLower(strings.TrimSpace(req.Region))
		if k := strings.TrimSpace(req.Key); k != "" {
			c.Key = k
		}
		if c.Region == "" || c.Key == "" {
			writeErr(w, http.StatusBadRequest, "Pega la clave y escribe la región de tu recurso de Voz.")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		list, err := s.Runner.Voice.Voices(ctx, c)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		if len(list) == 0 {
			writeErr(w, http.StatusBadRequest, "La clave funciona, pero esa región no tiene voces de España.")
			return
		}
		c.Active = true
		c = c.WithDefaults()
		voices.mu.Lock()
		voices.sig, voices.voices, voices.err = c.Region+"\x00"+c.Key, list, ""
		voices.mu.Unlock()
		if err := s.saveConfig(func(cfg *app.Config) { cfg.Voz = c }); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		// Doblar ya las historias que estén sin voz.
		go s.Runner.Process(context.Background())
	case "ajustar":
		err := s.saveConfig(func(cfg *app.Config) {
			if req.Voice != "" {
				cfg.Voz.Voice = req.Voice
			}
			if req.Rate != nil {
				cfg.Voz.Rate = clamp(*req.Rate, -30, 30)
			}
			if req.Pitch != nil {
				cfg.Voz.Pitch = clamp(*req.Pitch, -20, 20)
			}
			if req.Active != nil {
				cfg.Voz.Active = *req.Active
			}
		})
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
	case "quitar":
		if err := s.saveConfig(func(cfg *app.Config) { cfg.Voz = voz.Config{} }); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		voices.mu.Lock()
		voices.sig, voices.voices, voices.err = "", nil, ""
		voices.mu.Unlock()
	case "doblar":
		go s.Runner.Process(context.Background())
	default:
		writeErr(w, http.StatusBadRequest, "acción desconocida")
		return
	}
	writeJSON(w, s.setupState(r.Context()))
}

// apiVoiceSample dobla una frase de prueba con la voz elegida y devuelve el mp3
// (no guarda nada).
func (s *Server) apiVoiceSample(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "usa POST")
		return
	}
	var req struct {
		Voice string `json:"voz"`
		Rate  int    `json:"velocidad"`
		Pitch int    `json:"tono"`
		Text  string `json:"texto"`
	}
	if err := readBody(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "petición no válida")
		return
	}
	c := s.Runner.Config().Voz
	if c.Key == "" || c.Region == "" {
		writeErr(w, http.StatusBadRequest, "Primero conecta con Azure.")
		return
	}
	if req.Voice != "" {
		c.Voice, c.Rate, c.Pitch = req.Voice, clamp(req.Rate, -30, 30), clamp(req.Pitch, -20, 20)
	}
	text := strings.TrimSpace(req.Text)
	if text == "" || len([]rune(text)) > 600 {
		text = voz.Sample
	}
	ctx, cancel := context.WithTimeout(r.Context(), 40*time.Second)
	defer cancel()
	mp3, err := s.Runner.Voice.Speak(ctx, c, "", text)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(mp3)
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
