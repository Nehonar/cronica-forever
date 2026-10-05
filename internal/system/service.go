package system

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Service describe cómo arrancar «cronica vigilar» al iniciar sesión.
type Service struct {
	Exe     string // ruta absoluta del programa
	Config  string // ruta absoluta de cronica.json
	WorkDir string // carpeta del repositorio
	LogFile string // archivo de registro
}

const taskName = "Cronica"

// DefaultLogFile es donde el servicio deja su registro.
func DefaultLogFile() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "cronica", "cronica.log")
}

func (s Service) args() []string {
	return []string{"bandeja", "-config", s.Config, "-registro", s.LogFile, "-arranque"}
}

// Install deja el programa preparado para arrancar solo al iniciar sesión y lo arranca ya.
func Install(s Service) (string, error) {
	if err := os.MkdirAll(filepath.Dir(s.LogFile), 0o755); err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "linux":
		return installAutostart(s)
	case "windows":
		return installWindows(s)
	}
	return "", fmt.Errorf("el arranque automático aún no está preparado para %s", runtime.GOOS)
}

// Uninstall quita el arranque automático y cierra Crónica si está abierta.
func Uninstall() (string, error) {
	switch runtime.GOOS {
	case "linux":
		exec.Command("pkill", "-f", "cronica bandeja").Run()
	case "windows":
		hiddenCmd("schtasks", "/End", "/TN", taskName).Run()
	}
	if err := RemoveAutostart(); err != nil {
		return "", err
	}
	return "Arranque automático quitado.", nil
}

// RemoveAutostart quita el arranque automático sin cerrar el programa.
func RemoveAutostart() error {
	switch runtime.GOOS {
	case "linux":
		removeOldSystemd()
		if p, err := autostartPath(); err == nil {
			if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
		return nil
	case "windows":
		cmd := hiddenCmd("schtasks", "/Delete", "/TN", taskName, "/F")
		Hide(cmd)
		if out, err := cmd.CombinedOutput(); err != nil && Installed() {
			return fmt.Errorf("schtasks: %s", strings.TrimSpace(string(out)))
		}
		if dir, err := os.UserConfigDir(); err == nil {
			os.Remove(filepath.Join(dir, "Cronica", "iniciar.ps1"))
		}
		return nil
	}
	return fmt.Errorf("no hay arranque automático en %s", runtime.GOOS)
}

// ---------- Linux: servicio de usuario de systemd ----------

func systemdUnitPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "systemd", "user", "cronica.service"), nil
}

