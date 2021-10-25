package main

import (
	"io"
	"os"
)

// run a git-fast-export process
// but keep repo path the same with git-fast-export
// return a Writer for stream pipeline to feed data into this process
func (repo *Repository) FastImportOut() (io.WriteCloser, error) {
	args := []string{
		"-C",
		repo.path,
		"-c",
		"core.ignorecase=false",
		"fast-import",
		"--quiet",
		"--force",
		"--date-format=raw-permissive",
	}
	cmd := repo.GitCommand(args...)

	in, err := cmd.StdinPipe()
	if err != nil {
		in.Close()
		return nil, err
	}
	cmd.Stderr = os.Stderr

	cmd.Start()

	return in, nil
}
