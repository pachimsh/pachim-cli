package git

import (
	"os/exec"
	"strings"
)

type WorkingTreeStatus struct {
	Dirty         bool
	ChangedFiles  int
	CurrentBranch string
}

func WorkingTreeStatusIn(dir string) (*WorkingTreeStatus, bool) {
	if !IsRepo(dir) {
		return nil, false
	}

	status := &WorkingTreeStatus{}

	if head, ok := CurrentHead(dir); ok && head.Branch != "" {
		status.CurrentBranch = head.Branch
	}

	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return status, true
	}

	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		status.ChangedFiles++
	}

	status.Dirty = status.ChangedFiles > 0
	return status, true
}

func LocalBranches(dir string) ([]string, bool) {
	if !IsRepo(dir) {
		return nil, false
	}

	cmd := exec.Command("git", "branch", "--format=%(refname:short)")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return nil, false
	}

	var branches []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			branches = append(branches, line)
		}
	}

	return branches, len(branches) > 0
}
