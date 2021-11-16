package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"path/filepath"

	"github.com/github/git-sizer/counts"
	"github.com/github/git-sizer/git"
)

type Repository struct {
	git.Repository
	path   string
	gitBin string
	gitDir string
	opts   Options
}

type HistoryRecord struct {
	oid        string
	objectSize uint32
	objectName string
}

type BlobList []HistoryRecord

func (repo Repository) GetBlobName(oid string) (string, error) {
	cmd := exec.Command(repo.gitBin, "rev-list", "--objects", "--all")
	out, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		return "", err
	}

	buf := bufio.NewReader(out)
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				return "", err
			}
			return "", nil
		}
		// drop LF
		line = line[:len(line)-1]

		if len(line) <= 41 {
			continue
		}
		texts := strings.Split(line, " ")
		if texts[0] == oid {
			blobname := texts[1]
			return blobname, nil
		}
	}
}

func ScanRepository(repo Repository) (BlobList, error) {

	var empty []HistoryRecord
	var blobs []HistoryRecord

	if repo.opts.verbose {
		PrintGreen("开始扫描(如果仓库过大，扫描时间会比较长，请耐心等待)...")
	}

	// get reference iter
	refIter, err := repo.NewReferenceIter()
	if err != nil {
		return empty, err
	}
	defer func() {
		if refIter != nil {
			refIter.Close()
		}
	}()

	// get object iter
	iter, in, err := repo.NewObjectIter("--stdin", "--date-order")
	if err != nil {
		return empty, err
	}
	defer func() {
		if iter != nil {
			iter.Close()
		}
	}()

	// parse references
	errChan := make(chan error, 1)
	var refs []git.Reference
	go func() {
		defer in.Close()
		bufin := bufio.NewWriter(in)
		defer bufin.Flush()

		for {
			ref, ok, err := refIter.Next()
			if err != nil {
				errChan <- err
				return
			}
			if !ok {
				break
			}
			// save ref into refs list
			refs = append(refs, ref)
			_, err = bufin.WriteString(ref.OID.String())
			if err != nil {
				errChan <- err
				return
			}
			err = bufin.WriteByte('\n')
			if err != nil {
				errChan <- err
				return
			}
		}

		err := refIter.Close()
		errChan <- err
		refIter = nil
	}()

	// process blobs
	for {
		oid, objectType, objectSize, err := iter.Next()
		if err != nil {
			if err != io.EOF {
				return empty, err
			}
			break
		}
		switch objectType {
		case "blob":
			limit, err := UnitConvert(repo.opts.limit)
			if err != nil {
				return empty, err
			}
			if objectSize > counts.Count32(limit) {

				name, err := repo.GetBlobName(oid.String())
				if err != nil {
					fmt.Printf("run GetBlobName error: %s\n", err)
					os.Exit(1)
				}

				if len(repo.opts.types) != 0 || repo.opts.types != "*" {
					var pattern string
					if strings.HasSuffix(name, "\"") {
						pattern = "." + repo.opts.types + "\"$"
					} else {
						pattern = "." + repo.opts.types + "$"
					}
					if matches := Match(pattern, name); len(matches) == 0 {
						// matched none, skip
						continue
					}
				}
				// append this record blob into slice
				blobs = append(blobs, HistoryRecord{oid.String(), uint32(objectSize), name})
				// sort according by size
				sort.Slice(blobs, func(i, j int) bool {
					return blobs[i].objectSize > blobs[j].objectSize
				})
				// remain first [op.number] blobs
				if len(blobs) > int(repo.opts.number) {
					blobs = blobs[:repo.opts.number]
				}
			}
		case "tree":
		case "commit":
		case "tag":
			continue
		default:
			err = fmt.Errorf("expected blob object type, but got: %s", objectType)
			return empty, err
		}

	}

	err = <-errChan
	if err != nil {
		return empty, err
	}

	err = iter.Close()
	iter = nil
	if err != nil {
		return empty, err
	}
	return blobs, err
}

// check if the current repository is bare repo
func IsBare(gitbin, path string) (bool, error) {

	cmd := exec.Command(gitbin, "-C", path, "rev-parse", "--is-bare-repository")
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf(
			"could not run 'git rev-parse --is-bare-repository': %s", err,
		)
	}

	return string(bytes.TrimSpace(out)) == "true", nil
}

// check if the current repository is shallow repo, need Git version 2.15.0 or newer
func IsShallow(gitbin, path string) (bool, error) {
	cmd := exec.Command(gitbin, "-C", path, "rev-parse", "--is-shallow-repository")
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf(
			"could not run 'git rev-parse --is-shallow-repository': %s", err,
		)
	}

	return string(bytes.TrimSpace(out)) == "true", nil
}

// check if the current repository is flesh clone.
func IsFresh(gitbin, path string) (bool, error) {
	cmd := exec.Command(gitbin, "-C", path, "reflog", "show")
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf(
			"could not run 'git reflog show': %s", err,
		)
	}
	return strings.Count(string(out), "\n") < 2, nil
}

// check if Git-LFS has installed in host machine
func HasLFSEnv(gitbin, path string) (bool, error) {
	cmd := exec.Command(gitbin, "-C", path, "lfs", "version")
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("could not run 'git lfs version': %s", err)
	}

	return strings.Contains(string(out), "git-lfs"), nil
}

