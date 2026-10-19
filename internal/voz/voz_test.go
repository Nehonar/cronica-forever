package voz

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeAzure imita los dos puntos de la API que se usan.
func fakeAzure(t *testing.T, key string, got *string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Ocp-Apim-Subscription-Key") != key {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/cognitiveservices/voices/list":
			io.WriteString(w, `[{"ShortName":"es-ES-DarioNeural","LocalName":"Darío","Gender":"Male","Locale":"es-ES","VoiceType":"Neural"},{"ShortName":"es-ES-AlvaroNeural","LocalName":"Álvaro","Gender":"Male","Locale":"es-ES","VoiceType":"Neural"},
			{"ShortName":"es-MX-JorgeNeural","LocalName":"Jorge","Gender":"Male","Locale":"es-MX","VoiceType":"Neural"},
			{"ShortName":"es-ES-ElviraNeural","LocalName":"Elvira","Gender":"Female","Locale":"es-ES","VoiceType":"Neural"}]`)
		case "/cognitiveservices/v1":
			b, _ := io.ReadAll(r.Body)
			*got = string(b)
			if r.Header.Get("X-Microsoft-OutputFormat") == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "audio/mpeg")
			w.Write([]byte(strings.Repeat("\xff\xfb", 200)))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestVoicesAndSpeak(t *testing.T) {
	var got string
	srv := fakeAzure(t, "buena", &got)
	defer srv.Close()
	c := Client{Base: srv.URL}
	cfg := Config{Active: true, Region: "westeurope", Key: "buena"}
	vs, err := c.Voices(context.Background(), cfg)
	if err != nil || len(vs) != 3 || vs[0].Name != "Álvaro" || vs[1].Name != "Darío" || vs[2].Gender != "Mujer" {
		t.Fatalf("voces: %+v %v", vs, err)
	}
	mp3, err := c.Speak(context.Background(), cfg, "El & la <espada>", "Primer párrafo.\n\nSegundo párrafo.")
	if err != nil || len(mp3) != 400 {
		t.Fatalf("doblar: %d %v", len(mp3), err)
	}
	for _, want := range []string{`name="es-ES-AlvaroNeural"`, `rate="-8%"`, `pitch="-4%"`, "El &amp; la &lt;espada&gt;", `Primer párrafo.<break time="700ms"/>Segundo párrafo.`} {
		if !strings.Contains(got, want) {
			t.Errorf("falta %q en %s", want, got)
		}
	}
	if _, err := c.Voices(context.Background(), Config{Region: "westeurope", Key: "mala"}); err != ErrKey {
		t.Fatalf("clave mala: %v", err)
	}
	if _, err := (Client{}).Voices(context.Background(), Config{Region: "west europe/../x", Key: "x"}); err == nil {
		t.Fatal("región rara aceptada")
	}
}
