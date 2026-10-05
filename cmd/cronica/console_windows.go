//go:build windows

package main

import (
	"bufio"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32              = syscall.NewLazyDLL("kernel32.dll")
	user32                = syscall.NewLazyDLL("user32.dll")
	procGetConsoleWindow  = kernel32.NewProc("GetConsoleWindow")
	procGetConsoleProcess = kernel32.NewProc("GetConsoleProcessList")
	procShowWindow        = user32.NewProc("ShowWindow")
	procAttachConsole     = kernel32.NewProc("AttachConsole")
)

// attachParentConsole: el programa se compila como aplicación de ventana (sin
// consola negra al hacer doble clic). Si se ejecuta desde una terminal con
// órdenes («cronica demo»…), escribe en esa terminal.
func attachParentConsole() {
	if r, _, _ := procAttachConsole.Call(uintptr(^uint32(0))); r == 0 { // ATTACH_PARENT_PROCESS
		return
	}
	if f, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0); err == nil {
		os.Stdout, os.Stderr = f, f
	}
	if f, err := os.OpenFile("CONIN$", os.O_RDONLY, 0); err == nil {
		os.Stdin = f
	}
	fmt.Println()
}

// ownConsole indica si la ventana de consola es solo nuestra (se ha abierto con
// doble clic) y no una terminal que el usuario ya tenía abierta.
func ownConsole() bool {
	var ids [4]uint32
	n, _, _ := procGetConsoleProcess.Call(uintptr(unsafe.Pointer(&ids[0])), 4)
	return n == 1
}

// hideOwnConsole oculta la ventana negra si la abrió el doble clic.
func hideOwnConsole() {
	if !ownConsole() {
		return
	}
	if h, _, _ := procGetConsoleWindow.Call(); h != 0 {
		procShowWindow.Call(h, 0) // SW_HIDE
	}
}

// pauseIfOwnConsole espera a que el usuario lea el resultado antes de cerrar la ventana.
func pauseIfOwnConsole() {
	if !ownConsole() {
		return
	}
	fmt.Print("\nPulsa Intro para cerrar esta ventana…")
	bufio.NewReader(os.Stdin).ReadString('\n')
}
