package main

import (
	"fmt"
	"os"
	"strings"
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
		r.bare,
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
			ft := LocalPrinter().Sprintf("ask question module fail: %s", err)
			PrintRedln(ft)
			os.Exit(1)
		}
	}
	if repo.opts.lfs {
		repo.opts.scan = true
	}

	PrintLocalWithPlain("current repository size")
	PrintLocalWithYellowln(repo.GetDatabaseSize())
	if lfs := repo.GetLFSObjSize(); len(lfs) > 0 {
		PrintLocalWithPlain("including LFS objects size")
		PrintLocalWithYellowln(lfs)
	}

	if repo.opts.scan {
		bloblist, err := ScanRepository(*repo)
		if err != nil {
			ft := LocalPrinter().Sprintf("scanning repository error: %s", err)
			PrintRedln(ft)
			os.Exit(1)
		}
		if len(bloblist) == 0 {
			PrintLocalWithRedln("no files were scanned")
			os.Exit(1)
		} else {
			repo.ShowScanResult(bloblist)
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
		//  record target file's name
		for _, item := range bloblist {
			for _, target := range scanned_targets {
				if item.oid == target {
					// cut their prefix
					list := strings.Split(item.objectName, "/")
					Files_changed.Add(list[len(list)-1])
				}
			}
		}

	} else {
		if repo.opts.file != nil {
			file_paths = repo.opts.file
			repo.opts.limit = ""
		}
		if repo.opts.limit != "" {
			repo.opts.number = ^uint32(0) // UINT_MAX
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

func LFSPrompt(repo Repository) {
	FilesChanged()
	PrintLocalWithPlainln("before you push to remote, you have to do something below:")
	PrintLocalWithYellowln("1. install git-lfs")
	PrintLocalWithYellowln("2. run command: git lfs install")
	PrintLocalWithYellowln("3. edit .gitattributes file")
	PrintLocalWithYellowln("4. commit your .gitattributes file.")
}

func Prompt(repo Repository) {
	PrintLocalWithGreenln("cleaning completed")
	PrintLocalWithPlain("current repository size")
	PrintLocalWithYellowln(repo.GetDatabaseSize())
	if lfs := repo.GetLFSObjSize(); len(lfs) > 0 {
		PrintLocalWithPlain("including LFS objects size")
		PrintLocalWithYellowln(lfs)
	}
	if repo.opts.lfs {
		LFSPrompt(repo)
	}
	var pushed bool
	if !repo.opts.lfs {
		if AskForUpdate() {
			PrintLocalWithPlainln("execute force push")
			PrintLocalWithYellowln("git push origin --all --force")
			PrintLocalWithYellowln("git push origin --tags --force")
			err := repo.PushRepo(repo.gitBin, repo.path)
			if err == nil {
				pushed = true
			}
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
	// ask for lfs migrate
	if AskForMigrateToLFS() {
		// can't run lfs-migrate in bare repo
		if filter.repo.bare {
			PrintLocalWithYellowln("bare repo error")
			os.Exit(1)
		}
		filter.repo.opts.lfs = true
	} else {
		filter.repo.opts.lfs = false
	}
	// filter data
	filter.Parser()

	filter.repo.CleanUp()
	Prompt(*filter.repo)
}
