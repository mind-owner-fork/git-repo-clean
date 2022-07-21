package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"path/filepath"
)

type Context struct {
	workDir string
	gitBin  string
	gitDir  string
	bare    bool
	opts    *Options
	scan_t  ScanType
}

type ScanType struct {
	filepath bool
	filesize bool
	filetype bool
}

type Repository struct {
	context  *Context
	filtered []string
}

type HistoryRecord struct {
	oid        string
	objectSize uint64
	objectName string
}

type BlobList []HistoryRecord

var Blob_size_list = make(map[string]string)

func InitContext(path string) (*Context, error) {
	// Find the `git` executable to be used:
	gitBin, err := findGitBin()
	if err != nil {
		return nil, fmt.Errorf(LocalPrinter().Sprintf(
			"couldn't find Git execute program: %s", err))
	}

	// check git version
	version, err := GitVersion(gitBin)
	if err != nil {
		return nil, err
	}
	// Git version should >= 2.24.0
	if GitVersionConvert(version) < 2240 {
		return nil, fmt.Errorf(LocalPrinter().Sprintf(
			"sorry, this tool requires Git version at least 2.24.0"))
	}

	// check is current repo is in bare repo
	var bare bool
	if b, err := IsBare(gitBin, path); b && err == nil {
		bare = true
		PrintLocalWithYellowln("bare repo warning")
	}

	// check if current repo has uncommited files
	if !bare {
		err = GetCurrentStatus(gitBin, path)
		if err != nil {
			PrintLocalWithRedln(LocalPrinter().Sprintf("%s", err))
			os.Exit(1)
		}
	}

	// check if current repo is in shallow repo
	if shallow, err := IsShallow(gitBin, path); shallow {
		return nil, err
	}

	gitdir, err := GitDir(gitBin, path)
	if err != nil {
		return nil, err
	}

	return &Context{
		workDir: path,   // worktree dir
		gitDir:  gitdir, // .git dir
		gitBin:  gitBin,
		bare:    bare,
		opts:    &op, // global
	}, nil
}

func NewRepository() *Repository {
	// init repo context
	ctx, err := InitContext(op.path)
	if err != nil {
		PrintLocalWithRedln(LocalPrinter().Sprintf("%s", err))
		os.Exit(1)
	}
	// important! get repo blob list
	err = GetBlobSize(ctx.gitBin, ctx.workDir)
	if err != nil {
		ft := LocalPrinter().Sprintf("run getblobsize error: %s", err)
		PrintRedln(ft)
	}
	// scan repo files
	scanedfiles, err := ScanFiles(ctx)
	if err != nil {
		LocalFprintf(os.Stderr, "init repo filter error")
		os.Exit(1)
	}

	return &Repository{
		context:  ctx,
		filtered: scanedfiles,
	}
}

func GetBlobName(gitbin, path, oid string) (string, error) {
	cmd := exec.Command(gitbin, "-C", path, "rev-list", "--objects", "--all")
	out, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		return "", err
	}
	blobname := ""
	buf := bufio.NewReader(out)
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		// drop LF
		line = line[:len(line)-1]

		if len(line) <= 41 {
			continue
		}
		if line[0:40] == oid {
			blobname = line[41:]
			break
		}
	}
	return blobname, nil
}

func parseBatchHeader(header string) (objectid, objecttype, objectsize string, err error) {
	// drop LF
	header = header[:len(header)-1]

	infos := strings.Split(header, " ")
	if infos[len(infos)-1] == "missing" {
		return "", "", "", errors.New("got missing object")
	}
	return infos[0], infos[1], infos[2], nil
}

// GetBlobSize to get repository blobs list
func GetBlobSize(gitbin, path string) error {
	cmd := exec.Command(gitbin, "-C", path, "cat-file", "--batch-all-objects",
		"--batch-check=%(objectname) %(objecttype) %(objectsize)")
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		return err
	}

	buf := bufio.NewReader(out)
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		objectid, objecttype, objectsize, err := parseBatchHeader(line)
		if err != nil {
			return err
		}
		if objecttype == "blob" {
			Blob_size_list[objectid] = objectsize
		}
	}
	return nil
}

