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
		op.PreCmd()
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

func NewFilter(args []string) (*RepoFilter, error) {

	var repo = InitContext(args)
	var first_target []string
	var final_target []string

	// when run git-clean-repo -i, its means run scan too
	if repo.opts.interact {
		repo.opts.scan = true
	}
	if repo.opts.scan {
		bloblist, err := ScanRepository(*repo)
		if err != nil {
			fmt.Println("scanning repository error:\n *", err)
			os.Exit(1)
		}

		if repo.opts.verbose {
			fmt.Println("根据选择扫描出的详细信息，分别为：文件ID，文件大小，文件名")
			fmt.Println("同一个文件，因为版本不同，ID号不同，因此可能有多个同名文件")
			for _, item := range bloblist {
				fmt.Printf("%s  %d 字节  %s\n", item.oid, item.objectSize, item.objectName)
			}
		}

		if repo.opts.interact {
			first_target = PostCmd(bloblist)
		} else {
			for _, item := range bloblist {
				final_target = append(final_target, item.oid)
			}
		}
	}

	if len(first_target) == 0 && len(final_target) == 0 && !(repo.opts.help || repo.opts.version || len(args) == 0) {
		fmt.Println("根据你所选条件，没有匹配到任何文件，请调整筛选条件再试一试")
		os.Exit(1)
	}

	if repo.opts.interact {
		final_target = DoubleCheckCmd(first_target)
	}

	Preader, Pwriter := io.Pipe()

	return &RepoFilter{
		repo:    repo,
		input:   *Pwriter,
		output:  *Preader,
		targets: final_target}, nil
}

func main() {
	filter, err := NewFilter(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "init New RepoFilter error")
		os.Exit(1)
	}
	filter.Parser()

	filter.repo.CleanUp()
}
