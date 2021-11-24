package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const (
	idx_re  = `(\d+)\n$`              // mark :1, from :1, merge :1
	oid_re  = `([0-9a-f]{40})\n$`     // original-oid 401fb905f1abf1d35331d0cddc8556ba23c1a212
	user_re = `(.*?) <(.*?)> (.*)\n$` // author|commiter|tagger Li Linchao <lilinchao@oschina.cn> 1633964331 +0800
	ref_re  = `(.*)\n$`               // commit|reset|tag refs/tags/v1.0.0
)

var (
	Lasted_commit      = make(map[string]int32)
	Lasted_orig_commit = make(map[string]int32)
)

type origParents []int32
type hasFilechange bool

type Helper_info struct {
	orig_parents   origParents
	has_filechange hasFilechange
}

func Match(pattern string, str string) []string {
	re := regexp.MustCompile(pattern)
	return re.FindStringSubmatch(str)
}

/*
blob
mark :1
original-oid 401fb905f1abf1d35331d0cddc8556ba23c1a212
data 9
"file a"
*/
type Blob struct {
	ele          *GitElementsWithID // contain: id, old_id, types, dumped
	original_oid string             // 40 bytes
	data_size    int64              // blob size maybe very large
	data         []byte             // raw data block
}

func NewBlob(size_ int64, data_ []byte, hash_id_ string) Blob {
	var ele = NewGitElementsWithID()
	ele.base.types = "blob"
	return Blob{
		ele:          &ele,
		original_oid: hash_id_,
		data_size:    size_,
		data:         data_,
	}
}

func (blob Blob) dump(writer io.WriteCloser) {
	blob.ele.base.dumped = true
	HASH_ID[blob.original_oid] = blob.ele.id
	ID_HASH[blob.ele.id] = blob.original_oid

	mark_line := fmt.Sprintf("mark :%d\n", blob.ele.id)
	oid_line := fmt.Sprintf("original-oid %s\n", blob.original_oid)
	data_line := fmt.Sprintf("data %d\n%s\n", blob.data_size, blob.data)

	writer.Write([]byte("blob\n"))
	writer.Write([]byte(mark_line))
	writer.Write([]byte(oid_line))
	writer.Write([]byte(data_line))
}

/*
the filechange format is: "type mode id filepath\n"

M 100644 :18 files/1.c
M 100644 :18 files/2.c
M 100644 :16 output

type specification:
M: modify
A: add
C: copy
D: delete

mode specification:
100644 or 644: normal, but non executable file
100755 or 755: normal, but executable file
120000: symlink
160000: gitlink
040000: *subdirectory

**NOTE**
when the mode is "040000", the id must be 40-byte HASH-1 value, but not short mark id

filechange can compose together :
	D A
	M 100644 :16 dir/B
this means rename a file and move it from another location
*/
type FileChange struct {
	base       *GitElements
	changetype string
	mode       string
	blob_id    string
	filepath   string
}

// **NOTE**
// when type is "M", mode and id must not nil, when type is "D", mode and id is nil
//
// filechange usually have multi-line
func NewFileChange(types_, mode_, id_, filepath_ string) FileChange {
	var base = NewGitElement()
	base.types = "filechange"
	return FileChange{
		base:       &base,
		changetype: types_,
		mode:       mode_,
		blob_id:    id_,
		filepath:   filepath_,
	}
}

func (fc *FileChange) dumpToString() string {
	if fc.changetype == "M" {
		if fc.mode == "160000" || fc.mode == "040000" { // gitlink or subdirectory
			filechange_ := fmt.Sprintf("M %s %s %s\n", fc.mode, fc.blob_id, fc.filepath)
			return filechange_
		} else {
			filechange_ := fmt.Sprintf("M %s :%s %s\n", fc.mode, fc.blob_id, fc.filepath)
			return filechange_
		}
	} else if fc.changetype == "D" {
		filechange_ := fmt.Sprintf("D %s\n", fc.filepath)
		return filechange_
	}
	return ""
}

