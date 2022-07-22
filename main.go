package main

import (
	"os"
)

var op Options

func main() {
	if err := ParseOptions(os.Args[1:]); err != nil {
		PrintLocalWithRedln("Parse Option error")
		os.Exit(1)
	}
	var repo = NewRepository()
	// repo backup
	BackUp(repo.context.gitBin, repo.context.workDir)

	// ask for lfs migrate
	if repo.context.opts.lfs {
		if ok := AskForMigrateToLFS(); !ok {
			repo.context.opts.lfs = false
		}
	}
	// filter data
	repo.Parser()

	repo.context.CleanUp()
	repo.context.Prompt()
}
