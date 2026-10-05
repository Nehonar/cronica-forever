//go:build windows

package system

import (
	"os/exec"
	"syscall"
	"unsafe"
)

const createNoWindow = 0x08000000

// Hide hace que el comando se ejecute sin abrir ninguna ventana de consola.
func Hide(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
}

var procMessageBox = syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW")

// Alert muestra una ventana de aviso (con un solo botón Aceptar) y espera a que se cierre.
func Alert(title, message string) {
	t, _ := syscall.UTF16PtrFromString(title)
	m, _ := syscall.UTF16PtrFromString(message)
	// MB_OK | MB_ICONWARNING | MB_SETFOREGROUND | MB_TOPMOST
	procMessageBox.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), 0x30|0x10000|0x40000)
}