func (fc *FileChange) dump(writer io.WriteCloser) {
	if fc.changetype == "M" && fc.blob_id == "0" {
		return
	}
	fc.base.dumped = true
	// currently only consider M type "M 100644 :18 files/1.c" and D type "D files/1.c"
	// when the type is M, and the id is short mark id
	if fc.changetype == "M" {
		if fc.mode == "160000" || fc.mode == "040000" {
			filechange_ := fmt.Sprintf("M %s %s %s\n", fc.mode, fc.blob_id, fc.filepath)
			writer.Write([]byte(filechange_))
		} else {
			filechange_ := fmt.Sprintf("M %s :%s %s\n", fc.mode, fc.blob_id, fc.filepath)
			writer.Write([]byte(filechange_))
		}
	} else if fc.changetype == "D" {
		filechange_ := fmt.Sprintf("D %s\n", fc.filepath)
		writer.Write([]byte(filechange_))
	} else if fc.changetype == "R" {
		// **NOTE** if we don't add '-M' in git-fast-export, then it will use 'M' and 'D' to represent 'R'
		// filepath_ := fmt.Sprintf("R %s %s\n", fc.old-path, new-path)
		// writer.Write([]byte(filepath_))
	} else {
		// unhandle filechange type
		PrintLocalWithRedln("unsupported filechange type")
		return
	}
}

/*

commit refs/heads/4
mark :25
original-oid daca020f8360e0b2ea383e195b09b9c6a4a4979b
author Li Linchao <lilinchao@oschina.cn> 1634117087 +0800
committer Li Linchao <lilinchao@oschina.cn> 1634117087 +0800
data 39
Merge branches '5', '6' and '7' into 4  <---- merge three branches into one
from :20
merge :22								<---- one merge command
merge :24								<---- and another one
M 100644 :21 6.md
M 100644 :23 7.md

The author format is：<author_name> SP <author_email> SP <author_date> LF
and the commiter format is : <committer_name> SP <committer_email> SP <committer_date> LF

As for the format of "from", see:
https://git-scm.com/docs/git-fast-import#_from

**Special Case 1:**

commit refs/heads/review-mode/readonly
mark :23
original-oid 46d3d959019ffe22c8399755dd7028cece366ccd
author Cactusinhand <lilinchao@oschina.cn> 1626747427 +0000
committer Gitee <noreply@gitee.com> 1626747427 +0000
data 99
!13 change readonly file
Merge pull request !13 from Cactusinhand/auto-7670704-master-1626686679876from :4   <-------------
merge :22
M 100644 :21 README.en.md
M 100644 :5 files/1.c

**Special Case 2:**
reset refs/heads/review-mode/readonly
commit refs/heads/review-mode/readonly
mark :4
original-oid 1072abf2c53ce53ad88cb6fdea64d2cf51f9fac8
author Cactusinhand <lilinchao@oschina.cn> 1626405532 +0000
committer Gitee <noreply@gitee.com> 1626405532 +0000
data 14
Initial commitM 100644 :1 .gitee/PULL_REQUEST_TEMPLATE.zh-CN.md  <-------------
M 100644 :2 README.en.md
M 100644 :3 README.md

**NOTE**

a). When a commit has no parent(usually the first commit), then there will be a reset command in the front of commit command.

c). When merge multi branches into one, a commit have multi parents, then there will be at least one merge commands.

c). When a commit has parent commit, then it has from or merge(or both), otherwise, have none of them.
*/
type Commit struct {
	ele          *GitElementsWithID // mark id
	old_id       int32              // previous mark id, given by GitElementsWithID
	original_oid string
	branch       string
	author       string
	commiter     string
	msg_size     int32
	message      []byte       // commit message
	parents      []int32      // from and merge. from maybe none, and merge maybe multi
	filechanges  []FileChange // multi-line
}

func parent_filter(str []int32) (int32, []int32) {
	var empty []int32
	if len(str) == 0 {
		return 0, empty
	} else {
		return str[0], str[1:]
	}
}

func NewCommit(original_oid_, branch_, author_, commiter_ string, size_ int32, msg_ []byte, parents_ []int32, filechanges_ []FileChange) Commit {
	var ele = NewGitElementsWithID()
	ele.base.types = "commit"
	return Commit{
		ele:          &ele,
		old_id:       ele.id,
		original_oid: original_oid_,
		branch:       branch_,
		author:       author_,
		commiter:     commiter_,
		msg_size:     size_,
		message:      msg_,
		parents:      parents_,
		filechanges:  filechanges_,
	}
}

