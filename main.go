package main

import (
	"os"
)

func main() {
	filter, err := NewFilter(os.Args[1:])
	if err != nil {
		LocalFprintf(os.Stderr, "init repo filter error")
		os.Exit(1)
	}
	// repo backup
	filter.repo.BackUp()

	// ask for lfs migrate
	if filter.repo.opts.lfs && AskForMigrateToLFS() {
		// can't run lfs-migrate in bare repo
		// git lfs track must be run in a work tree.
		if filter.repo.bare {
			PrintLocalWithYellowln("bare repo error")
			os.Exit(1)
		}
	} else {
		filter.repo.opts.lfs = false
	}
	// filter data
	filter.Parser()

	filter.repo.CleanUp()
	filter.repo.Prompt()
}
