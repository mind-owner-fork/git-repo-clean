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
	"github.com/github/git-sizer/sizes"
)

type BlobRecord struct {
	oid        git.OID
	objectSize counts.Count32
}

type HistoryRecord struct {
	sizes.HistorySize
	bigblob []BlobRecord
}

func (repo *Repository) GetBlobName(oid string) (string, error) {

	cmd := repo.gitCommand("rev-list", "--objects", "--all")

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

func ScanRepository(repo Repository, op Options) (HistoryRecord, error) {
	graph := sizes.NewGraph(sizes.NameStyleFull)
	history := HistoryRecord{}

	if op.verbose {
		fmt.Println("Start to scan repository: ")
	}

	// get reference iter
	refIter, err := repo.NewReferenceIter()
	if err != nil {
		return history, err
	}
	defer func() {
		if refIter != nil {
			refIter.Close()
		}
	}()

	// get object iter
	iter, in, err := repo.NewObjectIter("--stdin", "--date-order")
	if err != nil {
		return history, err
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

	var blobs []BlobRecord

	// process blobs
	for {
		oid, objectType, objectSize, err := iter.Next()
		if err != nil {
			if err != io.EOF {
				return history, err
			}
			break
		}
		switch objectType {
		case "blob":
			graph.RegisterBlob(oid, objectSize)

			limit, err := UnitConvert(op.limit)
			if err != nil {
				return history, err
			}
			if objectSize > counts.Count32(limit) {
				// append this record blob into slice
				blobs = append(blobs, BlobRecord{oid, objectSize})
				// sort
				sort.Slice(blobs, func(i, j int) bool {
					return blobs[i].objectSize > blobs[j].objectSize
				})
				// remain first [op.number] blobs
				if len(blobs) > int(op.number) {
					blobs = blobs[:op.number]
				}
			}
		default:
			err = fmt.Errorf("expected blob object type, but got: %s", objectType)
		}

	}
	return HistoryRecord{graph.HistorySize(), blobs}, err
}