func (commit *Commit) dump(writer io.WriteCloser) {

	commit.ele.base.dumped = true

	HASH_ID[commit.original_oid] = commit.ele.id
	ID_HASH[commit.ele.id] = commit.original_oid

	commit_line := fmt.Sprintf("commit%s\n", commit.branch)
	mark_line := fmt.Sprintf("mark :%d\n", commit.ele.id)
	orig_id := fmt.Sprintf("original-oid %s\n", commit.original_oid)

	writer.Write([]byte(commit_line))
	writer.Write([]byte(mark_line))
	writer.Write([]byte(orig_id))

	if len(commit.author) != 0 {
		author_line := fmt.Sprintf("%s", commit.author)
		writer.Write([]byte(author_line))
	}
	if len(commit.commiter) != 0 {
		commiter_line := fmt.Sprintf("%s", commit.commiter)
		writer.Write([]byte(commiter_line))
	}

	size_line := fmt.Sprintf("data %d\n", commit.msg_size)
	data_line := fmt.Sprintf("%s", commit.message)
	writer.Write([]byte(size_line))
	writer.Write([]byte(data_line))

	if len(commit.parents) > 0 {
		from_line := fmt.Sprintf("from :%d\n", commit.parents[0])
		writer.Write([]byte(from_line))
	}
	if len(commit.parents) > 1 {
		for _, merge := range commit.parents[1:] {
			parent_line := fmt.Sprintf("merge :%d\n", merge)
			writer.Write([]byte(parent_line))
		}
	}
	// **NOTE** the filechanges here are multi-string line
	for _, filechange := range commit.filechanges {
		writer.Write([]byte(filechange.dumpToString()))
	}

	writer.Write([]byte("\n"))
}

func (commit *Commit) first_parent() int32 {
	if len(commit.parents) > 0 {
		return commit.parents[0]
	}
	return 0
}

func (commit *Commit) skip(new_id int32) {
	if commit.old_id != 0 {
		SKIPPED_COMMITS.Add(commit.old_id)
	} else {
		SKIPPED_COMMITS.Add(commit.ele.id)
	}
	commit.ele.skip(new_id)
}

/*
reset refs/heads/main
from :12
*/
type Reset struct {
	base *GitElements
	ref  string
	from int32
}

func NewReset(ref_ string, from_ref_ int32) Reset {
	base := NewGitElement()
	base.types = "reset"
	return Reset{
		base: &base,
		ref:  ref_,      // ref is string
		from: from_ref_, // but from_ref is short mark id, optional exist
	}
}

func (r *Reset) dump(writer io.WriteCloser) {
	r.base.dumped = true
	ref_line := fmt.Sprintf("reset%s\n", r.ref)
	writer.Write([]byte(ref_line))
	if r.from > 0 {
		from_ref_line := fmt.Sprintf("from :%d\n", r.from)
		writer.Write([]byte(from_ref_line))
		writer.Write([]byte("\n"))
	}
}

/*
tag v1.0.1
mark :13
from :12
original-oid 0e04e40bdf7cb956b36ed39b3063c253bd0d165c
tagger Li Linchao <lilinchao@oschina.cn> 1633941258 +0800
data 11
heavy tagg

tagger的格式为：<tagger_name> SP <tagger_email> SP <tagger_date>

tag 内容为数据块，大小固定(e.g. data 11，指定大小为11), 不包含LF
*/
type Tag struct {
	ele          *GitElementsWithID // mark_id, old_id, types, dumped
	old_id       int32              // mark_id too
	tag_name     string             // tag name(ref) line: tag v1.0.1, tag refs/heads/main
	from_ref     int32              // from :id line
	original_oid string
	tagger       string // tagger line
	msg_size     int32  // tager size is not as large as blob's
	msg          []byte // message line, raw bytes
}

func NewTag(tag_name_ string, from_ref_ int32, original_oid_, tagger_ string, size_ int32, msg_ []byte) Tag {
	ele := NewGitElementsWithID()
	ele.base.types = "tag"
	return Tag{
		ele:          &ele,
		old_id:       ele.id, // old_id = current mark id
		tag_name:     tag_name_,
		from_ref:     from_ref_,     // parent mark id
		original_oid: original_oid_, // sha-1 id
		tagger:       tagger_,
		msg_size:     size_,
		msg:          msg_,
	}
}

