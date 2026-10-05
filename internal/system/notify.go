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
		Toast(title, message)
		return
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

// Toast muestra un aviso discreto que no interrumpe (una notificación del sistema).
func Toast(title, message string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("notify-send", "--app-name=Crónica", title, message)
	case "windows":
		ps := `[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
$t = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02)
$x = $t.GetElementsByTagName('text')
$x.Item(0).AppendChild($t.CreateTextNode('` + psQuote(title) + `')) | Out-Null
$x.Item(1).AppendChild($t.CreateTextNode('` + psQuote(message) + `')) | Out-Null
$n = [Windows.UI.Notifications.ToastNotification]::new($t)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\WindowsPowerShell\v1.0\powershell.exe').Show($n)`
		cmd = exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", ps)
	case "darwin":
		Notify(title, message)
		return
	default:
		return
	}
	if err := cmd.Start(); err == nil {
		go cmd.Wait()
	}
}
