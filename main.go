package main

import (
	"fmt"
	"os"
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

	r, err := NewRepository(op.path)
	if err != nil {
		ft := LocalPrinter().Sprintf("Couldn't open Git repository: %s", err)
		PrintRedln(ft)
		os.Exit(1)
	}

	// set default branch to all is to keep deleting process consistent with scanning process
	// user end pass '--branch=all', but git-fast-export takes '--all'
	if op.branch == "all" {
		op.branch = "--all"
	}

	return &Repository{
		op.path,
		r.gitBin,
		r.gitDir,
		op,
	}
}

func NewFilter(args []string) (*RepoFilter, error) {

	var repo = InitContext(args)
	err := GetBlobSize(*repo)
	if err != nil {
		ft := LocalPrinter().Sprintf("run getblobsize error: %s", err)
		PrintRedln(ft)
	}
	var first_target []string
	var scanned_targets []string
	var file_paths []string
	// when run git-repo-clean -i, its means run scan too
	if repo.opts.interact {
		repo.opts.scan = true
		repo.opts.delete = true
		repo.opts.verbose = true

		if err := repo.opts.SurveyCmd(); err != nil {
			os.Exit(1)
		}
	}

	PrintLocalWithPlain("current repository size")
	PrintLocalWithYellowln(GetDatabaseSize(repo.gitBin, repo.path))

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
			ok, scanned_targets = Confirm(first_target)
			if !ok {
				PrintLocalWithRedln("operation aborted")
				os.Exit(1)
			}
		} else {
			for _, item := range bloblist {
				scanned_targets = append(scanned_targets, item.oid)
			}
		}
	} else {
		if repo.opts.file != nil {
			file_paths = repo.opts.file
		}
	}

	if !repo.opts.delete {
		os.Exit(1)
	}

	return &RepoFilter{
		repo:      repo,
		scanned:   scanned_targets,
		filepaths: file_paths}, nil
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
		PrintYellowln(url)
	}
	fmt.Println()
	PrintLocalWithRedln("3. (Undo)")
	PrintLocalWithRed("for detailed documentation, see")
	PrintYellowln("https://gitee.com/oschina/git-repo-clean/blob/main/docs/repo-update.md")
	fmt.Println()
	PrintLocalWithPlainln("suggest operations done")
	PrintLocalWithPlainln("introduce GIT LFS")
	PrintLocalWithPlain("for the use of Gitee LFS, see")
	PrintYellowln("https://gitee.com/help/articles/4235")
}

func main() {
	filter, err := NewFilter(os.Args[1:])
	if err != nil {
		LocalFprintf(os.Stderr, "init repo filter error")
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
