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
	if op.interact {
		op.Cmd()
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

	if bare, err := IsBare(gitBin, op.path); err != nil || bare {
		fmt.Println("Couldn't support running in bare repository")
		os.Exit(1)
	}
	if shallow, err := IsShallow(gitBin, op.path); err != nil || shallow {
		fmt.Println("Couldn't support running in shallow repository")
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

type SelectedBlob struct {
	idx      int
	oid      git.OID
	filesize uint32
	filename string
}
type BlobList []SelectedBlob

func NewFilter(args []string) (*RepoFilter, error) {

	var repo = InitContext(args)
	filtered := make(map[git.OID]string)
	if repo.opts.scan {
		bloblist, err := ScanRepository(*repo)
		if err != nil {
			fmt.Println("scanning repository error:\n *", err)
			os.Exit(1)
		}
		for _, b := range bloblist {
			name, _ := repo.GetBlobName(b.oid.String())
			filtered[b.oid] = name
		}
		if repo.opts.verbose {
			for idx, name := range filtered {
				fmt.Println(idx)
				fmt.Println(name)
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
