//go:build !windows

package install

import "fmt"

func ensureWindowsPath(dir string) error {
	return fmt.Errorf("internal error: ensureWindowsPath chamado fora do Windows")
}
