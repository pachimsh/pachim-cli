package git

import (
	"os/exec"
	"strings"
)

type Head struct {
	Branch string
	Commit string
}

func CurrentHead(dir string) (*Head, bool) {
	if !IsRepo(dir) {
		return nil, false
	}

	branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	branchCmd.Dir = dir
	branchOut, err := branchCmd.Output()
	if err != nil {
		return nil, false
	}

	commitCmd := exec.Command("git", "rev-parse", "HEAD")
	commitCmd.Dir = dir
	commitOut, err := commitCmd.Output()
	if err != nil {
		return nil, false
	}

	branch := strings.TrimSpace(string(branchOut))
	if branch == "HEAD" {
		branch = ""
	}

	return &Head{
		Branch: branch,
		Commit: strings.TrimSpace(string(commitOut)),
	}, true
}

func IsRepo(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = dir
	return cmd.Run() == nil
}