func ScanRepository(context *Context) (BlobList, error) {
	var empty BlobList
	var blobs BlobList

	if context.opts.verbose {
		PrintLocalWithGreenln("start scanning")
	}

	for objectid, objectsize := range Blob_size_list {
		// set bitsize to 64, means max single blob size is 4 GiB
		size, _ := strconv.ParseUint(objectsize, 10, 64)
		if context.opts.lfs && !context.opts.interact {
			name, err := GetBlobName(context.gitBin, context.workDir, objectid)
			if err != nil {
				if err != io.EOF {
					return empty, fmt.Errorf(LocalPrinter().Sprintf(
						"run GetBlobName error: %s", err))
				}
			}
			if name == "" {
				continue
			}
			if len(context.opts.types) != 0 && context.opts.types != DefaultFileType {
				extent := filepath.Ext(name)
				if extent == "."+context.opts.types {
					limit, err := UnitConvert(context.opts.limit)
					if err != nil {
						return empty, fmt.Errorf(LocalPrinter().Sprintf(
							"convert uint error: %s", err))
					}
					if size < limit {
						continue
					}
					// append this record blob into slice
					blobs = append(blobs, HistoryRecord{objectid, size, name})
					// sort according by size
					sort.Slice(blobs, func(i, j int) bool {
						return blobs[i].objectSize > blobs[j].objectSize
					})
				}
			}
		} else {
			limit, err := UnitConvert(context.opts.limit)
			if err != nil {
				return empty, fmt.Errorf(LocalPrinter().Sprintf(
					"convert uint error: %s", err))
			}

			if size > limit {
				name, err := GetBlobName(context.gitBin, context.workDir, objectid)
				if err != nil {
					if err != io.EOF {
						return empty, fmt.Errorf(LocalPrinter().Sprintf(
							"run GetBlobName error: %s", err))
					}
				}
				if name == "" {
					continue
				}
				if len(context.opts.types) != 0 && context.opts.types != DefaultFileType {
					extent := filepath.Ext(name)
					if extent != "."+context.opts.types {
						// matched none, skip
						continue
					}
				}
				// append this record blob into slice
				blobs = append(blobs, HistoryRecord{objectid, size, name})
				// sort according by size
				sort.Slice(blobs, func(i, j int) bool {
					return blobs[i].objectSize > blobs[j].objectSize
				})
				// remain first [op.number] blobs
				if len(blobs) > int(context.opts.number) {
					blobs = blobs[:context.opts.number]
					// break
				}
			}
		}
	}
	return blobs, nil
}

func ScanMode(ctx *Context) (result []string) {
	var first_target []string

	bloblist, err := ScanRepository(ctx)
	if err != nil {
		ft := LocalPrinter().Sprintf("scanning repository error: %s", err)
		PrintRedln(ft)
		os.Exit(1)
	}
	if len(bloblist) == 0 {
		PrintLocalWithRedln("no files were scanned")
		os.Exit(1)
	} else {
		ShowScanResult(bloblist)
	}

	if ctx.opts.interact {
		first_target = MultiSelectCmd(bloblist)
		if len(bloblist) != 0 && len(first_target) == 0 {
			PrintLocalWithRedln("no files were selected")
			os.Exit(1)
		}
		var ok = false
		ok, result = Confirm(first_target)
		if !ok {
			PrintLocalWithRedln("operation aborted")
			os.Exit(1)
		}
	} else {
		for _, item := range bloblist {
			result = append(result, item.oid)
		}
	}
	//  record target file's name
	for _, item := range bloblist {
		for _, target := range result {
			if item.oid == target {
				Files_changed.Add(item.objectName)
			}
		}
	}
	return result
}

func NonScanMode(ctx *Context, file_limit string, file_type string, file_num uint32) {
	ctx.opts.limit = file_limit
	ctx.opts.types = file_type
	ctx.opts.number = file_num
}

