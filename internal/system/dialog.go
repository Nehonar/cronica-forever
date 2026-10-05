package system

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Ask muestra una ventana con una pregunta de sí/no y devuelve la respuesta.
// ok es false si no hay forma de mostrar ventanas (por ejemplo, sin escritorio).
func Ask(title, question string) (yes, ok bool) {
	switch runtime.GOOS {
	case "windows":
		ps := "Add-Type -AssemblyName PresentationFramework; [System.Windows.MessageBox]::Show('" +
			psQuote(question) + "', '" + psQuote(title) + "', 'YesNo', 'Question')"
		out, err := exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", ps).Output()
		if err != nil {
			return false, false
		}
		return strings.TrimSpace(string(out)) == "Yes", true
	case "linux":
		if p, err := exec.LookPath("zenity"); err == nil {
			err := exec.Command(p, "--question", "--title="+title, "--text="+question, "--ok-label=Sí", "--cancel-label=No", "--width=420").Run()
			return err == nil, true
		}
		if p, err := exec.LookPath("kdialog"); err == nil {
			err := exec.Command(p, "--title", title, "--yesno", question).Run()
			return err == nil, true
		}
	case "darwin":
		script := `display dialog "` + strings.ReplaceAll(question, `"`, `'`) + `" with title "` + title + `" buttons {"No", "Sí"} default button "Sí"`
		out, err := exec.Command("osascript", "-e", script).Output()
		if err != nil {
			return false, true
		}
		return strings.Contains(string(out), "Sí"), true
	}
	return false, false
}

// HasDesktop indica si parece haber un escritorio donde mostrar ventanas.
func HasDesktop() bool {
	if runtime.GOOS != "linux" {
		return true
	}
	return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
}

// OpenTerminal abre una ventana de terminal visible que ejecuta command
// (bash en Linux/macOS, PowerShell en Windows) y espera a que el usuario la cierre.
func OpenTerminal(title, command string) error {
	switch runtime.GOOS {
	case "windows":
		ps := "$host.UI.RawUI.WindowTitle = '" + psQuote(title) + "'; " + command +
			"; Write-Host ''; Read-Host 'Pulsa Intro para cerrar esta ventana'"
		return exec.Command("cmd", "/c", "start", title, "powershell", "-NoProfile", "-NoExit", "-Command", ps).Start()
	case "darwin":
		script := `tell application "Terminal" to do script "` + strings.ReplaceAll(command, `"`, `\"`) + `"`
		return exec.Command("osascript", "-e", script).Start()
	}
	sh := command + `; echo; read -r -p "Pulsa Intro para cerrar esta ventana" _`
	candidates := [][]string{
		{"x-terminal-emulator", "-e", "bash", "-c", sh},
		{"gnome-terminal", "--title=" + title, "--", "bash", "-c", sh},
		{"konsole", "-p", "tabtitle=" + title, "-e", "bash", "-c", sh},
		{"xfce4-terminal", "--title=" + title, "-x", "bash", "-c", sh},
		{"kitty", "--title", title, "bash", "-c", sh},
		{"alacritty", "-t", title, "-e", "bash", "-c", sh},
		{"xterm", "-T", title, "-e", "bash", "-c", sh},
	}
	var last error = exec.ErrNotFound
	for _, c := range candidates {
		if _, err := exec.LookPath(c[0]); err != nil {
			continue
		}
		if last = exec.Command(c[0], c[1:]...).Start(); last == nil {
			return nil
		}
	}
	return last
}
