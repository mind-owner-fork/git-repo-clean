package main

import (
	"errors"
	"os"

	"github.com/spf13/pflag"
)

var BuildVersion string

const Usage = `usage: git repo-clean [options]

********************* Important! **********************
*** The rewrite command is a destructive operation ****
*** Please backup your repo before do any operation ***
*******************************************************

git repo-clean is a tool to scan Git repository metadata,
and filter out specify file by its type, size, and delete
those files completely from the repo, and will rewrite the
commit history relatived to those files.

Options:
  -v, --verbose		show process information
  -V, --version		show git-repo-clean version number
  -h, --help		show usage information
  -p, --path		Git repository path, default is '.'
  -s, --scan		scan the Git repository objects, default to scan all branches
  -f, --file		provie file path directly to delete, incompatible with --scan
  -b, --branch		set the branch where files need to be deleted , default all branches
  -l, --limit		set the file size limitation, like: '--limit=10m'
  -n, --number		set the number of results to show
  -t, --type		set the file name suffix to filter from Git repository
  -i, --interactive 	enable interactive operation
  -d, --delete		execute file cleanup and history rewrite process

These options can provide users with two ways of using: 
interactive way, command line way.

Interactive way:
  Execute "git repo clean" or "git repo clean -i" to enter the interactive interface.
  The program interacts with the user through question and answer, making the whole process
  of file filtering, backup, deletion and history rewrite easier for the user.
  
Command-Line way:
  You can apply various options on the command line to realize functions, such as:

  To scan only files with file type tar.gz and its size greater than 1G in the repo: 
    git repo-clean --scan --limit=1G --type=tar.gz

  When you need to delete specified files, add --delete option and execute:
    git repo-clean --scan --limit=1G --type=tar.gz --delete

  If the same file exists in multiple branches, or the same file still exists after
  the previous deletion, you can use the --branch option to delete it from all branches:
    git repo-clean --scan --limit=1G --type=tar.gz --delete --branch=all

  If there are too many scanning results according to the specified conditions, 
  you can limit the number of results by --number option:
    git repo-clean --scan --limit=1G --type=tar.gz --delete --number=3

  If you want to delete a known file, there is no need to scan the whole repo,
  just use the '--file' option:
    git repo-clean --file file1 --file file2 --delete

  Or, if you want to delete all files under dir/ :
    git repo-clean --file dir/ --delete

`
const Usage_ZH = `用法: git repo-clean [选项]

********************* 重要! *****************
*** 该历史重写过程是不可逆的破坏性的操作 ***
*** 请在做任何操作之前先备份您的仓库数据 ***
*********************************************

git repo-clean 是一款扫描Git仓库元数据，然后根据指定的文件类型
以及大小来过滤出文件，并且从仓库中完全删除掉这些指定文件的工具
，它将重写跟删除的文件相关的提交以及之后的提交的历史。

选项：
  -v, --verbose		显示处理的详细过程
  -V, --version		显示 git-repo-clean 版本号
  -h, --help		显示使用信息
  -p, --path		指定Git仓库的路径, 默认是当前目录，即'.'
  -s, --scan		扫描Git仓库数据，默认是扫描所有分支中的数据
  -f, --file		直接指定仓库中的文件或目录，与'--scan'不兼容
  -b, --branch		设置需要删除文件的分支, 默认是从所有分支中删除文件
  -l, --limit		设置扫描文件阈值, 比如: '--limit=10m'
  -n, --number		设置显示扫描结果的数量
  -t, --type		设置扫描文件后缀名，即文件类型
  -i, --interactive 	开启交互式操作
  -d, --delete		执行文件删除和历史重写过程


这些选项主要可以给用户提供两种使用方法：交互式、命令行式

交互式用法:
  直接执行git repo-clean或git repo-clean -i进入交互式界面
  程序与用户通过问答的方式进行交互，使得用户在处理文件筛选、
  备份、删除、历史重写的整个过程变得更加简单。

命令行式用法：
  用户可以在命令行中通过指定各种选项的参数，来实现功能，例如：

  为了只扫描仓库中文件类型为tar.gz，且大小超过1G的文件，执行：
    git repo-clean --scan --limit=1G --type=tar.gz

  当需要删除指定文件时，需要加上--delete选项，执行：
    git repo-clean --scan --limit=1G --type=tar.gz --delete

  如果相同文件存在多个分支中，或者发现前一次删除之后，相同的
  文件仍然存在，则可以使用--branch选项，从所有分支删除，执行：
    git repo-clean --scan --limit=1G --type=tar.gz --delete --branch=all

  如果根据指定条件，扫描结果过多，可以通过--number限制结果数量，执行：
    git repo-clean --scan --limit=1G --type=tar.gz --delete --number=3

  如果你想删除某个已知的文件，则不必扫描仓库，使用'--file'选项，直接指定文件：
	  git repo-clean --file file1 --file file2 --delete

  或者，你想一次性删除某个目录下所有的文件，以及相关提交记录：
	  git repo-clean --file dir/ --delete

`

