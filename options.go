package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/pflag"
)

const Usage = `usage: git clean-repo [options]

git clean-repo have two stage, one is to scan your Git
repository, like --scan and some other options followed
by it, the next stage is to perform some actions in repo,
like delete action(--delete) or interaction with the user
(--interactive)

  --verbose, -v		show process information
  --version, -V		show git-clean-repo version number
  --help, -h		show usage information
  --scan, -s		scan the Git repository objects
  --range, -r		specify repository scan range
			it can be full|blobs|commits|trees|refs|tags
  --limit, -l		set the file size limitation, like: '--limit=10m'
  --number, -n		set the number of results to show
  --type, -t		set the file type to filter from Git repository
  --delete, -d		delete the file from Git repository history
  --interactive, -i 	enable interactive operation

Git Large File Storage(LFS) replaces large files such as
multi-media file, executable file with text pointers inside Git,
while storing the file contents on a remote server like Gitee.
So please make sure you have installed git-lfs in your local first.
To download see: https://github.com/git-lfs/git-lfs/releases
If you let LFS tool to handle your large file in your repo,
git-clean-repo will config git-lfs to track the file you scanned before.

  --lfs, -L		use LFS server to storage local large file
			must followed by --add option
  --add <file>, -a	let LFS track specified file, have the same effect with
			"git lfs migrate import --include=file --everything"

`

type Options struct {
	verbose  bool
	version  bool
	help     bool
	scan     bool
	ranges   string
	limit    string
	number   uint32
	types    string
	delete   bool
	interact bool
	lfs      bool
	add      string
}

func (op *Options) init(args []string) error {

	flags := pflag.NewFlagSet("git-clean-repo", pflag.ContinueOnError)

	// flags.Usage = func() {
	// 	fmt.Print(Usage)
	// }
	flags.BoolVarP(&op.verbose, "verbose", "v", false, "show process information")
	flags.BoolVarP(&op.version, "version", "V", false, "show git-clean-repo version number")
	flags.BoolVarP(&op.help, "help", "h", false, "show usage information")
	// default is to scan repo
	flags.BoolVarP(&op.scan, "scan", "s", true, "scan the Git repository objects")
	// default to scan repo's blob objects
	flags.StringVarP(&op.ranges, "range", "r", "blobs", "scan the Git repository objects")
	// default file threshold is 1M
	flags.StringVarP(&op.limit, "limit", "l", "1m", "set the file size limitation")
	// default to show top 3 largest file
	flags.Uint32VarP(&op.number, "number", "n", 3, "set the number of results to show")
	// default is null, which means all type
	flags.StringVarP(&op.types, "type", "t", "", "set the file type to filter from Git repository")
	flags.BoolVarP(&op.delete, "delete", "d", false, "delete the file from Git repository history")
	flags.BoolVarP(&op.interact, "interative", "i", true, "enable interactive operation")
	flags.BoolVarP(&op.lfs, "lfs", "L", false, "use LFS server to storage local large file, must followed by --add option")
	flags.StringVarP(&op.add, "add", "a", "", "let LFS track specified file, will execute 'git lfs migrate import --include=file --everything'")

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
	fmt.Fprintf(os.Stderr, Usage)
}

func (op *Options) OptionInit(args []string) error {
	if err := op.init(args); err != nil {
		fmt.Printf("%s\n", err)
	}
	if op.help {
		usage()
	}
	if op.version {
		fmt.Printf("Build version: %s\n", BuildVersion)
	}
	return nil
}
