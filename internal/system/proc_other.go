//go:build !windows

package system

import (
	"fmt"
	"os"
	"os/exec"
)

// Hide no hace nada fuera de Windows (allí no se abren ventanas de consola).
func Hide(cmd *exec.Cmd) {}

// Alert muestra una ventana de aviso si hay escritorio y, si no, lo escribe en la terminal.
func Alert(title, message string) {
	if HasDesktop() {
		if p, err := exec.LookPath("zenity"); err == nil {
			exec.Command(p, "--warning", "--title="+title, "--text="+message, "--width=420").Run()
			return
		}
		if p, err := exec.LookPath("kdialog"); err == nil {
			exec.Command(p, "--title", title, "--sorry", message).Run()
			return
		}
	}
	fmt.Fprintf(os.Stderr, "%s: %s\n", title, message)
}

func messageBoxYesNo(title, question string) bool  { return false }
func pickFolderNative(title string) (string, bool) { return "", false }
