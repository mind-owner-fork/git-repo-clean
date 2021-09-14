package main

import (
	"fmt"
	"io"
	"os"

	"github.com/github/git-sizer/git"
	"github.com/github/git-sizer/sizes"
)

var BuildVersion string

func main() {
	// parse argument
	var op = Options{}
	if err := op.OptionInit(os.Args[1:]); err != nil {
		fmt.Println("init error")
	}

	// create repo
	repo, err := git.NewRepository(".")
	if err != nil {
		fmt.Println("couldn't open Git repository")

	}
	defer repo.Close()

	// scan repo
	var nameStyle sizes.NameStyle = sizes.NameStyleFull

	var size uint
	if op.ranges == "full" {
		size = 0
	} else {
		size = 1
	}
	var threshold sizes.Threshold = sizes.Threshold(size)
	var historySize HistoryRecord
	if op.scan {
		historySize, err = ScanRepository(*repo, op)
		if err != nil {
			fmt.Println("scanning repository error:\n *", err)
		}
		for idx, b := range historySize.bigblob {
			fmt.Printf("blob[%d]: %s -> %d\n", idx, b.oid.String(), b.objectSize)
		}
		// report
		io.WriteString(os.Stdout, historySize.TableString(threshold, nameStyle))
	}
}
