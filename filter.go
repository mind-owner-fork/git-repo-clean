package main

import (
	"math"
	"os"
	"path/filepath"
	"strconv"
)

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
		scanned_targets = scanMode(ctx)
	} else if ctx.opts.files != nil {
		/* Filter by provided files
		 * Default: file size limit and file type
		 * Max file number limit
		 */
		ctx.scan_t.filepath = true
		nonScanMode(ctx, DefaultFileSize, DefaultFileType, math.MaxUint32)
	} else if ctx.opts.limit != DefaultFileSize {
		/* Filter by file size
		 * Default: file type
		 * Max file number limit
		 */
		ctx.scan_t.filesize = true
		nonScanMode(ctx, ctx.opts.limit, DefaultFileType, math.MaxUint32)
	} else if ctx.opts.types != DefaultFileType {
		/* Filter by file type
		 * Default: file size limit
		 * Max file number limit
		 */
		ctx.scan_t.filetype = true
		nonScanMode(ctx, DefaultFileSize, ctx.opts.types, math.MaxUint32)
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

func nonScanMode(ctx *Context, file_limit string, file_type string, file_num uint32) {
	ctx.opts.limit = file_limit
	ctx.opts.types = file_type
	ctx.opts.number = file_num
}

func scanMode(ctx *Context) (result []string) {
	var first_target []string

	bloblist, err := ctx.ScanRepository()
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

///////////////////////////////////////////////////////
// tweak git objects

func (repo *Repository) tweak_blob(blob *Blob) {
	for _, target := range repo.filtered {
		if target == blob.original_oid {
			// replace old blob with new LFS info
			if repo.context.opts.lfs {
				ConvertToLFSObj(blob)
				UpdateBlob(blob)
				break
			}
			// set new id to 0
			blob.ele.skip(0)
		}
	}
}

func (repo *Repository) tweak_commit(commit *Commit, helper *Helper_info) {
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

	filter_filechange(commit, repo)

	if len(commit.filechanges) == 0 {
		commit.skip(old_1st_parent)
	}

	// 如果 from-id 在ID-hash中能够查询到，则正常，否则说明parent commit被删了
	// 或者，如果from-id在Skipped-commit中能够查询到，则也需要skip
	if SKIPPED_COMMITS.Contains(old_1st_parent) {
		commit.skip(old_1st_parent)
	}
}

func filter_filechange(commit *Commit, repo *Repository) {
	newfilechanges := make([]FileChange, 0)
	matched := false
	for _, filechange := range commit.filechanges {
		// scan mode, filter by blob oid
		if repo.context.opts.scan {
			for _, target := range repo.filtered {
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
			if repo.context.scan_t.filesize {
				objectsize := Blob_size_list[filechange.blob_id]
				// set bitsize to 64, means max single blob size is 4 GiB
				size, _ := strconv.ParseUint(objectsize, 10, 64)
				limit, err := UnitConvert(repo.context.opts.limit)
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
			if repo.context.scan_t.filetype {
				if filepath.Ext(filechange.filepath) == "."+repo.context.opts.types {
					Branch_changed.Add(filechange.branch)
					matched = true
				}
			}
			// filter by blob name or directory
			if repo.context.scan_t.filepath {
				for _, path := range repo.context.opts.files {
					matches := Match(path, EndcodePath(TrimeDoubleQuote(filechange.filepath)))
					if len(matches) != 0 {
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

func (repo *Repository) tweak_reset(reset *Reset) {
	if SKIPPED_COMMITS.Contains(reset.from) {
		reset.base.dumped = false
		reset.base.skip()
	}
}

func (repo *Repository) tweak_tag(tag *Tag) {
	// the tag may have no parent, if so skip it
	if SKIPPED_COMMITS.Contains(tag.from_ref) {
		tag.ele.base.dumped = false
		tag.ele.skip(0)
	}
}
