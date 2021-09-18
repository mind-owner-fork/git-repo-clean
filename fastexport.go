package main

import (
	"bufio"
	"io"
	"os"
	"os/exec"
)

// fast-export output stream iterater
type FEOutPutIter struct {
	cmd     *exec.Cmd
	out     io.ReadCloser
	f       *bufio.Reader
	errChan <-chan error
}

func (repo *Repository) NewFastExportIter() (*FEOutPutIter, error) {
	args := []string{
		"-C",
		repo.path,
		"fast-export",
		"--show-original-ids",
		"--use-done-feature",
		"--mark-tags",
		"--reencode=yes",
		"--all",
	}

	cmd := repo.gitCommand(args...)
	out, err := cmd.StdoutPipe()
	if err != nil {
		out.Close()
		return nil, err
	}

	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	return &FEOutPutIter{
		cmd:     cmd,
		out:     out,
		f:       bufio.NewReader(out),
		errChan: make(chan error, 1),
	}, nil
}

// get data line by line from output stream
func (iter *FEOutPutIter) Next() (string, error) {
	errChan := make(chan error, 1)
	line, err := iter.f.ReadString('\n')
	if err != nil {
		if err != io.EOF {
			errChan <- err
			return "", err
		}
		return "", nil
	}
	return line, nil
}

func (iter *FEOutPutIter) Close() error {
	err := iter.out.Close()
	err2 := iter.cmd.Wait()
	if err == nil {
		err = err2
	}
	return err
}