func ScanFiles(ctx *Context) ([]string, error) {
	var scanned_targets []string
	// when run git-repo-clean -i, its means run scan too
	if ctx.opts.interact {
		ctx.opts.scan = true
		ctx.opts.delete = true
		ctx.opts.verbose = true
		ctx.opts.lfs = true

		if err := ctx.opts.SurveyCmd(); err != nil {
			ft := LocalPrinter().Sprintf("ask question module fail: %s", err)
			PrintRedln(ft)
			os.Exit(1)
		}
	}

	// set default branch to all is to keep deleting process consistent with scanning process
	// user end pass '--branch=all', but git-fast-export takes '--all'
	if op.branch == DefaultRepoBranch {
		op.branch = "--all"
	}

	if ctx.opts.lfs {
		limit, _ := UnitConvert(ctx.opts.limit)
		if limit < 200 {
			ctx.opts.limit = "200b" // to project LFS file
		}
		// can't run lfs-migrate in bare repo
		// git lfs track must be run in a work tree.
		if ctx.bare {
			PrintLocalWithYellowln("bare repo error")
			os.Exit(1)
		}
	}
	if ctx.opts.limit == DefaultFileSize && ctx.opts.scan {
		ctx.opts.limit = "1M" // set default to 1M for scan
	}

	PrintLocalWithPlain("current repository size")
	PrintLocalWithYellowln(GetDatabaseSize(ctx.workDir, ctx.bare))
	if lfs := GetLFSObjSize(ctx.workDir); len(lfs) > 0 {
		PrintLocalWithPlain("including LFS objects size")
		PrintLocalWithYellowln(lfs)
	}

	if ctx.opts.scan {
		scanned_targets = ScanMode(ctx)
	} else if ctx.opts.files != nil {
		/* Filter by provided files
		 * Default: file size limit and file type
		 * Max file number limit
		 */
		ctx.scan_t.filepath = true
		NonScanMode(ctx, DefaultFileSize, DefaultFileType, math.MaxUint32)
	} else if ctx.opts.limit != DefaultFileSize {
		/* Filter by file size
		 * Default: file type
		 * Max file number limit
		 */
		ctx.scan_t.filesize = true
		NonScanMode(ctx, ctx.opts.limit, DefaultFileType, math.MaxUint32)
	} else if ctx.opts.types != DefaultFileType {
		/* Filter by file type
		 * Default: file size limit
		 * Max file number limit
		 */
		ctx.scan_t.filetype = true
		NonScanMode(ctx, DefaultFileSize, ctx.opts.types, math.MaxUint32)
	}

	if !ctx.opts.delete {
		os.Exit(1)
	}
	if (ctx.scan_t.filepath || ctx.scan_t.filesize || ctx.scan_t.filetype) && ctx.opts.lfs {
		PrintLocalWithRedln("Convert LFS file error")
		os.Exit(1)
	}
	return scanned_targets, nil
}

// GitDir get .git dir in repository with absolute path
func GitDir(gitbin, path string) (string, error) {
	cmd := exec.Command(gitbin, "-C", path, "rev-parse", "--absolute-git-dir")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf(
			"could not run 'git rev-parse --git-dir': %s", err,
		)
	}
	return string(bytes.TrimSpace(out)), nil
}

// check if the current repository is bare repo
func IsBare(gitbin, path string) (bool, error) {
	cmd := exec.Command(gitbin, "-C", path, "rev-parse", "--is-bare-repository")
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf(LocalPrinter().Sprintf(
			"could not run 'git rev-parse --is-bare-repository': %s", err),
		)
	}
	return string(bytes.TrimSpace(out)) == "true", nil
}

// check if the current repository is shallow repo, need Git version 2.15.0 or newer
func IsShallow(gitbin, path string) (bool, error) {
	cmd := exec.Command(gitbin, "-C", path, "rev-parse", "--is-shallow-repository")
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf(LocalPrinter().Sprintf(
			"could not run 'git rev-parse --is-shallow-repository': %s", err),
		)
	}
	if string(bytes.TrimSpace(out)) == "true" {
		return true, fmt.Errorf(LocalPrinter().Sprintf("could not run in a shallow repo"))
	}
	return false, nil
}

// check if the current repository is flesh clone.
func IsFresh(gitbin, path string) (bool, error) {
	cmd := exec.Command(gitbin, "-C", path, "reflog", "show")
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf(LocalPrinter().Sprintf(
			"could not run 'git reflog show': %s", err),
		)
	}
	return strings.Count(string(out), "\n") < 2, nil
}

// check if Git-LFS has installed in host machine
func HasLFSEnv(gitbin string) (bool, error) {
	cmd := exec.Command(gitbin, "lfs", "version")
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf(LocalPrinter().Sprintf("could not run 'git lfs version': %s", err))
	}
	// #FIXME $?
	return strings.Contains(string(out), "git-lfs"), nil
}

// get git version string
func GitVersion(gitbin string) (string, error) {
	cmd := exec.Command(gitbin, "version")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf(LocalPrinter().Sprintf("could not run 'git version': %s", err))
	}
	matches := Match("[0-9]+.[0-9]+.[0-9]+.?[0-9]?", string(out))
	if len(matches) == 0 {
		return "", fmt.Errorf(LocalPrinter().Sprintf("match git version wrong"))
	}
	return matches[0], nil
}

