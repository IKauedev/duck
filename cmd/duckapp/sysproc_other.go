//go:build !windows

package main

import "syscall"

// windowsNewWindow não faz nada em plataformas não-Windows
func windowsNewWindow() *syscall.SysProcAttr {
	return nil
}
