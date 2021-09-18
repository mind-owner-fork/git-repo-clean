package main

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cli/safeexec"
	"github.com/github/git-sizer/git"
)

func findGitBin() (string, error) {
	gitBin, err := safeexec.LookPath("git")
	if err != nil {
		return "", err
	}

	gitBin, err = filepath.Abs(gitBin)
	if err != nil {
		return "", err
	}

	return gitBin, nil
}

type Repository struct {
	git.Repository
	path   string
	gitBin string
}

func (repo Repository) gitCommand(callerArgs ...string) *exec.Cmd {
	args := []string{
		"--no-replace-objects",
		"-c",
		"advice.graftFileDeprecated=false",
		"-C",
		repo.path}

	args = append(args, callerArgs...)

	cmd := exec.Command(repo.gitBin, args...)
	cmd.Env = append(
		os.Environ(),
		"GIT_DIR"+repo.path,
		// Disable grafts when running our commands:
		"GIT_GRAFT_FILE="+os.DevNull,
	)

	return cmd
}
