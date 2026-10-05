//go:build !windows

package system

import "errors"

func setRunKey(string) error { return errors.New("solo en Windows") }
func deleteRunKey() error    { return nil }
func runKeySet() bool        { return false }