type Options struct {
	verbose  bool
	version  bool
	help     bool
	path     string
	scan     bool
	file     []string
	delete   bool
	branch   string
	limit    string
	number   uint32
	types    string
	interact bool
	// lfs      bool
}

func (op *Options) init(args []string) error {

	flags := pflag.NewFlagSet("git-repo-clean", pflag.ContinueOnError)

	flags.BoolVarP(&op.verbose, "verbose", "v", false, "show process information")
	flags.BoolVarP(&op.version, "version", "V", false, "show git-repo-clean version number")
	flags.BoolVarP(&op.help, "help", "h", false, "show usage information")

	flags.StringVarP(&op.path, "path", "p", ".", "Git repository path, default is '.'")
	// default is to scan repo
	flags.BoolVarP(&op.scan, "scan", "s", false, "scan the Git repository objects")
	// specify the target files to delete
	flags.StringArrayVarP(&op.file, "file", "f", nil, "specify the target files to delete")
	// since the deleting process is not very slow, default is all branch
	flags.StringVarP(&op.branch, "branch", "b", "all", "set the branch to scan")
	// default file size threshold is zero byte
	flags.StringVarP(&op.limit, "limit", "l", "0b", "set the file size limitation")
	// default to show top 3 largest files
	flags.Uint32VarP(&op.number, "number", "n", 3, "set the number of results to show")
	// default is null, which means all types
	flags.StringVarP(&op.types, "type", "t", "", "set the file type to filter from Git repository")
	// interactive with user end
	flags.BoolVarP(&op.interact, "interative", "i", false, "enable interactive operation")
	// perform delete files action
	flags.BoolVarP(&op.delete, "delete", "d", false, "execute file cleanup and history rewrite process")

	err := flags.Parse(args)
	if err != nil {
		if err == pflag.ErrHelp {
			return nil
		}
		return err
	}
	if len(flags.Args()) != 0 {
		return errors.New("excess arguments")
	}
	return nil
}

func usage() {
	LocalFprintf(os.Stderr, "help info")
}

func (op *Options) ParseOptions(args []string) error {
	if err := op.init(args); err != nil {
		ft := LocalPrinter().Sprintf("option format error: %s", err)
		PrintRedln(ft)
		os.Exit(1)
	}
	if op.help {
		usage()
		os.Exit(1)
	}
	if op.version {
		ft := LocalPrinter().Sprintf("build version: %s", BuildVersion)
		PrintPlainln(ft)
		os.Exit(1)
	}
	if len(args) == 1 && op.SingleOpts() && !op.interact {
		PrintLocalWithRedln("single parameter is invalid")
		os.Exit(1)
	}
	return nil
}

func (op *Options) SingleOpts() bool {
	if op.verbose || op.scan || op.delete || op.path != "" {
		return true
	} else {
		return false
	}
}
