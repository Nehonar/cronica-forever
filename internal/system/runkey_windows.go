//go:build windows

package system

import (
	"errors"

	"golang.org/x/sys/windows/registry"
)

// Arranque con Windows: la clave «Run» del usuario (la que se ve en
// Administrador de tareas → Inicio). No necesita permisos de administrador.
const runKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`
const runValue = "Cronica"

func setRunKey(cmdline string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue(runValue, cmdline)
}

func deleteRunKey() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return nil
	}
	defer k.Close()
	if err := k.DeleteValue(runValue); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return err
	}
	return nil
}

func runKeySet() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(runValue)
	return err == nil
}
