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
		"-c",
		"core.quotepath=false",
		"fast-export",
		"--show-original-ids",
		"--signed-tags=strip",
		"--fake-missing-tagger",
		"--tag-of-filtered-object=rewrite",
		"--use-done-feature",
		"--mark-tags",    // git >= 2.24.0
		"--reencode=yes", // git >= 2.23.0
		repo.context.opts.branch,
		"--no-data",
	}
	// drop "--no-data"
	if repo.context.opts.lfs {
		args = args[:len(args)-1]
	}

	cmd := repo.GitCommand(args...)
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
