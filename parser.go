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
	ele          GitElementsWithID // contain: id, old_id, types, dumped
	original_oid string            // 40 bytes
	data_size    int64             // blob size maybe very large
	data         []byte            // raw data block
}

func NewBlob(size_ int64, data_ []byte, hash_id_ string) Blob {
	var ele = NewGitElementsWithID()
	ele.base.types = "blob"
	return Blob{
		ele:          ele,
		original_oid: hash_id_,
		data_size:    size_,
		data:         data_,
	}
}

func (blob Blob) dump(writer io.WriteCloser) {
	blob.ele.base.dumped = true
	Hash_id[blob.original_oid] = blob.ele.id
	Id_hash[blob.ele.id] = blob.original_oid

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
	base       GitElements
	changetype string
	mode       string
	blob_id    int32
	filepath   string
}

// **NOTE**
// when type is "M", mode and id must not nil, when type is "D", mode and id is nil
//
// filechange usually have multi-line
func NewFileChange(types_, mode_ string, id_ int32, filepath_ string) FileChange {
	var base = NewGitElement()
	base.types = "filechange"
	return FileChange{
		base:       base,
		changetype: types_,
		mode:       mode_,
		blob_id:    id_,
		filepath:   filepath_,
	}
}

func (fc *FileChange) dumpToString() string {
	if fc.changetype == "M" {
		filechange_ := fmt.Sprintf("M %s :%d %s\n", fc.mode, fc.blob_id, fc.filepath)
		return filechange_
	} else if fc.changetype == "D" {
		filechange_ := fmt.Sprintf("D %s\n", fc.filepath)
		return filechange_
	} else {
		return ""
	}
}

func (fc *FileChange) dump(writer io.WriteCloser) {
	fc.base.dumped = true
	// currently only consider M type "M 100644 :18 files/1.c" and D type "D files/1.c"
	// when the type is M, and the id is short mark id
	if fc.changetype == "M" {
		filechange_ := fmt.Sprintf("M %s :%d %s\n", fc.mode, fc.blob_id, fc.filepath)
		writer.Write([]byte(filechange_))
	} else if fc.changetype == "D" {
		filechange_ := fmt.Sprintf("D %s\n", fc.filepath)
		writer.Write([]byte(filechange_))
	} else {
		// unhandle filechange type
		fmt.Println("unsupported filechange type")
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


**NOTE**

a). When a commit has no parent(usually the first commit), then there will be a reset command in the front of commit command.

c). When merge multi branches into one, a commit have multi parents, then there will be at least one merge commands.

c). When a commit has parent commit, then it has from or merge(or both), otherwise, have none of them.
*/
type Commit struct {
	ele          GitElementsWithID // mark id
	old_id       int32             // previous mark id, given by GitElementsWithID
	original_oid string
	branch       string
	author       string
	commiter     string
	msg_size     int32
	message      []byte   // commit message
	from         int32    // parent mark id, exist when a commit have single-parent
	merges       []int32  // merge mark id, exist when a commit have multi-parents
	filechanges  []string // multi-line
}

func parent_filter(str []int32) (int32, []int32) {
	var empty []int32
	if len(str) == 0 {
		return 0, empty
	} else {
		return str[0], str[1:]
	}
}

func NewCommit(original_oid_, branch_, author_, commiter_ string, size_ int32, msg_ []byte, parents_ []int32, filechanges_ []string) Commit {
	var ele = NewGitElementsWithID()
	ele.base.types = "commit"
	from, merges := parent_filter(parents_)
	return Commit{
		ele:          ele,
		old_id:       ele.id,
		original_oid: original_oid_,
		branch:       branch_,
		author:       author_,
		commiter:     commiter_,
		msg_size:     size_,
		message:      msg_,
		from:         from,
		merges:       merges,
		filechanges:  filechanges_,
	}
}

