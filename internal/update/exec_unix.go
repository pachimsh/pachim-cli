//go:build !windows

package update

import (
	"os"
	"syscall"
)

func syscallExec(executable string, args []string) error {
	return syscall.Exec(executable, args, os.Environ())
}
