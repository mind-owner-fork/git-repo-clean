package main

import (
	"fmt"
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

	if fresh, err := IsFresh(gitBin, op.path); err == nil && !fresh && !op.force {
		PrintYellow("不支持在不是刚克隆的仓库中进行重写操作，请确保已经将仓库进行备份")
		PrintYellow("备份请参考： git clone --no-local <原始仓库路径> <备份仓库路径>")
		PrintYellow("如果确定继续进行操作，也可以使用'--force'参数强制执行")
		os.Exit(1)
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
		repo.opts.SurveyCmd()
	}
	if repo.opts.scan {
		bloblist, err := ScanRepository(*repo)
		if err != nil {
			fmt.Println("scanning repository error:\n *", err)
			os.Exit(1)
		}

		if repo.opts.verbose && len(bloblist) != 0 {
			ShowScanResult(bloblist)
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

func Prompt() {
	PrintGreen("本地仓库清理完成！")
	PrintYellow("由于本地仓库的历史已经被修改，如果没有新的提交，建议先完成如下工作：")
	PrintYellow("1. 更新远程仓库。将本地清理后的仓库推送到远程仓库：")
	PrintYellow("    git push origin --all --force")
	PrintYellow("    git push origin --tags --force")
	PrintYellow("2. 清理远程仓库。提交成功后，请前往你对应的仓库管理页面，执行GC操作")
	PrintYellow("(如果是 Gitee 仓库，请查阅GC帮助文档: https://gitee.com/help/articles/4173)")
	PrintYellow("3. 处理关联仓库。处理具有同一个远程仓库的其他副本仓库，确保不会将同样的文件再次提交到远程仓库")
	PrintYellow("请参阅详细文档 https://gitee.com/oschina/git-repo-clean/blob/main/docs/repo-update.md")
	PrintPlain("完成以上三步后，恭喜你，所有的清理工作已经完成！")
	PrintPlain("如果有大文件的存储需求，请使用Git-LFS功能，避免仓库体积再次膨胀")
	PrintPlain("(Gitee LFS 的使用请参阅：https://gitee.com/help/articles/4235)")
}

func main() {
	filter, err := NewFilter(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "init repo filter error")
		os.Exit(1)
	}
	filter.Parser()

	filter.repo.CleanUp()
	Prompt()
}
