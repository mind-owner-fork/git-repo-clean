package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/github/git-sizer/counts"
	"github.com/github/git-sizer/git"
)

type HistoryRecord struct {
	oid        string
	objectSize uint32
	objectName string
}

type BlobList []HistoryRecord

func (repo Repository) GetBlobName(oid string) (string, error) {
	cmd := repo.GitCommand("rev-list", "--objects", "--all")

	out, err := cmd.StdoutPipe()
	if err != nil {
		return "", nil
	}
	cmd.Stderr = os.Stderr

	err = cmd.Start()
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
		// return "", nil
	}
	cmd.Stderr = os.Stderr

	err = cmd.Start()
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
			id := Hash_id[info[3]]
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
		fmt.Println("开始扫描...")
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
					fmt.Println("get blob name fail, but will continue to try to get next one")
					continue
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
		default:
			err = fmt.Errorf("expected blob object type, but got: %s", objectType)
		}

	}
	return blobs, err
}
