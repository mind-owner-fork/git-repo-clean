package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

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
	objectSize int32
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

/*
$ git diff-tree -r HEAD^^ HEAD^
:100644 000000 257cc5642cb1a054f08cc83f2d943e56fd3ebe99 0000000000000000000000000000000000000000 D      "path with\nnewline"
:000000 100644 0000000000000000000000000000000000000000 257cc5642cb1a054f08cc83f2d943e56fd3ebe99 A      "subdir/path with\nnewline"
*/
func (repo Repository) GetFilechange(parent_hash, commit_hash string) []FileChange {
	cmd := repo.GitCommand("diff-tree", "r", parent_hash, commit_hash)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return []FileChange{}
	}
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		return []FileChange{}
	}

	buf := bufio.NewReader(out)
	var filechanges []FileChange
	for {
		raw_line, err := buf.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				return []FileChange{}
			}
			return []FileChange{}
		}
		raw_line = raw_line[:len(raw_line)-1]
		line := strings.Split(raw_line, "\t")
		// :000000 100644 0000000000000000000000000000000000000000 257cc5642cb1a054f08cc83f2d943e56fd3ebe99 A
		fileinfo := line[0]
		filepath := line[1]

		info := strings.Split(fileinfo, " ")
		// info[0]: old-mode
		// info[1]: new-mode
		// info[2]: old-hash
		// info[3]: new-hash
		// info[4]: file-type
		if info[4] == "D" {
			filechanges = append(filechanges, NewFileChange("D", "", "", filepath))
		} else if info[4] == "A" || info[4] == "M" || info[4] == "T" {
			id := HASH_ID[info[3]]
			filechanges = append(filechanges, NewFileChange("M", info[1], string(id), filepath))
		} else {
			// un-support type
			fmt.Println("ERROR:unsupport filechange type")
			break
		}
	}
	return filechanges
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
				blobs = append(blobs, HistoryRecord{oid.String(), int32(objectSize), name})
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

	return strings.Split(string(out), " ")[2], nil
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
	cmd1 := repo.GitCommand("reset", "--hard")
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
	cmd2 := repo.GitCommand("reflog", "expire", "--expire=now", "--all")
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
	cmd3 := repo.GitCommand("gc", "--prune=now")
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
