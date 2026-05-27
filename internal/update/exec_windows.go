//go:build windows

package update

import (
	"os"
	"os/exec"
)

func syscallExec(executable string, args []string) error {
	cmd := exec.Command(executable, args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
