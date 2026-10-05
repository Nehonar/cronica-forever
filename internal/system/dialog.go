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
		return messageBoxYesNo(title, question), true
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

// PickFolder abre el selector de carpetas del sistema. Devuelve "" si se
// cancela; ok=false si no hay forma de mostrarlo.
func PickFolder(title string) (path string, ok bool) {
	switch runtime.GOOS {
	case "windows":
		return pickFolderNative(title)
	case "linux":
		if !HasDesktop() {
			return "", false
		}
		if p, err := exec.LookPath("zenity"); err == nil {
			out, err := exec.Command(p, "--file-selection", "--directory", "--title="+title).Output()
			if err != nil {
				return "", true // cancelado
			}
			return strings.TrimSpace(string(out)), true
		}
		if p, err := exec.LookPath("kdialog"); err == nil {
			out, err := exec.Command(p, "--getexistingdirectory", ".", "--title", title).Output()
			if err != nil {
				return "", true
			}
			return strings.TrimSpace(string(out)), true
		}
	}
	return "", false
}
