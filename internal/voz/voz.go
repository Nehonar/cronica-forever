// Package voz dobla las historias con las voces neuronales de Azure (servicio
// «Voz» de Microsoft). El plan gratuito (F0) da 500.000 caracteres al mes y,
// al llegar al límite, deja de responder hasta el mes siguiente: nunca cobra.
//
// Solo hace falta la clave y la región del recurso; se habla con la API REST,
// sin SDK. La clave se guarda en la configuración de Crónica, en tu PC, y no
// se publica nunca.
package voz

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Config es la parte de cronica.json que toca a la voz.
type Config struct {
	Active bool   `json:"activa"`
	Region string `json:"region"` // p. ej. «westeurope»
	Key    string `json:"clave"`
	Voice  string `json:"voz"`       // p. ej. «es-ES-AlvaroNeural»
	Rate   int    `json:"velocidad"` // en %, negativo = más lento
	Pitch  int    `json:"tono"`      // en %, negativo = más grave
}

// Defaults: un narrador grave y pausado.
const (
	DefaultVoice = "es-ES-AlvaroNeural"
	DefaultRate  = -8
	DefaultPitch = -4
)

// Ready indica si está todo para doblar.
func (c Config) Ready() bool { return c.Active && c.Key != "" && c.Region != "" }

// WithDefaults rellena voz, velocidad y tono si están vacíos.
func (c Config) WithDefaults() Config {
	if c.Voice == "" {
		c.Voice, c.Rate, c.Pitch = DefaultVoice, DefaultRate, DefaultPitch
	}
	return c
}

// Voice es una voz disponible.
type Voice struct {
	ID     string `json:"id"`     // ShortName: es-ES-AlvaroNeural
	Name   string `json:"nombre"` // Álvaro
	Gender string `json:"genero"` // Hombre / Mujer
}

// Client habla con Azure.
type Client struct {
	HTTP *http.Client
	// Base sustituye a https://<región>.tts.speech.microsoft.com (para pruebas).
	Base string
}

var reRegion = regexp.MustCompile(`^[a-z0-9]+$`)

func (c Client) base(region string) (string, error) {
	if c.Base != "" {
		return c.Base, nil
	}
	region = strings.ToLower(strings.TrimSpace(region))
	if !reRegion.MatchString(region) {
		return "", fmt.Errorf("la región «%s» no es válida (por ejemplo: westeurope)", region)
	}
	return "https://" + region + ".tts.speech.microsoft.com", nil
}

func (c Client) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 90 * time.Second}
}

// ErrKey: la clave o la región no valen.
var ErrKey = errors.New("Azure no acepta la clave: revisa que la clave y la región son las de tu recurso de Voz")

// ErrQuota: se ha gastado lo gratuito del mes (o demasiadas peticiones seguidas).
var ErrQuota = errors.New("Azure no dobla más por ahora: se ha llegado al límite gratuito o a demasiadas peticiones seguidas. Se reintentará más tarde")

func statusErr(resp *http.Response) error {
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrKey
	case http.StatusTooManyRequests:
		return ErrQuota
	}
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 400))
	msg := strings.TrimSpace(string(b))
	if msg == "" {
		msg = resp.Status
	}
	return fmt.Errorf("Azure ha respondido con un error: %s", msg)
}

// Voices comprueba la clave y devuelve las voces de España.
func (c Client) Voices(ctx context.Context, cfg Config) ([]Voice, error) {
	base, err := c.base(cfg.Region)
	if err != nil {
		return nil, err
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, base+"/cognitiveservices/voices/list", nil)
	req.Header.Set("Ocp-Apim-Subscription-Key", strings.TrimSpace(cfg.Key))
	resp, err := c.http().Do(req)
	if err != nil {
		return nil, fmt.Errorf("no llego a Azure (¿hay internet? ¿la región es correcta?): %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, statusErr(resp)
	}
	var list []struct {
		ShortName   string `json:"ShortName"`
		DisplayName string `json:"DisplayName"`
		LocalName   string `json:"LocalName"`
		Gender      string `json:"Gender"`
		Locale      string `json:"Locale"`
		VoiceType   string `json:"VoiceType"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("respuesta de Azure no válida: %w", err)
	}
	var out []Voice
	for _, v := range list {
		if v.Locale != "es-ES" || (v.VoiceType != "" && v.VoiceType != "Neural") {
			continue
		}
		g := "Mujer"
		if v.Gender == "Male" {
			g = "Hombre"
		}
		name := v.LocalName
		if name == "" {
			name = v.DisplayName
		}
		out = append(out, Voice{ID: v.ShortName, Name: name, Gender: g})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Gender != out[j].Gender {
			return out[i].Gender < out[j].Gender
		}
		return plain(out[i].Name) < plain(out[j].Name)
	})
	return out, nil
}

var accents = strings.NewReplacer("Á", "A", "É", "E", "Í", "I", "Ó", "O", "Ú", "U", "á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u")

// plain quita las tildes para ordenar (Álvaro antes que Darío).
func plain(s string) string { return accents.Replace(s) }

// SSML construye el texto a leer: párrafos con una pausa entre ellos, a la
// velocidad y el tono elegidos.
func SSML(cfg Config, title, text string) string {
	cfg = cfg.WithDefaults()
	var b strings.Builder
	b.WriteString(`<speak version="1.0" xml:lang="es-ES" xmlns="http://www.w3.org/2001/10/synthesis">`)
	fmt.Fprintf(&b, `<voice name="%s">`, html.EscapeString(cfg.Voice))
	fmt.Fprintf(&b, `<prosody rate="%+d%%" pitch="%+d%%">`, cfg.Rate, cfg.Pitch)
	if t := strings.TrimSpace(title); t != "" {
		b.WriteString(html.EscapeString(t))
		b.WriteString(`<break time="900ms"/>`)
	}
	first := true
	for _, p := range strings.Split(strings.ReplaceAll(text, "\r", ""), "\n") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !first {
			b.WriteString(`<break time="700ms"/>`)
		}
		first = false
		b.WriteString(html.EscapeString(p))
	}
	b.WriteString(`</prosody></voice></speak>`)
	return b.String()
}

// Speak dobla un texto y devuelve el mp3.
func (c Client) Speak(ctx context.Context, cfg Config, title, text string) ([]byte, error) {
	base, err := c.base(cfg.Region)
	if err != nil {
		return nil, err
	}
	body := SSML(cfg, title, text)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, base+"/cognitiveservices/v1", bytes.NewBufferString(body))
	req.Header.Set("Ocp-Apim-Subscription-Key", strings.TrimSpace(cfg.Key))
	req.Header.Set("Content-Type", "application/ssml+xml")
	req.Header.Set("X-Microsoft-OutputFormat", "audio-24khz-48kbitrate-mono-mp3")
	req.Header.Set("User-Agent", "Cronica")
	resp, err := c.http().Do(req)
	if err != nil {
		return nil, fmt.Errorf("no llego a Azure (¿hay internet?): %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, statusErr(resp)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 50<<20))
	if err != nil {
		return nil, err
	}
	if len(b) < 100 {
		return nil, errors.New("Azure ha devuelto un audio vacío")
	}
	return b, nil
}

// Sample es el texto de «Probar voz».
const Sample = "Las tierras de Azeroth guardan muchas historias. Esta es la tuya: la de quien partió sin nombre y, paso a paso, empezó a ganárselo."
