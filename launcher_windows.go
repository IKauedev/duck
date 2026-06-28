//go:build windows

package main

import (
	"os"
	"os/exec"
	"syscall"
	"unsafe"
)

var kernel32 = syscall.NewLazyDLL("kernel32.dll")
var procGetConsoleProcessList = kernel32.NewProc("GetConsoleProcessList")

// hasConsole retorna true quando o processo está rodando dentro de um terminal
// real (PowerShell, cmd, Windows Terminal, VS Code…).
//
// A lógica usa GetConsoleProcessList: se só o próprio processo está anexado ao
// console (contagem == 1), o console foi criado exclusivamente para nós pelo
// Windows — o que acontece quando o programa é lançado via Win+R, duplo-clique
// ou atalho de desktop. Nesse caso retornamos false para que launchInTerminal()
// abra uma janela de terminal adequada.
func hasConsole() bool {
	// Aloca um buffer pequeno; precisamos apenas da contagem.
	buf := make([]uint32, 16)
	ret, _, _ := procGetConsoleProcessList.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)
	// ret == 0  → erro (sem console algum)
	// ret == 1  → apenas nós mesmos: console criado pelo Win+R / atalho
	// ret >= 2  → há um shell-pai: já estamos num terminal real
	return ret >= 2
}

// launchInTerminal reabre o duck em uma nova janela de terminal rodando "duck shell".
// Tenta Windows Terminal (wt), depois PowerShell 7 (pwsh), depois cmd.exe clássico.
func launchInTerminal() {
	self, err := os.Executable()
	if err != nil {
		self = "duck.exe"
	}

	shellCmd := `"` + self + `" shell`

	// Tenta Windows Terminal (wt.exe)
	if wt, err := exec.LookPath("wt.exe"); err == nil {
		cmd := exec.Command(wt,
			"--title", "Duck Shell",
			"--",
			"cmd.exe", "/k", shellCmd,
		)
		// wt.exe é um app GUI — não precisa de CREATE_NEW_CONSOLE
		if cmd.Start() == nil {
			return
		}
	}

	// Tenta PowerShell 7 (pwsh.exe)
	if pwsh, err := exec.LookPath("pwsh.exe"); err == nil {
		cmd := exec.Command(pwsh, "-NoExit", "-Command", shellCmd)
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000010}
		if cmd.Start() == nil {
			return
		}
	}

	// Fallback: cmd.exe clássico com /k para manter a janela aberta
	cmd := exec.Command("cmd.exe", "/k", shellCmd)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000010}
	_ = cmd.Start()
}
