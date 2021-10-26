package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/pflag"
)

var BuildVersion string

const Usage = `usage: git clean-repo [options]

********************* Important! **********************
*** The rewrite command is a destructive operation ****
*** Please backup your repo before do any operation ***
*******************************************************

git clean-repo have two stage, one is to scan your Git
repository, by using --scan and some other options followed
by it, the next stage is to perform some operations in repo,
like delete operation(--delete) or interaction with the user
(--interactive)

  -v, --verbose		show process information
  -V, --version		show git-clean-repo version number
  -h, --help		show usage information
  -p, --path		Git repository path, default is '.'
  -s, --scan		scan the Git repository objects
  -l, --limit		set the file size limitation, like: '--limit=10m'
  -n, --number		set the number of results to show
  -t, --type		set the file type to filter from Git repository
  -i, --interactive 	enable interactive operation

Git Large File Storage(LFS) replaces large files such as
multi-media file, executable file with text pointers inside Git,
while storing the file contents on a remote server like Gitee.
So please make sure you have installed git-lfs in your local first.
To download it, clik: https://github.com/git-lfs/git-lfs/releases
If you let LFS tool to handle your large file in your repo,
git-clean-repo will config it to track the file you scanned before.

  -L, --lfs		use LFS server to storage local large file
			must followed by --add option

`

type Options struct {
	verbose  bool
	version  bool
	help     bool
	path     string
	scan     bool
	ranges   string
	limit    string
	number   uint32
	types    string
	interact bool
	lfs      bool
}

func (op *Options) init(args []string) error {

	flags := pflag.NewFlagSet("git-clean-repo", pflag.ContinueOnError)

	flags.BoolVarP(&op.verbose, "verbose", "v", false, "show process information")
	flags.BoolVarP(&op.version, "version", "V", false, "show git-clean-repo version number")
	flags.BoolVarP(&op.help, "help", "h", false, "show usage information")

	flags.StringVarP(&op.path, "path", "p", ".", "Git repository path, default is '.'")
	// default is to scan repo
	flags.BoolVarP(&op.scan, "scan", "s", false, "scan the Git repository objects")
	// default file threshold is 1M
	flags.StringVarP(&op.limit, "limit", "l", "1m", "set the file size limitation")
	// default to show top 3 largest file
	flags.Uint32VarP(&op.number, "number", "n", 3, "set the number of results to show")
	// default is null, which means all type
	flags.StringVarP(&op.types, "type", "t", "", "set the file type to filter from Git repository")
	flags.BoolVarP(&op.interact, "interative", "i", false, "enable interactive operation")
	flags.BoolVarP(&op.lfs, "lfs", "L", false, "use LFS server to storage local large file, must followed by --add option")

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

func (op *Options) ParseOptions(args []string) error {
	if err := op.init(args); err != nil {
		fmt.Printf("option format error: %s\n", err)
		os.Exit(1)
	}
	if op.help || len(args) == 0 {
		usage()
	}
	if op.version {
		fmt.Printf("Build version: %s\n", BuildVersion)
	}
	return nil
}