func installSystemd(s Service) (string, error) {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return "", fmt.Errorf("no encuentro systemctl; tu sistema no usa systemd")
	}
	unit, err := systemdUnitPath()
	if err != nil {
		return "", err
	}
	q := func(p string) string { return `"` + strings.ReplaceAll(p, `"`, `\"`) + `"` }
	var exec_ strings.Builder
	exec_.WriteString(q(s.Exe))
	for _, a := range s.args() {
		exec_.WriteString(" " + q(a))
	}
	env := "Environment=" + q("PATH="+os.Getenv("PATH")) + "\n"
	for _, k := range []string{"DISPLAY", "WAYLAND_DISPLAY", "DBUS_SESSION_BUS_ADDRESS", "XDG_RUNTIME_DIR"} {
		if v := os.Getenv(k); v != "" {
			env += "Environment=" + q(k+"="+v) + "\n"
		}
	}
	content := fmt.Sprintf(`[Unit]
Description=Crónica de Forever: narra tus sesiones de WoW
After=network-online.target

[Service]
Type=simple
WorkingDirectory=%s
ExecStart=%s
%sRestart=on-failure
RestartSec=30

[Install]
WantedBy=default.target
`, s.WorkDir, exec_.String(), env)
	if err := os.MkdirAll(filepath.Dir(unit), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(unit, []byte(content), 0o644); err != nil {
		return "", err
	}
	for _, args := range [][]string{{"--user", "daemon-reload"}, {"--user", "enable", "--now", "cronica.service"}, {"--user", "restart", "cronica.service"}} {
		if out, err := exec.Command("systemctl", args...).CombinedOutput(); err != nil {
			return "", fmt.Errorf("systemctl %s: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
		}
	}
	return fmt.Sprintf("Instalado como servicio de usuario (%s).\nRegistro: %s  (o: journalctl --user -u cronica)", unit, s.LogFile), nil
}

// ---------- Windows: tarea programada al iniciar sesión ----------

func installWindows(s Service) (string, error) {
	dir, err := os.UserConfigDir() // %APPDATA%
	if err != nil {
		return "", err
	}
	scriptDir := filepath.Join(dir, "Cronica")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		return "", err
	}
	script := filepath.Join(scriptDir, "iniciar.ps1")
	var b strings.Builder
	fmt.Fprintf(&b, "Set-Location -LiteralPath '%s'\r\n", psQuote(s.WorkDir))
	fmt.Fprintf(&b, "& '%s'", psQuote(s.Exe))
	for _, a := range s.args() {
		fmt.Fprintf(&b, " '%s'", psQuote(a))
	}
	b.WriteString("\r\n")
	if err := os.WriteFile(script, []byte(b.String()), 0o644); err != nil {
		return "", err
	}
	tr := `powershell.exe -NoProfile -WindowStyle Hidden -ExecutionPolicy Bypass -File "` + script + `"`
	out, err := hiddenCmd("schtasks", "/Create", "/TN", taskName, "/TR", tr, "/SC", "ONLOGON", "/RL", "LIMITED", "/F").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("schtasks: %s", strings.TrimSpace(string(out)))
	}
	hiddenCmd("schtasks", "/Run", "/TN", taskName).Run()
	return fmt.Sprintf("Instalado como tarea programada «%s» al iniciar sesión.\nRegistro: %s", taskName, s.LogFile), nil
}

// ---------- Linux: autoarranque del escritorio (para el icono de la bandeja) ----------

func autostartPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "autostart", "cronica.desktop"), nil
}

// removeOldSystemd quita el servicio de versiones anteriores, que no tenía bandeja.
func removeOldSystemd() {
	unit, err := systemdUnitPath()
	if err != nil {
		return
	}
	if _, err := os.Stat(unit); err != nil {
		return
	}
	exec.Command("systemctl", "--user", "disable", "--now", "cronica.service").Run()
	os.Remove(unit)
	exec.Command("systemctl", "--user", "daemon-reload").Run()
}

func installAutostart(s Service) (string, error) {
	removeOldSystemd()
	p, err := autostartPath()
	if err != nil {
		return "", err
	}
	q := func(a string) string {
		r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "`", "\\`", "$", `\$`)
		return `"` + r.Replace(a) + `"`
	}
	var b strings.Builder
	b.WriteString(q(s.Exe))
	for _, a := range s.args() {
		b.WriteString(" " + q(a))
	}
	content := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Crónica de Forever
Comment=Narra tus sesiones de WoW: Forever
Exec=%s
Path=%s
Terminal=false
X-GNOME-Autostart-enabled=true
`, b.String(), s.WorkDir)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		return "", err
	}
	// Arrancarla ya, sin esperar al próximo inicio de sesión.
	cmd := exec.Command(s.Exe, "bandeja", "-config", s.Config, "-registro", s.LogFile)
	cmd.Dir = s.WorkDir
	if err := cmd.Start(); err == nil {
		go cmd.Wait()
	}
	return fmt.Sprintf("Instalado en el autoarranque del escritorio (%s).\nRegistro: %s", p, s.LogFile), nil
}

// Installed indica si el arranque automático está configurado.
func Installed() bool {
	switch runtime.GOOS {
	case "linux":
		p, err := autostartPath()
		if err != nil {
			return false
		}
		_, err = os.Stat(p)
		return err == nil
	case "windows":
		return hiddenCmd("schtasks", "/Query", "/TN", taskName).Run() == nil
	}
	return false
}

// hiddenCmd prepara un comando que no abre ventana de consola.
func hiddenCmd(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	Hide(cmd)
	return cmd
}