func (tag *Tag) dump(writer io.WriteCloser) {
	tag.ele.base.dumped = true
	HASH_ID[tag.original_oid] = tag.ele.id
	ID_HASH[tag.ele.id] = tag.original_oid

	tag_line := fmt.Sprintf("tag%s\n", tag.tag_name)
	mark_line := fmt.Sprintf("mark :%d\n", tag.ele.id)
	from_line := fmt.Sprintf("from :%d\n", tag.from_ref)
	origin_oid := fmt.Sprintf("original-oid %s\n", tag.original_oid)
	tagger_line := fmt.Sprintf("%s", tag.tagger)
	data_line := fmt.Sprintf("data %d\n%s\n", tag.msg_size, tag.msg)

	writer.Write([]byte(tag_line))
	writer.Write([]byte(mark_line))
	writer.Write([]byte(from_line))
	writer.Write([]byte(origin_oid))
	writer.Write([]byte(tagger_line))
	writer.Write([]byte(data_line))
}

// ref_line are like:
// commit refs/xxx/
// reset refs/xxx/
// tag xxx
// ref types are: commit, reset, tag
func parse_ref_line(reftype, line string) (refname string) {
	matches := Match(reftype+ref_re, line)
	// don't match
	if len(matches) == 0 {
		return ""
	}
	// return literal ref, not its type
	return matches[1]
}

// parent refs are like:
// from :parent_ref_id
// merge :parent_ref_id
// parent ref types are: from or merge
func parse_parent_ref(reftype, line string) (orig_ref, ref int32) {
	matches := Match(reftype+" :"+ref_re, line)

	// from 0000000000000000000000000000000000000000
	if len(line) == 46 {
		if line[5:len(line)-1] == "0000000000000000000000000000000000000000" {
			// mark to delete
			PrintLocalWithRedln("nested tags error")
			os.Exit(1)
		}
	}
	if len(matches) == 0 {
		// don't matched parent ref line
		return 0, 0
	}
	orig_baseref := matches[1]
	origref, _ := strconv.Atoi(orig_baseref)
	baseref := IDs.translate(int32(origref))
	// return ref mark id, not the whole line
	return int32(origref), baseref
}

func parse_mark(line string) (idx int32) {
	matches := Match("mark :"+idx_re, line)
	if len(matches) == 0 {
		PrintLocalWithRedln("no match mark id")
		return 0
	}
	if idx, err := strconv.Atoi(matches[1]); err == nil {
		return int32(idx)
	}
	return 0
}

func parse_original_oid(line string) (oid string) {
	matches := Match("original-oid "+oid_re, line)
	if len(matches) == 0 {
		PrintLocalWithRedln("no match original-oid")
		return ""
	}
	// single oid string
	return matches[1]
}

func parse_datasize(line string) int64 {
	matches := Match("data "+idx_re, line)
	if len(matches) == 0 {
		PrintLocalWithRedln("no match data size")
		return -1
	}
	size, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return -1
	}
	return size
}

// author, commiter, tagger
func parse_user(usertype, line string) (use string) {
	matches := Match(usertype+" "+user_re, line)
	if len(matches) == 0 {
		return ""
	}
	if len(matches[0]) != 0 {
		// return whole match line
		return matches[0]
	}
	return ""
}

// file mode can be: M(modify), D(delete), C(copy), R(rename), A(add)
// here we only handle M,D and R mode
func parse_filechange(line string) FileChange {
	arr := strings.Split(line, " ")
	types := arr[0]
	if types == "M" { // pattern: M mode :id path
		mode := arr[1]

		var parent_id string
		if strings.HasPrefix(arr[2], ":") {
			parent_id = strings.Split(arr[2], ":")[1]
			orig, _ := strconv.Atoi(parent_id)
			IDs.translate(int32(orig))
		} else { // pattern: M mode hash1-id path
			parent_id = arr[2]
		}
		path := strings.TrimSuffix(arr[3], "\n")

		filechange := NewFileChange("M", mode, parent_id, path)
		return filechange
	} else if types == "D" { // pattern: D path
		path := strings.TrimSuffix(arr[1], "\n")
		filechange := NewFileChange("D", "", "", path)
		return filechange
	} else if types == "R" { // pattern: R old new
		old_path := arr[1]
		new_path := strings.TrimSuffix(arr[2], "\n")
		filechange := NewFileChange("R", "", old_path, new_path)
		return filechange
	}

	return FileChange{}
}

