//go:build windows

package system

import (
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

const createNoWindow = 0x08000000

// Hide hace que el comando se ejecute sin abrir ninguna ventana de consola.
func Hide(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
}

var procMessageBox = user32.NewProc("MessageBoxW")

// Alert muestra una ventana de aviso (con un solo botón Aceptar) y espera a que se cierre.
func Alert(title, message string) {
	t, _ := syscall.UTF16PtrFromString(title)
	m, _ := syscall.UTF16PtrFromString(message)
	// MB_OK | MB_ICONWARNING | MB_SETFOREGROUND | MB_TOPMOST
	procMessageBox.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), 0x30|0x10000|0x40000)
}

// messageBoxYesNo pregunta con una ventana nativa de Windows (sin PowerShell).
func messageBoxYesNo(title, question string) bool {
	t, _ := syscall.UTF16PtrFromString(title)
	m, _ := syscall.UTF16PtrFromString(question)
	// MB_YESNO | MB_ICONQUESTION | MB_SETFOREGROUND | MB_TOPMOST
	r, _, _ := procMessageBox.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), 0x4|0x20|0x10000|0x40000)
	return r == 6 // IDYES
}

var (
	shell32               = syscall.NewLazyDLL("shell32.dll")
	ole32                 = syscall.NewLazyDLL("ole32.dll")
	user32                = syscall.NewLazyDLL("user32.dll")
	procSHBrowseForFolder = shell32.NewProc("SHBrowseForFolderW")
	procSHGetPathFromID   = shell32.NewProc("SHGetPathFromIDListW")
	procCoInitializeEx    = ole32.NewProc("CoInitializeEx")
	procCoUninitialize    = ole32.NewProc("CoUninitialize")
	procCoTaskMemFree     = ole32.NewProc("CoTaskMemFree")
	procSetWindowPos      = user32.NewProc("SetWindowPos")
	procSetForeground     = user32.NewProc("SetForegroundWindow")
)

type browseInfo struct {
	Owner       uintptr
	Root        uintptr
	DisplayName *uint16
	Title       *uint16
	Flags       uint32
	Callback    uintptr
	Param       uintptr
	Image       int32
}

// Al abrirse el selector, ponerlo delante del navegador.
var browseCallback = syscall.NewCallback(func(hwnd, msg, lParam, data uintptr) uintptr {
	if msg == 1 { // BFFM_INITIALIZED
		const swpNoSize, swpNoMove, swpShow = 0x1, 0x2, 0x40
		hwndTopmost := ^uintptr(0) // (HWND)-1
		procSetWindowPos.Call(hwnd, hwndTopmost, 0, 0, 0, 0, swpNoSize|swpNoMove|swpShow)
		procSetForeground.Call(hwnd)
	}
	return 0
})

// pickFolderNative abre el selector de carpetas de Windows (sin PowerShell).
func pickFolderNative(title string) (string, bool) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	procCoInitializeEx.Call(0, 0x2) // COINIT_APARTMENTTHREADED
	defer procCoUninitialize.Call()

	name := make([]uint16, 260)
	t, _ := syscall.UTF16PtrFromString(title)
	bi := browseInfo{
		DisplayName: &name[0],
		Title:       t,
		Flags:       0x1 | 0x40 | 0x200, // solo carpetas, diálogo nuevo, sin «Nueva carpeta»
		Callback:    browseCallback,
	}
	pidl, _, _ := procSHBrowseForFolder.Call(uintptr(unsafe.Pointer(&bi)))
	if pidl == 0 {
		return "", true // cancelado
	}
	defer procCoTaskMemFree.Call(pidl)
	path := make([]uint16, 1024)
	if r, _, _ := procSHGetPathFromID.Call(pidl, uintptr(unsafe.Pointer(&path[0]))); r == 0 {
		return "", true
	}
	return syscall.UTF16ToString(path), true
}

var (
	procEnumWindows     = user32.NewProc("EnumWindows")
	procGetWindowText   = user32.NewProc("GetWindowTextW")
	procIsWindowVisible = user32.NewProc("IsWindowVisible")
	procIsIconic        = user32.NewProc("IsIconic")
	procShowWindowW     = user32.NewProc("ShowWindow")
)

// FocusWindow trae delante la ventana cuyo título contiene text (por ejemplo,
// la del navegador con la pestaña de Crónica activa). Devuelve false si no la hay.
func FocusWindow(text string) bool {
	var found uintptr
	cb := syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
		if v, _, _ := procIsWindowVisible.Call(hwnd); v == 0 {
			return 1
		}
		buf := make([]uint16, 512)
		n, _, _ := procGetWindowText.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		if n > 0 && strings.Contains(syscall.UTF16ToString(buf[:n]), text) {
			found = hwnd
			return 0
		}
		return 1
	})
	procEnumWindows.Call(cb, 0)
	if found == 0 {
		return false
	}
	if ic, _, _ := procIsIconic.Call(found); ic != 0 {
		procShowWindowW.Call(found, 9) // SW_RESTORE
	}
	procSetForeground.Call(found)
	return true
}