// convert version string to int number for compare. e.g. convert 2.33.0 to 2330
func GitVersionConvert(version string) int {
	var vstr string
	dict := strings.Split(version, ".")
	if len(dict) == 3 {
		vstr = dict[0] + dict[1] + dict[2]
	}
	if len(dict) == 4 {
		vstr = dict[0] + dict[1] + dict[2] + dict[3]
	}
	vstr = strings.TrimSpace(vstr)
	ret, err := strconv.Atoi(vstr)
	if err != nil {
		return 0
	}
	return ret
}

// get current branch
func GetCurrentBranch(gitbin, path string) (string, error) {
	cmd := exec.Command(gitbin, "-C", path, "symbolic-ref", "HEAD", "--short")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf(LocalPrinter().Sprintf("could not run 'git symbolic-ref HEAD --short': %s", err))
	}
	return strings.TrimSuffix(string(out), "\n"), nil
}

// get current status
func GetCurrentStatus(gitbin, path string) error {
	cmd := exec.Command(gitbin, "-C", path, "status", "-s")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf(LocalPrinter().Sprintf("could not run 'git status'"))
	}
	st := string(out)
	if st == "" {
		return nil
	}
	list := strings.Split(st, "\n")
	for _, ele := range list {
		if strings.HasPrefix(ele, "M ") || strings.HasPrefix(ele, " M") || strings.HasPrefix(ele, "A ") {
			return fmt.Errorf(LocalPrinter().Sprintf("there's some changes to be committed, please commit them first"))
		}
	}
	return nil
}

// get git objects data size
func GetDatabaseSize(dir string, bare bool) string {
	var path string
	if bare {
		path = filepath.Join(dir, ".")
	} else {
		path = filepath.Join(dir, ".git/objects")
	}
	cmd := exec.Command("du", "-hs", path)
	out, err := cmd.Output()
	if err != nil {
		PrintLocalWithRedln("could not run 'du -hs'")
	}
	return strings.TrimSuffix(string(out), "\n")
}

// get lfs objects data size
func GetLFSObjSize(dir string) string {
	path := filepath.Join(dir, ".git/lfs")
	if _, err := os.Stat(path); err == nil {
		cmd := exec.Command("du", "-hs", path)
		out, err := cmd.Output()
		if err != nil {
			PrintLocalWithRedln("could not run 'du -hs .git/lfs/'")
		}
		return strings.TrimSuffix(string(out), "\n")
	}
	return ""
}

// get repo GC url if the repo is hosted on Gitee.com
func GetGiteeGCWeb(gitbin, path string) string {
	cmd := exec.Command(gitbin, "-C", path, "config", "--get", "remote.origin.url")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	url := string(out)
	if url == "" {
		return ""
	}
	if strings.Contains(url, "gitee.com") {
		if strings.HasPrefix(url, "git@") {
			url = strings.TrimPrefix(url, "git@")
			url = "https://" + strings.Replace(url, ":", "/", 1)
		}
		url = strings.TrimSuffix(url, ".git\n") + "/settings#git-gc"
	} else {
		return ""
	}
	return url
}

func GetRepoPath(gitbin, path string) string {
	cmd := exec.Command(gitbin, "-C", path, "worktree", "list")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	repopath := strings.Split(string(out), " ")[0]
	return repopath
}

func BackUp(gitbin, path string) {
	repo_path := GetRepoPath(gitbin, path)
	dst := repo_path + ".bak"
	// check if the same directory exist
	_, err := os.Stat(dst)
	if err == nil {
		ok := AskForOverride()
		if !ok {
			PrintLocalWithYellowln("backup canceled")
			return
		} else {
			os.RemoveAll(dst)
		}
	}
	PrintLocalWithGreenln("start backup")
	cmd := exec.Command(gitbin, "-C", path, "clone", "--no-local", path, dst)
	_, err = cmd.Output()
	if err != nil {
		PrintLocalWithRedln("clone error")
		return
	}
	abs_path, err := filepath.Abs(dst)
	if err != nil {
		PrintLocalWithRedln("run filepach.Abs error")
	}
	ft := LocalPrinter().Sprintf("backup done! Backup file path is: %s", abs_path)
	PrintGreenln(ft)
}

