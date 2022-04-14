package main

import (
	"os"
	"path/filepath"
	"strconv"
)

type RepoFilter struct {
	repo      *Repository
	scanned   []string // file's oid provided by scanner
	filepaths []string // files(or dir) provided by user
}

func (filter *RepoFilter) tweak_blob(blob *Blob) {
	for _, target := range filter.scanned {
		if target == blob.original_oid {
			// replace old blob with new LFS info
			if filter.repo.opts.lfs {
				ConvertToLFSObj(blob)
				UpdateBlob(blob)
				break
			}
			// set new id to 0
			blob.ele.skip(0)
		}
	}
}

func (filter *RepoFilter) tweak_commit(commit *Commit, helper *Helper_info) {

	// 如果没有parent, 且也没有filechange，则first commit是empty commit
	if len(commit.parents) == 0 && len(commit.filechanges) == 0 {
		return
	}
	// 如果有parent，但是没有filechange， 则可能是merge commit, 或者连续的empty commit
	if len(commit.parents) != 0 && len(commit.filechanges) == 0 {
		return
	}
	// 如果有parent，且from：0, 说明是从第一个blob就开始删除了
	if len(commit.parents) != 0 && commit.parents[0] == 0 {
		commit.skip(0)
	}

	old_1st_parent := commit.first_parent()

	filter_filechange(commit, filter)

	if len(commit.filechanges) == 0 {
		commit.skip(old_1st_parent)
	}

	// 如果 from-id 在ID-hash中能够查询到，则正常，否则说明parent commit被删了
	// 或者，如果from-id在Skipped-commit中能够查询到，则也需要skip
	if SKIPPED_COMMITS.Contains(old_1st_parent) {
		commit.skip(old_1st_parent)
	}
}

func filter_filechange(commit *Commit, filter *RepoFilter) {
	newfilechanges := make([]FileChange, 0)
	matched := false
	for _, filechange := range commit.filechanges {
		// scan mode, filter by blob oid
		if filter.repo.opts.scan {
			for _, target := range filter.scanned {
				if len(filechange.blob_id) == 40 {
					if target == filechange.blob_id {
						Branch_changed.Add(filechange.branch)
						matched = true
						break // break inner for-loop
					}
				} else {
					if filechange.changetype == "M" {
						id, _ := strconv.Atoi(filechange.blob_id)
						if _, ok := ID_HASH[int32(id)]; !ok {
							Branch_changed.Add(filechange.branch)
							matched = true
							break // break inner for-loop
						}
					}
				}
			}
		} else {
			// filter by blob size threshold
			if filter.repo.opts.limit != "" {
				objectsize := Blob_size_list[filechange.blob_id]
				// set bitsize to 64, means max single blob size is 4 GiB
				size, _ := strconv.ParseUint(objectsize, 10, 64)
				limit, err := UnitConvert(filter.repo.opts.limit)
				if err != nil {
					ft := LocalPrinter().Sprintf("convert uint error: %s", err)
					PrintRedln(ft)
					os.Exit(1)
				}
				if size > limit {
					Branch_changed.Add(filechange.branch)
					matched = true
				}
			}
			// filter by file type
			if filter.repo.opts.types != "*" {
				if filepath.Ext(filechange.filepath) == "."+filter.repo.opts.types {
					Branch_changed.Add(filechange.branch)
					matched = true
				}
			}
			// filter by blob name or directory
			if len(filter.filepaths) != 0 {
				for _, path := range filter.filepaths {
					if path == EndcodePath(TrimeDoubleQuote(filechange.filepath)) {
						Branch_changed.Add(filechange.branch)
						matched = true
					}
				}
			}
		}
		if matched {
			// skip this file
			continue
		}
		// otherwise, keep it in newfilechange
		newfilechanges = append(newfilechanges, filechange)
	}
	commit.filechanges = newfilechanges
}

func (filter *RepoFilter) tweak_reset(reset *Reset) {
	if SKIPPED_COMMITS.Contains(reset.from) {
		reset.base.dumped = false
		reset.base.skip()
	}
}

func (filter *RepoFilter) tweak_tag(tag *Tag) {
	// the tag may have no parent, if so skip it
	if SKIPPED_COMMITS.Contains(tag.from_ref) {
		tag.ele.base.dumped = false
		tag.ele.skip(0)
	}
}