// get git version string
func GitVersion(gitbin, path string) (string, error) {
	cmd := exec.Command(gitbin, "-C", path, "version")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("could not run 'git version': %s", err)
	}
	matches := Match("[0-9]+.[0-9]+.[0-9]?", string(out))
	if len(matches) == 0 {
		return "", errors.New("match git version wrong")
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
	vstr = strings.TrimSuffix(vstr, "\n")
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
		return "", fmt.Errorf("could not run 'git symbolic-ref HEAD --short': %s", err)
	}
	return strings.TrimSuffix(string(out), "\n"), nil
}

// get current status
func GetCurrentStatus(gitbin, path string) {
	cmd := exec.Command(gitbin, "-C", path, "status", "-z")
	out, err := cmd.Output()
	if err != nil {
		PrintRed("执行 'git status' 出错！")
	}
	st := string(out)
	if st == "" {
		PrintGreen("git status clean")
	}
	fmt.Println(st)
}

func GetDatabaseSize(gitbin, path string) string {
	path = filepath.Join(path, ".git")
	cmd := exec.Command("du", "-hs", path)
	out, err := cmd.Output()
	if err != nil {
		PrintRed("执行 'du -hs .git' 出错！")
	}
	return strings.TrimSuffix(string(out), "\n")
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

func (repo *Repository) BackUp(gitbin, path string) {
	PrintGreen("开始准备仓库数据")
	// #TODO specify backup directory by option
	dst := "../backup.bak"
	// check if the same directory exist
	_, err := os.Stat(dst)
	if err == nil {
		ok := AskForOverride()
		if !ok {
			PrintYellow("已取消备份")
			return
		} else {
			os.RemoveAll(dst)
		}
	}
	PrintGreen("开始备份...")
	cmd := exec.Command(gitbin, "-C", path, "clone", "--quiet", "--no-local", path, dst)
	out, err := cmd.Output()
	fmt.Println(string(out))
	if err != nil {
		PrintRed("克隆错误")
		fmt.Println(err)
		return
	}
	abs_path, err := filepath.Abs(dst)
	if err != nil {
		PrintRed("run filepach.Abs error")
	}
	fmter := fmt.Sprintf("备份完毕! 备份文件路径为：%s\n", abs_path)
	PrintGreen(fmter)
}

func (repo *Repository) PushRepo(gitbin, path string) error {
	cmd := exec.Command(gitbin, "-C", path, "push", "origin", "--all", "--force", "--porcelain")
	out1, err := cmd.Output()
	if err != nil {
		PrintRed("推送失败，可能是没有权限推送，或者该仓库没有设置远程仓库")
		return err
	}
	PrintYellow(strings.TrimSuffix(string(out1), "\n"))

	cmd2 := exec.Command(gitbin, "-C", path, "push", "origin", "--tags", "--force")
	out2, err := cmd2.Output()
	if err != nil {
		PrintRed("推送失败，可能网络不稳定，或者该仓库没有设置远程仓库'")
		return err
	}
	PrintYellow(strings.TrimSuffix(string(out2), "\n"))
	PrintYellow("Done")
	return nil
}

func NewRepository(path string) (*Repository, error) {
	// Find the `git` executable to be used:
	gitBin, err := findGitBin()
	if err != nil {
		return nil, fmt.Errorf(
			"could not find 'git' executable (is it in your PATH?): %v", err,
		)
	}
	gitdir, err := GitDir(gitBin, path)
	if err != nil {
		return &Repository{}, err
	}

	if bare, err := IsBare(gitBin, path); err != nil || bare {
		return &Repository{}, err
	}

	if shallow, err := IsShallow(gitBin, path); err != nil || shallow {
		return &Repository{}, err
	}

	return &Repository{
		path:   path,   // working dir
		gitDir: gitdir, // .git dir
		gitBin: gitBin,
	}, nil
}

func (repo *Repository) Close() error {
	return nil
}

func (repo *Repository) CleanUp() {
	// clean up
	PrintGreen("文件清理完毕，开始清理仓库...")

	fmt.Println("running git reset --hard")
	cmd1 := exec.Command(repo.gitBin, "-C", repo.path, "reset", "--hard")
	cmd1.Stdout = os.Stdout
	err := cmd1.Start()
	if err != nil {
		fmt.Println(err)
	}
	err = cmd1.Wait()
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("running git reflog expire --expire=now --all")
	cmd2 := exec.Command(repo.gitBin, "-C", repo.path, "reflog", "expire", "--expire=now", "--all")
	cmd2.Stderr = os.Stderr
	cmd2.Stdout = os.Stdout
	err = cmd2.Start()
	if err != nil {
		fmt.Println(err)
	}
	err = cmd2.Wait()
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("running git gc --prune=now")
	cmd3 := exec.Command(repo.gitBin, "-C", repo.path, "gc", "--prune=now")
	cmd3.Stderr = os.Stderr
	cmd3.Stdout = os.Stdout
	err = cmd3.Start()
	if err != nil {
		fmt.Println(err)
	}
	cmd3.Wait()
	if err != nil {
		fmt.Println(err)
	}
}
