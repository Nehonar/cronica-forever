package app

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitT(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %s", args, out)
	}
	return strings.TrimSpace(string(out))
}

// Git recién instalado (sin nombre ni correo) y un intento fallido anterior:
// Publish tiene que firmar como el dueño y subir lo que quedó pendiente.
func TestPublishWithoutIdentityAndRetries(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	root := t.TempDir()
	bare := filepath.Join(root, "origen.git")
	exec.Command("git", "init", "-q", "--bare", "-b", "master", bare).Run()
	repo := filepath.Join(root, "web")
	exec.Command("git", "clone", "-q", bare, repo).Run()
	gitT(t, repo, "remote", "set-url", "origin", bare)
	// Simular que el remoto es GitHub para sacar el dueño: usamos pushurl local.
	gitT(t, repo, "config", "remote.origin.url", "https://github.com/Nehonar/cronica-forever")
	gitT(t, repo, "config", "remote.origin.pushurl", bare)
	gitT(t, repo, "config", "url."+bare+".insteadOf", "https://github.com/Nehonar/cronica-forever")
	os.MkdirAll(filepath.Join(repo, "personajes"), 0o755)
	os.WriteFile(filepath.Join(repo, "personajes", "Nehonar.json"), []byte("{}"), 0o644)
	gitT(t, repo, "checkout", "-q", "-b", "master")

	// Primer intento: rama sin upstream → el push falla pero el commit queda hecho.
	err := Publish(repo, "primera", io.Discard)
	t.Logf("primer intento: %v", err)
	if a := gitT(t, repo, "log", "-1", "--format=%ae"); a != "Nehonar@users.noreply.github.com" {
		t.Fatalf("autor = %q", a)
	}
	gitT(t, repo, "push", "-q", "-u", "origin", "master")
	// Un cambio nuevo se sube.
	os.WriteFile(filepath.Join(repo, "personajes", "Nehonar.json"), []byte(`{"a":1}`), 0o644)
	if err := Publish(repo, "segunda", io.Discard); err != nil {
		t.Fatal(err)
	}
	if n := gitT(t, repo, "rev-list", "--count", "@{u}..HEAD"); n != "0" {
		t.Fatalf("quedan %s commits sin subir", n)
	}
	// Un commit que se quedó sin subir (sin cambios nuevos) también se sube.
	os.WriteFile(filepath.Join(repo, "personajes", "Nehonar.json"), []byte(`{"a":2}`), 0o644)
	gitT(t, repo, "add", "-A")
	gitT(t, repo, "-c", "user.name=x", "-c", "user.email=x@x", "commit", "-q", "-m", "local")
	if err := Publish(repo, "tercera", io.Discard); err != nil {
		t.Fatal(err)
	}
	if n := gitT(t, repo, "rev-list", "--count", "@{u}..HEAD"); n != "0" {
		t.Fatalf("quedan %s commits sin subir", n)
	}
}