// parse raw blob data, return data size, data and err
// **NOTE**
// blob data size maybe zero, so the parsed data matches maybe like:
// [data 0
//  0]
// thus we use -1 to indicate parse error
func (iter *FEOutPutIter) parse_data(line string, size int64) (n int64, data, extra_msg []byte) {

	var writer bytes.Buffer
	var sum int64
	newline := line
	// if data size is 0, no need to read any data more, just go to next line
	if size != 0 {

		for {
			n, err := writer.Write([]byte(newline))
			if err != nil {
				fmt.Println(err)
			}
			if n != len(newline) {
				PrintLocalWithRedln("failed to write data")
			}
			sum += int64(n)
			if sum == size {
				break
			}
			if sum > size {
				cur_linelen := int64(len(newline))
				extra_linelen := sum - size
				extra_msg = []byte(newline)[(cur_linelen - extra_linelen):]
				break
			}
			newline, _ = iter.Next()
		}
	}
	return sum, writer.Bytes(), extra_msg
}

func (iter *FEOutPutIter) parseBlob(line string) *Blob {
	// go to next line
	newline, _ := iter.Next()

	mark_id := parse_mark(newline)
	if mark_id == 0 {
		fmt.Println("DEBUG: parse blob error: mark id should > 0")
		return &Blob{}
	}

	newline, _ = iter.Next()
	original_oid := parse_original_oid(newline)
	if len(original_oid) == 0 {
		fmt.Println("DEBUG: parse blob error: original oid should not empty")
		return &Blob{}
	}

	newline, _ = iter.Next()
	size := parse_datasize(newline)

	newline, _ = iter.Next()
	actual_size, data_block, _ := iter.parse_data(newline, size)

	blob := NewBlob(actual_size, data_block, original_oid)

	if mark_id > 0 {
		blob.ele.old_id = mark_id
		IDs.record_rename(mark_id, blob.ele.id)
	}
	return &blob
}

func (iter *FEOutPutIter) parseCommit(line string) (*Commit, *Helper_info) {

	if line == "\n" {
		line, _ = iter.Next()
	}
	branch := parse_ref_line("commit", line)

	newline, _ := iter.Next()
	mark_id := parse_mark(newline)
	if mark_id == 0 {
		PrintLocalWithRedln("no match mark id")
		return &Commit{}, &Helper_info{}
	}

	newline, _ = iter.Next()
	original_oid := parse_original_oid(newline)

	newline, _ = iter.Next()
	author := parse_user("author", newline)

	newline, _ = iter.Next()
	commiter := parse_user("committer", newline)

	newline, _ = iter.Next()
	size := parse_datasize(newline)

	newline, _ = iter.Next()
	actual_size, msg, tail_msg := iter.parse_data(newline, size)

	// handle special case
	var actual_msg []byte
	var used bool
	orig_parents := make([]int32, 0)
	parents := make([]int32, 0)

	if actual_size > size {
		// treat this one as actual commit msg, the extra in the tail maybe filechange or parent(from), or just a LF
		/*
					Initial commit LF
			        M 100644 :1 LICENSE LF
			        M 100644 :1 README.md LF <--- tail-msg
		*/
		first_part := bytes.TrimRight(msg, string(tail_msg))
		actual_msg = append(first_part, '\n')

		if match := Match("from :"+ref_re, string(tail_msg)); len(match) > 0 {
			// get a from parent in extra_msg
			// must use parse_parent_ref() method to parse it, otherwise will get a dump error in some case.
			old_id, from_id := parse_parent_ref("from", string(tail_msg))
			orig_parents = append(orig_parents, old_id)
			parents = append(parents, from_id)
			used = true
		} else {
			used = false
		}
		msg = actual_msg
	}

	// next line maybe parents or filechanges
	newline, _ = iter.Next()

	// from parent
	if strings.HasPrefix(newline, "from") {
		old_id, from_id := parse_parent_ref("from", newline)
		orig_parents = append(orig_parents, old_id)
		parents = append(parents, from_id)
		newline, _ = iter.Next()
	}
	// merge parents
	for strings.HasPrefix(newline, "merge") {
		old_id, merge_id := parse_parent_ref("merge", newline)
		orig_parents = append(orig_parents, old_id)
		parents = append(parents, merge_id)
		newline, _ = iter.Next()
	}

	if n := len(orig_parents); n == 0 {
		if Lasted_commit[branch] > 0 {
			parents = []int32{Lasted_commit[branch]}
		}
	}
	if n := len(orig_parents); n == 0 {
		if Lasted_orig_commit[branch] > 0 {
			orig_parents = []int32{Lasted_orig_commit[branch]}
		}
	}

	// parse filechanges
	file_changes := make([]FileChange, 0)
	var filechange FileChange

	// if extra_msg is not empty and haven't been used, treat it as filechange
	if len(tail_msg) > 1 && !used {
		filechange = parse_filechange(string(tail_msg))
		file_changes = append(file_changes, filechange)
	}
	for newline != "\n" {
		filechange = parse_filechange(newline)
		file_changes = append(file_changes, filechange)
		newline, _ = iter.Next()
	}

	commit := NewCommit(original_oid, branch, author, commiter, int32(len(msg)),
		msg, parents, file_changes)

	if mark_id > 0 {
		commit.old_id = mark_id
		IDs.record_rename(mark_id, commit.ele.id)
	}

	hinfo := &Helper_info{
		orig_parents:   orig_parents,
		has_filechange: len(commit.filechanges) != 0,
	}

	return &commit, hinfo
}

