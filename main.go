package main

import (
	"fmt"
	"os"

	"github.com/github/git-sizer/git"
)

func main() {
	// parse argument
	var op = Options{}
	if err := op.OptionInit(os.Args[1:]); err != nil {
		fmt.Println("init error")
		os.Exit(1)
	}

	// create repo
	raw_repo, err := git.NewRepository(op.path)
	if err != nil {
		fmt.Println("couldn't open Git repository")
		os.Exit(1)
	}
	gitBin, err := findGitBin()
	if err != nil {
		fmt.Println("Couldn't find Git execute program")
		os.Exit(1)
	}

	var repo = &Repository{*raw_repo, op.path, gitBin}

	defer repo.Close()
	// scan repo
	var historySize HistoryRecord
	if op.scan {
		historySize, err = ScanRepository(*repo, op)
		if err != nil {
			fmt.Println("scanning repository error:\n *", err)
			os.Exit(1)
		}
		if op.verbose {
			for idx, b := range historySize.bigblob {
				name, _ := repo.GetBlobName(b.oid.String())
				fmt.Printf("[%d]: %s %d %s\n", idx, b.oid.String(), b.objectSize, name)

			}
		}
	}
}
