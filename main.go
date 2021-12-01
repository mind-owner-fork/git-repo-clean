package main

import (
	"fmt"
	"os"

	"github.com/github/git-sizer/git"
)

func InitContext(args []string) *Repository {
	var op = Options{}
	if err := op.ParseOptions(args); err != nil {
		PrintLocalWithRedln("Parse Option error")
		os.Exit(1)
	}
	if len(args) == 0 {
		op.interact = true
	}
	repo, err := git.NewRepository(op.path)
	if err != nil {
		PrintLocalWithRedln("Couldn't open Git repository")
		os.Exit(1)
	}
	defer repo.Close()

	gitBin, err := findGitBin()
	if err != nil {
		PrintLocalWithRedln("Couldn't find Git execute program")
		os.Exit(1)
	}

	version, err := GitVersion(gitBin, op.path)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// Git version should >= 2.24.0
	if GitVersionConvert(version) < 2240 {
		PrintLocalWithRedln("Sorry, this tool requires Git version at least 2.24.0")
		os.Exit(1)
	}

	cur, err := GetCurrentBranch(gitBin, op.path)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if op.branch == "" {
		op.branch = cur
	} else if op.branch == "all" {
		op.branch = "--all"
	}

	if bare, _ := IsBare(gitBin, op.path); bare {
		PrintLocalWithRedln("Couldn't support running in bare repository")
		os.Exit(1)
	}
	if shallow, _ := IsShallow(gitBin, op.path); shallow {
		PrintLocalWithRedln("Couldn't support running in shallow repository")
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

	// when run git-repo-clean -i, its means run scan too
	if repo.opts.interact {
		repo.opts.scan = true
		repo.opts.delete = true
		repo.opts.verbose = true

		if err := repo.opts.SurveyCmd(); err != nil {
			os.Exit(1)
		}
	}
	if repo.opts.scan {
		bloblist, err := ScanRepository(*repo)
		if err != nil {
			ft := LocalPrinter().Sprintf("scanning repository error: %s", err)
			PrintRedln(ft)
			os.Exit(1)
		}
		if len(bloblist) != 0 {
			repo.ShowScanResult(bloblist)
		}

		if len(bloblist) == 0 {
			PrintLocalWithRedln("no files were scanned")
			os.Exit(1)
		}

		if repo.opts.interact {
			first_target = MultiSelectCmd(bloblist)
			if len(bloblist) != 0 && len(first_target) == 0 {
				PrintLocalWithRedln("no files were selected")
				os.Exit(1)
			}
			var ok = false
			ok, final_target = Confirm(first_target)
			if !ok {
				PrintLocalWithRedln("operation aborted")
				os.Exit(1)
			}
		} else {
			for _, item := range bloblist {
				final_target = append(final_target, item.oid)
			}
		}
	}

	if !repo.opts.delete {
		os.Exit(1)
	}

	return &RepoFilter{
		repo:    repo,
		targets: final_target}, nil
}

func Prompt(repo Repository) {
	PrintLocalWithGreenln("cleaning completed")
	PrintLocalWithPlain("current repository size")
	PrintLocalWithGreenln(GetDatabaseSize(repo.gitBin, repo.path))
	var pushed bool
	if AskForUpdate() {
		PrintLocalWithPlainln("execute force push")
		PrintLocalWithYellowln("git push origin --all --force")
		PrintLocalWithYellowln("git push origin --tags --force")
		err := repo.PushRepo(repo.gitBin, repo.path)
		if err == nil {
			pushed = true
		}
	}
	PrintLocalWithPlainln("suggest operations header")
	if pushed {
		PrintLocalWithGreenln("1. (Done!)")
		PrintLocalWithGreenln("    git push origin --all --force")
		PrintLocalWithGreenln("    git push origin --tags --force")
		fmt.Println()
	} else {
		PrintLocalWithRedln("1. (Undo)")
		PrintLocalWithRedln("    git push origin --all --force")
		PrintLocalWithRedln("    git push origin --tags --force")
		fmt.Println()
	}
	PrintLocalWithRedln("2. (Undo)")
	url := GetGiteeGCWeb(repo.gitBin, repo.path)
	if url != "" {
		PrintLocalWithRed("gitee GC page link")
		url_fmter1 := fmt.Sprintf(FORMAT_YELLOW, url)
		fmt.Println(url_fmter1)
	}
	fmt.Println()
	PrintLocalWithRedln("3. (Undo)")
	PrintLocalWithRed("for detailed documentation, see")
	url_fmter2 := fmt.Sprintf(FORMAT_YELLOW, "https://gitee.com/oschina/git-repo-clean/blob/main/docs/repo-update.md")
	fmt.Println(url_fmter2)
	fmt.Println()
	PrintLocalWithPlainln("suggest operations done")
	PrintLocalWithPlainln("introduce GIT LFS")
	PrintLocalWithPlain("for the use of Gitee LFS, see")
	url_fmter3 := fmt.Sprintf(FORMAT_YELLOW, "https://gitee.com/help/articles/4235")
	fmt.Println(url_fmter3)
}

func main() {
	filter, err := NewFilter(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "init repo filter error")
		os.Exit(1)
	}
	// repo backup
	if AskForBackUp() {
		filter.repo.BackUp(filter.repo.gitBin, filter.repo.path)
	}
	// filter data
	filter.Parser()

	filter.repo.CleanUp()
	Prompt(*filter.repo)
}
