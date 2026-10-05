// Package system contiene lo que depende del sistema operativo: avisos en el
// escritorio y el arranque automático del programa al iniciar sesión.
package system

import (
	"os/exec"
	"runtime"
	"strings"
)

// Notify muestra un aviso en el escritorio. No espera a que el usuario lo cierre
// y, si el sistema no tiene forma de mostrarlo, no hace nada.
func Notify(title, message string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("notify-send", "--app-name=Crónica", "--urgency=critical", title, message)
	case "windows":
		ps := "Add-Type -AssemblyName PresentationFramework; [System.Windows.MessageBox]::Show('" +
			psQuote(message) + "', '" + psQuote(title) + "', 'OK', 'Warning') | Out-Null"
		cmd = exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", ps)
	case "darwin":
		cmd = exec.Command("osascript", "-e", `display notification "`+strings.ReplaceAll(message, `"`, `'`)+`" with title "`+strings.ReplaceAll(title, `"`, `'`)+`"`)
	default:
		return
	}
	if err := cmd.Start(); err == nil {
		go cmd.Wait()
	}
}

// psQuote escapa una cadena para ir entre comillas simples en PowerShell.
func psQuote(s string) string { return strings.ReplaceAll(s, "'", "''") }