func (iter *FEOutPutIter) parseReset(line string) *Reset {
	ref := parse_ref_line("reset", line)
	// this reset is the first reset on the first commit
	str, _ := iter.f.Peek(6)
	if string(str) == "commit" {
		reset := NewReset(ref, 0)
		return &reset
	}
	// then countinue to parse from-line in reset structure
	newline, _ := iter.Next()
	_, parent_id := parse_parent_ref("from", newline)

	if parent_id <= 0 {
		delete(Lasted_commit, ref)
		delete(Lasted_orig_commit, ref)
	}
	reset := NewReset(ref, parent_id)

	Lasted_commit[reset.ref] = reset.from
	Lasted_orig_commit[reset.ref] = reset.from

	return &reset
}

func (iter *FEOutPutIter) parseTag(line string) *Tag {
	tag_name := parse_ref_line("tag", line)

	// go to next new line
	newline, _ := iter.Next()
	mark_id := parse_mark(newline)
	if mark_id == 0 {
		PrintLocalWithRedln("no match mark id")
	}
	newline, _ = iter.Next()
	_, parent_id := parse_parent_ref("from", newline)

	newline, _ = iter.Next()
	original_oid := parse_original_oid(newline)

	newline, _ = iter.Next()
	tagger := parse_user("tagger", newline)

	newline, _ = iter.Next()
	size := parse_datasize(newline)

	newline, _ = iter.Next()
	actual_size, msg, _ := iter.parse_data(newline, size)

	tag := NewTag(tag_name, parent_id, original_oid, tagger, int32(actual_size), msg)

	// the parsed mark id from original source data is old id,
	// cause the new id is generated by IDs.New() in NewTag()
	if mark_id > 0 {
		// new_id = tag.ele.id
		tag.old_id = mark_id
		// map[old_id] = to new_id
		IDs.record_rename(mark_id, tag.ele.id)
	} else {
		tag.ele.skip(0)
	}
	return &tag
}

func (filter *RepoFilter) Parser() {
	if filter.repo.opts.verbose {
		PrintLocalWithGreenln("start to clean up specified files")
	}

	iter, err := filter.repo.NewFastExportIter()
	defer iter.Close()
	if err != nil {
		fmt.Fprint(os.Stdout, err)
	}

	input, cmd, err := filter.repo.FastImportOut()
	defer func() {
		input.Close()
		cmd.Wait()
	}()
	if err != nil {
		PrintLocalWithRedln("run git-fast-import process failed")
	}

	for {
		line, _ := iter.Next()
		if matches := Match("feature done\n", line); len(matches) != 0 {
			continue
		} else if matches := Match("blob\n", line); len(matches) != 0 {
			blob := iter.parseBlob(line)
			filter.tweak_blob(blob)

			if blob.ele.base.dumped {
				blob.dump(input)
			}

		} else if matches := Match("commit (.*)\n$", line); len(matches) != 0 {
			commit, aux_info := iter.parseCommit(line)
			filter.tweak_commit(commit, aux_info)

			if commit.ele.base.dumped {
				commit.dump(input)
			}

		} else if matches := Match("reset (.*)\n$", line); len(matches) != 0 {
			reset := iter.parseReset(line)

			filter.tweak_reset(reset)

			if reset.base.dumped {
				reset.dump(input)
			}
		} else if matches := Match("tag (.*)\n$", line); len(matches) != 0 {
			tag := iter.parseTag(line)

			filter.tweak_tag(tag)

			if tag.ele.base.dumped {
				tag.dump(input)
			}
		} else if strings.HasPrefix(line, "done\n") {
			iter.Close()
			break
		}
	}
}