func PushRepo(gitbin, path string) error {
	cmd := exec.Command(gitbin, "-C", path, "push", "origin", "--all", "--force", "--porcelain")
	out1, err := cmd.Output()
	if err != nil {
		PrintLocalWithRedln("Push failed")
		return err
	}
	PrintYellowln(strings.TrimSuffix(string(out1), "\n"))

	cmd2 := exec.Command(gitbin, "-C", path, "push", "origin", "--tags", "--force")
	out2, err := cmd2.Output()
	if err != nil {
		PrintLocalWithRedln("Push failed")
		return err
	}
	PrintYellowln(strings.TrimSuffix(string(out2), "\n"))
	PrintLocalWithYellowln("Done")
	return nil
}

// BrachesChanged prints all branches that have been changed
func BrachesChanged() bool {
	branches := Branch_changed.ToSlice()
	if len(branches) != 0 {
		PrintLocalWithYellowln("branches have been changed")
		for _, branch := range branches {
			s := strings.TrimSpace(branch.(string))
			if strings.HasPrefix(s, "refs/heads/") {
				PrintYellowln(strings.TrimPrefix(s, "refs/heads/"))
			}

			if strings.HasPrefix(s, "refs/tags/") {
				PrintYellowln(strings.TrimPrefix(s, "refs/tags/"))
			}

			if strings.HasPrefix(s, "refs/remotes/") {
				PrintYellowln(strings.TrimPrefix(s, "refs/remotes/"))
			}
		}
		fmt.Println()
		return true
	}
	return false
}

func (context Context) CleanUp() {
	if BrachesChanged() || context.opts.lfs {
		// clean up
		PrintLocalWithGreenln("file cleanup is complete. Start cleaning the repository")
	} else {
		// exit
		PrintLocalWithYellowln("nothing have changed, exit...")
		os.Exit(1)
	}

	if !context.bare {
		fmt.Println("running git reset --hard")
		cmd1 := exec.Command(context.gitBin, "-C", context.workDir, "reset", "--hard")
		cmd1.Stdout = os.Stdout
		err := cmd1.Start()
		if err != nil {
			PrintRedln(fmt.Sprint(err))
		}
		err = cmd1.Wait()
		if err != nil {
			PrintRedln(fmt.Sprint(err))
		}
	}

	fmt.Println("running git reflog expire --expire=now --all")
	cmd2 := exec.Command(context.gitBin, "-C", context.workDir, "reflog", "expire", "--expire=now", "--all")
	cmd2.Stderr = os.Stderr
	cmd2.Stdout = os.Stdout
	err := cmd2.Start()
	if err != nil {
		PrintRedln(fmt.Sprint(err))
	}
	err = cmd2.Wait()
	if err != nil {
		PrintRedln(fmt.Sprint(err))
	}

	fmt.Println("running git gc --prune=now")
	cmd3 := exec.Command(context.gitBin, "-C", context.workDir, "gc", "--prune=now")
	cmd3.Stderr = os.Stderr
	cmd3.Stdout = os.Stdout
	err = cmd3.Start()
	if err != nil {
		PrintRedln(fmt.Sprint(err))
	}
	cmd3.Wait()
	if err != nil {
		PrintRedln(fmt.Sprint(err))
	}
}

func LFSPrompt() {
	FilesChanged()
	PrintLocalWithPlainln("before you push to remote, you have to do something below:")
	PrintLocalWithYellowln("1. install git-lfs")
	PrintLocalWithYellowln("2. run command: git lfs install")
	PrintLocalWithYellowln("3. edit .gitattributes file")
	PrintLocalWithYellowln("4. commit your .gitattributes file.")
}

func (context Context) Prompt() {
	PrintLocalWithGreenln("cleaning completed")
	PrintLocalWithPlain("current repository size")
	PrintLocalWithYellowln(GetDatabaseSize(context.workDir, context.bare))
	if lfs := GetLFSObjSize(context.workDir); len(lfs) > 0 {
		PrintLocalWithPlain("including LFS objects size")
		PrintLocalWithYellowln(lfs)
	}
	if context.opts.lfs {
		LFSPrompt()
	}
	var pushed bool
	if !context.opts.lfs {
		if AskForUpdate() {
			PrintLocalWithPlainln("execute force push")
			PrintLocalWithYellowln("git push origin --all --force")
			PrintLocalWithYellowln("git push origin --tags --force")
			err := PushRepo(context.gitBin, context.workDir)
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
	url := GetGiteeGCWeb(context.gitBin, context.workDir)
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
