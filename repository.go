package main

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/github/git-sizer/counts"
	"github.com/github/git-sizer/git"
	"github.com/github/git-sizer/meter"
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

func UnitConvert(input string) (uint64, error) {
	v := input[:len(input)-1]
	u := input[len(input)-1:]
	cv, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		return 0, err
	}
	if strings.ToLower(u) == "k" {
		return cv * 1024, nil
	} else if strings.ToLower(u) == "m" {
		return cv * 1024 * 1024, nil
	} else if strings.ToLower(u) == "g" {
		return cv * 1024 * 1024 * 1024, nil
	} else {
		err := fmt.Errorf("expected format: --limit=<n>k|m|g, but you are: --limit=%s", input)
		return 0, err
	}
}

func ScanRepository(repo git.Repository, op Options) (HistoryRecord, error) {
	graph := sizes.NewGraph(sizes.NameStyleFull)
	history := HistoryRecord{}
	var progressMeter meter.Progress

	if op.verbose {
		progressMeter = meter.NewProgressMeter(100 * time.Millisecond)
	} else {
		progressMeter = &meter.NoProgressMeter{}
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

	//  handle objects commits, use refs as stdin
	type ObjectHeader struct {
		oid        git.OID
		objectSize counts.Count32
	}
	type CommitHeader struct {
		ObjectHeader
		tree git.OID
	}

	var trees, tags []ObjectHeader
	var commits []CommitHeader
	var blobs []BlobRecord

	// process blobs
	if op.ranges == "blobs" || op.ranges == "full" {
		progressMeter.Start("Starting blob: %d")
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
				progressMeter.Inc()
				graph.RegisterBlob(oid, objectSize)

				limit, err := UnitConvert(op.limit)
				if err != nil {
					return history, err
				}
				if objectSize > counts.Count32(limit) {
					// append this record blob into slice
					blobs = append(blobs, BlobRecord{oid, objectSize})
					sort.Slice(blobs, func(i, j int) bool {
						return blobs[i].objectSize > blobs[j].objectSize
					})
					// cut the last one
					if len(blobs) > int(op.number) {
						blobs = blobs[:op.number]
					}
				}
			case "tree":
				trees = append(trees, ObjectHeader{oid, objectSize})
			case "commit":
				commits = append(commits, CommitHeader{ObjectHeader{oid, objectSize}, git.NullOID})
			case "tag":
				tags = append(tags, ObjectHeader{oid, objectSize})
			default:
				err = fmt.Errorf("unexpected object type: %s", objectType)
			}

		}
		progressMeter.Done()
	}

	if op.ranges == "trees" || op.ranges == "full" {
		fmt.Printf("trees number: %d\n", len(trees))
	}
	if op.ranges == "commits" || op.ranges == "full" {
		fmt.Printf("commits number: %d\n", len(commits))
	}
	if op.ranges == "refs" || op.ranges == "full" {
		fmt.Printf("refs number: %d\n", len(refs))
	}
	return HistoryRecord{graph.HistorySize(), blobs}, err
}
