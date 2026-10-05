//go:build !windows

package main

func hideOwnConsole()      {}
func pauseIfOwnConsole()   {}
func attachParentConsole() {}
