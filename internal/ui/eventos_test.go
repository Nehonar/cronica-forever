package ui

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Nehonar/cronica-forever/internal/app"
)

func TestNavigateSendsToOpenPage(t *testing.T) {
	s := &Server{Runner: &app.Runner{Cfg: app.Config{Repo: t.TempDir()}}}
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()
	if s.Navigate("/misiones/") {
		t.Fatal("sin páginas abiertas no debe navegar")
	}
	req, _ := http.NewRequest("GET", strings.Replace(ts.URL, "127.0.0.1", "localhost", 1)+"/api/eventos", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	rd := bufio.NewReader(resp.Body)
	rd.ReadString('\n') // «: hola»
	deadline := time.Now().Add(2 * time.Second)
	for !s.Navigate("/misiones/") {
		if time.Now().After(deadline) {
			t.Fatal("la página no se ha registrado")
		}
		time.Sleep(10 * time.Millisecond)
	}
	var got string
	for !strings.HasPrefix(got, "data:") {
		got, _ = rd.ReadString('\n')
	}
	if !strings.Contains(got, `"/misiones/"`) {
		t.Fatalf("evento = %q", got)
	}
}
