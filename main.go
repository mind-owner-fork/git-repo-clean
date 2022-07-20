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
	if repo.context.opts.lfs && AskForMigrateToLFS() {
		// can't run lfs-migrate in bare repo
		// git lfs track must be run in a work tree.
		if repo.context.bare {
			PrintLocalWithYellowln("bare repo error")
			os.Exit(1)
		}
	} else {
		repo.context.opts.lfs = false
	}
	// filter data
	repo.Parser()

	repo.context.CleanUp()
	repo.context.Prompt()
}
