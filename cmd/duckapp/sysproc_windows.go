//go:build windows

package main

import "syscall"

const createNewConsole = 0x00000010

// windowsNewWindow retorna SysProcAttr que cria uma nova janela de console no Windows
func windowsNewWindow() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: createNewConsole,
	}
}