func (commit *Commit) dump(writer io.WriteCloser) {

	commit.ele.base.dumped = true

	Hash_id[commit.original_oid] = commit.ele.id
	Id_hash[commit.ele.id] = commit.original_oid

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

	if commit.from > 0 {
		from_line := fmt.Sprintf("from :%d\n", commit.from)
		writer.Write([]byte(from_line))
	}

	for _, parent := range commit.merges {
		parent_line := fmt.Sprintf("merge :%d\n", parent)
		writer.Write([]byte(parent_line))
	}
	// **NOTE** the filechanges here are multi-string line
	for _, filechange := range commit.filechanges {
		writer.Write([]byte(filechange))
	}

	writer.Write([]byte("\n"))
}

func (commit *Commit) first_parent() int32 {
	return commit.from
}

func (commit *Commit) skip(new_id int32) {
	if commit.old_id != 0 {
		Skiped_commits = append(Skiped_commits, commit.old_id)
	} else {
		Skiped_commits = append(Skiped_commits, commit.ele.id)
	}
	commit.ele.skip(new_id)
}

/*
reset refs/heads/main
from :12
*/
type Reset struct {
	base GitElements
	ref  string
	from int32
}

func NewReset(ref_ string, from_ref_ int32) Reset {
	base := NewGitElement()
	base.types = "reset"
	return Reset{
		base: base,
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
	ele          GitElementsWithID // mark_id, old_id, types, dumped
	old_id       int32             // mark_id too
	ref          string            // tag name(ref) line: tag v1.0.1, tag refs/heads/main
	from_ref     int32             // from :id line
	original_oid string
	tagger       string // tagger line
	data_size    int32  // tager size is not as large as blob's
	msg          []byte // message line, raw bytes
}

func NewTag(ref_ string, from_ref_ int32, original_oid_, tagger_ string, size_ int32, msg_ []byte) Tag {
	ele := NewGitElementsWithID()
	ele.base.types = "tag"
	return Tag{
		ele:          ele,
		old_id:       ele.id,
		ref:          ref_,
		from_ref:     from_ref_,
		original_oid: original_oid_,
		tagger:       tagger_,
		data_size:    size_,
		msg:          msg_,
	}
}

func (tag *Tag) dump(writer io.WriteCloser) {
	tag.ele.base.dumped = true
	Hash_id[tag.original_oid] = tag.ele.id
	Id_hash[tag.ele.id] = tag.original_oid

	tag_line := fmt.Sprintf("tag%s\n", tag.ref)
	mark_line := fmt.Sprintf("mark :%d\n", tag.ele.id)
	from_line := fmt.Sprintf("from :%d\n", tag.from_ref)
	origin_oid := fmt.Sprintf("original-oid %s\n", tag.original_oid)
	tagger_line := fmt.Sprintf("%s", tag.tagger)
	data_line := fmt.Sprintf("data %d\n%s\n", tag.data_size, tag.msg)

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
func (iter *FEOutPutIter) parse_ref_line(reftype, line string) (refname string) {
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
func (iter *FEOutPutIter) parse_parent_ref(reftype, line string) (refid int32) {
	matches := Match(reftype+" :"+ref_re, line)
	if len(matches) == 0 {
		// don't matched parent ref line
		return 0
	}
	orig_baseref := matches[1]
	ref_id, _ := strconv.Atoi(orig_baseref)
	baseref := IDs.translate(int32(ref_id))
	// return ref mark id, not the whole line
	return baseref
}

func (iter *FEOutPutIter) parse_mark(line string) (idx int32) {
	matches := Match("mark :"+idx_re, line)
	if len(matches) == 0 {
		fmt.Println("no match mark id")
		return 0
	}
	if idx, err := strconv.Atoi(matches[1]); err == nil {
		return int32(idx)
	}
	return 0
}

func (iter *FEOutPutIter) parse_original_id(line string) (oid string) {
	matches := Match("original-oid "+oid_re, line)
	if len(matches) == 0 {
		fmt.Println("no match original-oid")
		return ""
	}
	// single oid string
	return matches[1]
}

// parse raw blob data, return data size, data and err
// **NOTE**
// blob data size maybe zero, so the parsed data matches maybe like:
// [data 0
//  0]
// thus we use -1 to indicate parse error
func (iter *FEOutPutIter) parse_data(line string) (n int64, data []byte) {
	blank_data := make([]byte, 0)
	var writer bytes.Buffer

	matches := Match("data "+idx_re, line)
	if len(matches) == 0 {
		fmt.Println("no match data size")
		return -1, blank_data
	}
	size, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return -1, blank_data
	}

	// go to next line
	newline, _ := iter.Next()

	// if data size is 0, no need to read any data more, just go to next line
	if size != 0 {
		var sum int64
		for {
			n, err := writer.Write([]byte(newline))
			if err != nil {
				fmt.Println(err)
			}
			if n != len(newline) {
				fmt.Println("failed to write data")
			}
			sum += int64(n)
			if sum >= size {
				break
			}
			newline, _ = iter.Next()
		}
	}
	return size, writer.Bytes()
}

// author, commiter, tagger
func (iter *FEOutPutIter) parse_user(usertype, line string) (use string) {
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
// #FIXME: fix file path format in different OS platform
func (iter *FEOutPutIter) parse_filechange(line string) FileChange {
	arr := strings.Split(line, " ")
	types := arr[0]
	if types == "M" { // pattern: M mode :id path
		mode := arr[1]
		parent_id, _ := strconv.ParseInt(strings.Split(arr[2], ":")[1], 10, 32)
		path := strings.TrimSuffix(arr[3], "\n")
		IDs.translate(int32(parent_id))

		filechange := NewFileChange("M", mode, int32(parent_id), path)
		return filechange
	} else if types == "D" { // pattern: D path
		path := strings.TrimSuffix(arr[1], "\n")
		filechange := NewFileChange("D", "", 0, path)
		return filechange
	} else if types == "R" { // pattern: R old new
		original_path := arr[1]
		// ???
		// new_path := strings.TrimSuffix(arr[2], "\n")
		filechange := NewFileChange("R", "", 0, original_path)
		return filechange
	}

	return FileChange{}
}

func (iter *FEOutPutIter) parseBlob(op Options, line string) Blob {
	// go to next line
	newline, _ := iter.Next()

	mark_id := iter.parse_mark(newline)
	if mark_id == 0 {
		// #FIXME throw err info, then exit
		return Blob{}
	}

	newline, _ = iter.Next()
	original_oid := iter.parse_original_id(newline)
	if len(original_oid) == 0 {
		// #FIXME throw err info, then exit
		return Blob{}
	}

	newline, _ = iter.Next()
	size, data_block := iter.parse_data(newline)

	blob := NewBlob(size, data_block, original_oid)

	// decide whether to drop this blob
	// dumped == false, means will not dump into pipe
	limit, _ := UnitConvert(op.limit)
	if size > int64(limit) {
		blob.ele.base.dumped = false
		// fmt.Println("will drop this blob")
	}

	if mark_id > 0 {
		blob.ele.old_id = mark_id
		IDs.record_rename(mark_id, blob.ele.id)
	}
	return blob
}

func (iter *FEOutPutIter) parseCommit(line string) Commit {
	var merge_id int32
	parent_ids := make([]int32, 0)

	if line == "\n" {
		line, _ = iter.Next()
	}
	branch := iter.parse_ref_line("commit", line)

	newline, _ := iter.Next()
	mark_id := iter.parse_mark(newline)
	if mark_id == 0 {
		// #FIXME throw err info, then exit
		return Commit{}
	}

	newline, _ = iter.Next()
	orign_oid := iter.parse_original_id(newline)

	newline, _ = iter.Next()
	author := iter.parse_user("author", newline)

	newline, _ = iter.Next()
	commiter := iter.parse_user("committer", newline)

	newline, _ = iter.Next()
	size, msg := iter.parse_data(newline)

	newline, _ = iter.Next()
	if strings.HasPrefix(newline, "from") {
		from_id := iter.parse_parent_ref("from", newline)
		parent_ids = append(parent_ids, from_id)
	}

	for strings.HasPrefix(newline, "merge") {
		merge_id = iter.parse_parent_ref("merge", newline)
		parent_ids = append(parent_ids, merge_id)
		newline, _ = iter.Next()
	}

	file_changes := make([]string, 0)
	var filechange FileChange
	for newline != "\n" {
		filechange = iter.parse_filechange(newline)
		file_changes = append(file_changes, filechange.dumpToString())
		newline, _ = iter.Next()
	}
	commit := NewCommit(orign_oid, branch, author, commiter, int32(size),
		msg, parent_ids, file_changes)

	if mark_id > 0 {
		commit.old_id = mark_id
		IDs.record_rename(mark_id, commit.ele.id)
	}
	// #TODO dump Commit into fast-import
	if commit.ele.base.dumped {
		// commit.dump(writer)
	}
	return commit
}

func (iter *FEOutPutIter) parseReset(line string) Reset {
	ref := iter.parse_ref_line("reset", line)
	// this reset is the first reset on the first commit
	str, _ := iter.f.Peek(6)
	if string(str) == "commit" {
		return NewReset(ref, 0)
	}
	// then countinue to parse from-line in reset structure
	newline, _ := iter.Next()
	parent_id := iter.parse_parent_ref("from", newline)

	reset := NewReset(ref, parent_id)
	// #TODO dump Reset into git-fast-import
	if reset.base.dumped {
		// reset.dump(writer)
	}
	return reset
}

func (iter *FEOutPutIter) parseTag(line string) Tag {
	tag_name := iter.parse_ref_line("tag", line)

	// go to next new line
	newline, _ := iter.Next()
	mark_id := iter.parse_mark(newline)
	if mark_id == 0 {
		// #FIXME throw err info, then exit
	}
	newline, _ = iter.Next()
	parent_id := iter.parse_parent_ref("from", newline)

	newline, _ = iter.Next()
	orig_id := iter.parse_original_id(newline)

	newline, _ = iter.Next()
	tagger := iter.parse_user("tagger", newline)

	newline, _ = iter.Next()
	size, msg := iter.parse_data(newline)

	tag := NewTag(tag_name, parent_id, orig_id, tagger, int32(size), msg)
	if mark_id > 0 {
		// the current parsed mark id is old id
		// the new id is generated by IDs.nextID() in NewTag()
		tag.old_id = mark_id
		IDs.record_rename(mark_id, tag.ele.id)
	}

	return tag
}

// pass some parameters into this for blob filter?
func (repo *Repository) Parser(opts Options) {
	var iter *FEOutPutIter
	var input io.WriteCloser
	// var input *os.File
	var err error

	iter, err = repo.NewFastExportIter()
	if err != nil {
		fmt.Fprint(os.Stdout, err)
	}

	go func() {
		input, err = repo.FastImportOut()
		if err != nil {
			fmt.Println("run git-fast-import process failed")
		}
		// this is for test
		// input, err = os.Create(".git/fast-export.out")
	}()

	for {
		line, _ := iter.Next()
		if matches := Match("feature done\n", line); len(matches) != 0 {
			continue
		} else if matches := Match("blob\n", line); len(matches) != 0 {
			// pass user-end options to filter blob
			blob := iter.parseBlob(opts, line)
			blob.dump(input)
		} else if matches := Match("reset (.*)\n$", line); len(matches) != 0 {
			reset := iter.parseReset(line)
			reset.dump(input)
		} else if matches := Match("commit (.*)\n$", line); len(matches) != 0 {
			commit := iter.parseCommit(line)
			commit.dump(input)
		} else if matches := Match("tag (.*)\n$", line); len(matches) != 0 {
			tag := iter.parseTag(line)
			tag.dump(input)
		} else if strings.HasPrefix(line, "done\n") && strings.HasPrefix(line, "done\n") {
			iter.Close()
			break
		}
	}

}
