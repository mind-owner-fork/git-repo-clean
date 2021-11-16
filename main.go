package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/github/git-sizer/git"
)

func InitContext(args []string) *Repository {
	var op = Options{}
	if err := op.ParseOptions(args); err != nil {
		fmt.Println("Parse Option error")
		os.Exit(1)
	}
	if len(args) == 0 {
		op.interact = true
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

	version, err := GitVersion(gitBin, op.path)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// Git version should >= 2.24.0
	if GitVersionConvert(version) < 2240 {
		PrintRed("Sorry, this tool requires Git version at least 2.24.0")
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
		fmt.Println("Couldn't support running in bare repository")
		os.Exit(1)
	}
	if shallow, _ := IsShallow(gitBin, op.path); shallow {
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
			fmt.Println("scanning repository error:\n *", err)
			os.Exit(1)
		}

		if repo.opts.verbose && len(bloblist) != 0 {
			repo.ShowScanResult(bloblist)
		}

		if len(bloblist) == 0 {
			PrintRed("根据你所选择的筛选条件，没有扫描到任何文件，请调整筛选条件再试一次")
			os.Exit(1)
		}

		if repo.opts.interact {
			first_target = MultiSelectCmd(bloblist)
			if len(bloblist) != 0 && len(first_target) == 0 {
				PrintRed("您没有选择任何文件，请至少选择一个文件")
				os.Exit(1)
			}
			var ok = false
			ok, final_target = Confirm(first_target)
			if !ok {
				PrintRed("操作已中止，请重新确认文件后再次尝试")
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
	PrintGreen("本地仓库清理完成！")
	fmt.Print(fmt.Sprintf("\033[32m%s\033[0m", "当前仓库大小："))
	PrintGreen(GetDatabaseSize(repo.gitBin, repo.path))
	var pushed bool
	if AskForUpdate() {
		PrintPlain("将会执行如下两条命令，远端的的提交将会被覆盖:")
		PrintGreen("git push origin --all --force")
		PrintGreen("git push origin --tags --force")
		err := repo.PushRepo(repo.gitBin, repo.path)
		if err == nil {
			pushed = true
		}
	}
	PrintPlain("由于本地仓库的历史已经被修改，如果没有新的提交，建议先完成如下工作：\n")
	if pushed {
		PrintGreen("1. (已完成！)更新远程仓库。将本地清理后的仓库推送到远程仓库：")
		PrintGreen("    git push origin --all --force")
		PrintGreen("    git push origin --tags --force")
		fmt.Println()
	} else {
		PrintRed("1. (待完成)更新远程仓库。将本地清理后的仓库推送到远程仓库：")
		PrintRed("    git push origin --all --force")
		PrintRed("    git push origin --tags --force")
		fmt.Println()
	}
	PrintRed("2. (待完成)清理远程仓库。提交成功后，请前往你对应的仓库管理页面，执行GC操作")
	url := GetGiteeGCWeb(repo.gitBin, repo.path)
	if url != "" {
		url_fmter1 := fmt.Sprintf(FORMAT_YELLOW, url)
		fmter1 := fmt.Sprintf("如果是 Gitee 仓库，且有管理权限，请点击链接: %s", url_fmter1)
		PrintRed(strings.TrimSuffix(fmter1, "\n"))
	}
	fmt.Println()
	PrintRed("3. (待完成)处理关联仓库。处理同一个远程仓库下clone的其它仓库，确保不会将同样的文件再次提交到远程仓库")
	url_fmter2 := fmt.Sprintf(FORMAT_YELLOW, "https://gitee.com/oschina/git-repo-clean/blob/main/docs/repo-update.md")
	fmter2 := fmt.Sprintf("详细文档请参阅: %s", url_fmter2)
	PrintRed(fmter2)
	PrintPlain("完成以上三步后，恭喜你，所有的清理工作已经完成！")
	PrintPlain("如果有大文件的存储需求，请使用Git-LFS功能，避免仓库体积再次膨胀")
	url_fmter3 := fmt.Sprintf(FORMAT_YELLOW, "https://gitee.com/help/articles/4235")
	fmter3 := fmt.Sprintf("Gite LFS 的使用请参阅：%s", url_fmter3)
	PrintPlain(strings.TrimSuffix(fmter3, "\n"))
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
