package main

import (
	"fmt"
	"io"
	"os"

	"github.com/github/git-sizer/git"
)

func InitContext(args []string) *Repository {
	var op = Options{}
	if err := op.ParseOptions(args); err != nil {
		fmt.Println("Parse Option error")
		os.Exit(1)
	}

	repo, err := git.NewRepository(op.path)
	if err != nil {
		fmt.Println("couldn't open Git repository")
		os.Exit(1)
	}
	defer repo.Close()

	gitBin, err := findGitBin()
	if err != nil {
		fmt.Println("Couldn't find Git execute program")
		os.Exit(1)
	}
	gitDir, err := GitDir(gitBin, op.path)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return &Repository{
		*repo,
		op.path,
		gitBin,
		gitDir,
		op,
	}
}

func NewFilter(args []string) (*RepoFilter, error) {

	var repo = InitContext(args)
	filtered := make(map[git.OID]string)
	if repo.opts.scan {
		blobsize, err := ScanRepository(*repo)
		if err != nil {
			fmt.Println("scanning repository error:\n *", err)
			os.Exit(1)
		}
		if repo.opts.verbose {
			for idx, b := range blobsize.bigblob {
				name, _ := repo.GetBlobName(b.oid.String())
				filtered[b.oid] = name
				fmt.Printf("[%d]: %s %d %s\n", idx, b.oid.String(), b.objectSize, name)
			}
		}
	}

	Preader, Pwriter := io.Pipe()

	return &RepoFilter{
		repo:    repo,
		input:   *Pwriter,
		output:  *Preader,
		targets: filtered}, nil
}

func main() {
	filter, err := NewFilter(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "init New RepoFilter error")
		os.Exit(1)
	}
	filter.Parser()
}
