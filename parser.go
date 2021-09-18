package main

import (
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
	ref_re  = `(.*)\n$`               // commit|reset|ref refs/tags/v1.0.0
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
	data_line := fmt.Sprintf("data %d\n", blob.data_size)

	writer.Write([]byte("blob\n"))
	writer.Write([]byte(mark_line))
	writer.Write([]byte(oid_line))
	writer.Write([]byte(data_line))

	// #TODO write certain size data
	// writer.Write([]byte(blob.mark_id_))
	writer.Write([]byte("\n"))
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

func NewCommit(original_oid_, branch_, author_, commiter_ string, size_ int32, msg_ []byte, parents_ []int32, filechanges_ []string) Commit {
	var ele = NewGitElementsWithID()
	ele.base.types = "commit"

	return Commit{
		ele:          ele,
		old_id:       ele.id,
		original_oid: original_oid_,
		branch:       branch_,
		author:       author_,
		commiter:     commiter_,
		msg_size:     size_,
		message:      msg_,
		from:         parents_[0],
		merges:       parents_[1:],
		filechanges:  filechanges_,
	}
}

func (commit *Commit) dump(writer io.WriteCloser) {

	commit.ele.base.dumped = true

	Hash_id[commit.original_oid] = commit.ele.id
	Id_hash[commit.ele.id] = commit.original_oid

	if commit.from == 0 || len(commit.merges) == 0 {
		reset_line := fmt.Sprintf("reset %s\n", commit.branch)
		writer.Write([]byte(reset_line))
	}
	commit_line := fmt.Sprintf("commit %s\n", commit.branch)
	mark_line := fmt.Sprintf("mark :%d\n", commit.ele.id)
	orig_id := fmt.Sprintf("original-oid %s\n", commit.original_oid)

	writer.Write([]byte(commit_line))
	writer.Write([]byte(mark_line))
	writer.Write([]byte(orig_id))

	if len(commit.author) != 0 {
		author_line := fmt.Sprintf("author %s\n", commit.author)
		writer.Write([]byte(author_line))
	}
	if len(commit.commiter) != 0 {
		commiter_line := fmt.Sprintf("commiter %s\n", commit.commiter)
		writer.Write([]byte(commiter_line))
	}
	size_line := fmt.Sprintf("data %d\n", commit.msg_size)
	data_line := fmt.Sprintf("%s\n", commit.message)
	from_line := fmt.Sprintf("from :%d\n", commit.from)
	writer.Write([]byte(size_line))
	writer.Write([]byte(data_line))
	writer.Write([]byte(from_line))

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
		from: from_ref_, // but from_ref is short id
	}
}

func (r *Reset) dump(writer io.WriteCloser) {
	r.base.dumped = true
	ref_line := fmt.Sprintf("reset %s\n", r.ref)
	from_ref_line := fmt.Sprintf("from :%d\n", r.from)

	writer.Write([]byte(ref_line))
	writer.Write([]byte(from_ref_line))

	writer.Write([]byte("\n"))
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

	tag_line := fmt.Sprintf("tag %s\n", tag.ref)
	mark_line := fmt.Sprintf("mark :%d\n", tag.ele.id)
	from_line := fmt.Sprintf("from :%d\n", tag.from_ref)
	tagger_line := fmt.Sprintf("%s\n", tag.tagger)
	size_line := fmt.Sprintf("data %d\n", tag.data_size)

	writer.Write([]byte(tag_line))
	writer.Write([]byte(mark_line))
	writer.Write([]byte(from_line))
	writer.Write([]byte(tagger_line))
	writer.Write([]byte(size_line))

	// #TODO write certain size data

	writer.Write([]byte("\n"))
}

// ref_line are like:
// commit refs/xxx/
// reset refs/xxx/
// tag xxx
// ref types are: commit, reset, tag
func (iter *FEOutPutIter) parse_ref_line(reftype, line string) (ref, newline string) {
	matches := Match(reftype+ref_re, line)
	// go to next
	new_line, _ := iter.Next()
	// return literal ref, not contain type field
	return matches[1], new_line
}

// parent refs are like:
// from :parent_ref_id
// merge :parent_ref_id
// parent ref types are: from or merge

func (iter *FEOutPutIter) parse_parent_ref(reftype, line string) (ref, newline string) {
	matches := Match(reftype+" :"+ref_re, line)
	orig_baseref := matches[1]
	ref_id, _ := strconv.Atoi(orig_baseref)
	baseref := IDs.translate(int32(ref_id))
	// return ref id, not the whole line
	return strconv.Itoa(int(baseref)), newline
}

func (iter *FEOutPutIter) parse_mark(line string) (idx int32, newline string) {
	matches := Match("mark :"+idx_re, line)
	if len(matches) <= 0 {
		fmt.Println("no match mark id")
		return 0, ""
	}
	if idx, err := strconv.Atoi(matches[1]); err == nil {
		// go to next
		new_line, _ := iter.Next()
		return int32(idx), new_line
	}
	return 0, ""
}

func (iter *FEOutPutIter) parse_original_id(line string) (oid string, newline string) {
	matches := Match("original-oid "+oid_re, line)
	if len(matches) == 0 {
		fmt.Println("no match original-oid")
		return "", ""
	}
	// go to next
	new_line, _ := iter.Next()
	// single oid string
	return matches[1], new_line
}

// parse data size, return data size
// **NOTE**
// blob data size maybe zero!, so the parsed data matches maybe like:
// [data 0
//  0]

// parse data block too
func (iter *FEOutPutIter) parse_data(line string) (size int64, newline string) {
	matches := Match("data "+idx_re, line)
	fmt.Printf("parsed data matches: %s\n", matches)
	if len(matches) == 0 {
		fmt.Println("no match data")
		return -1, ""
	}
	if size, err := strconv.Atoi(matches[1]); err == nil {
		// go to next
		new_line, _ := iter.Next()
		return int64(size), new_line
	}
	return -1, ""
}

// author, commiter, tagger
func (iter *FEOutPutIter) parse_user(usertype, line string) (use string, newline string) {

	if matches := Match(usertype+" "+user_re, line); len(matches[0]) != 0 {
		// go to next
		new_line, _ := iter.Next()
		// return whole line
		return matches[0], new_line
	}
	return "", ""
}

// file mode can be: M(modify), D(delete), C(copy), R(rename), A(add)
// here we only handle A, M and D mode
// TODO: file path
func (iter *FEOutPutIter) parse_filechange(line string) (filechange, newline string) {
	arr := strings.Split(line, " ")
	flag := arr[0]
	// mode := arr[1]
	id_s := arr[2] // :18
	path := arr[3]

	if flag == "M" {
		id_s = strings.Split(id_s, ":")[1] // 18
		id_t, _ := strconv.Atoi(id_s)
		IDs.translate(int32(id_t))
		if strings.HasPrefix(path, "\"") {
			// dequote path
		}
		// type FileChange struct
	} else if flag == "A" {

	} else if flag == "D" {

	}

	return "", ""
}

func (iter *FEOutPutIter) parseBlob(op Options, line string) Blob {
	// go to next
	newline, _ := iter.Next()
	id, newline := iter.parse_mark(newline)
	fmt.Printf("parsed mark id: %d\n", id)
	if id == 0 {
		// throw err info, then exit
		return Blob{}
	}
	original_oid, newline := iter.parse_original_id(newline)
	fmt.Printf("parsed original oid: %s\n", original_oid)
	if len(original_oid) == 0 {
		// throw err info, then exit
		return Blob{}
	}

	size, newline := iter.parse_data(newline)
	fmt.Printf("parsed data size: %d\n", size)
	data_block := make([]byte, size)

	fmt.Printf("read size: %d\n", len(data_block))
	// io.ReadFull(iter.out, data_block)

	if size != 0 {
		// parse size of raw blob data
		for {
			line_byte, err := iter.f.ReadBytes('\n')
			if err != nil {
				if err != io.EOF {
					// normal exit
					break
				}
				// parse wrong
				break
			}

			// if we want to end the blob data read, only its size can be a flag
			if line_byte[len(line_byte)-1] == '\n' && len(line_byte) == 1 {
				break
			}

			fmt.Printf("read line : %s\n", line_byte)
			data_block = append(data_block, line_byte...)
			fmt.Printf("copy to data_block : %s\n", line_byte)
		}
	}
	fmt.Printf("read total : %s\n", data_block)
	newline, _ = iter.Next()
	// Parse and construct a complete Blob object
	blob := NewBlob(size, data_block, original_oid)

	// decide whether to drop this blob
	limit, _ := UnitConvert(op.limit)
	if size > int64(limit) {
		blob.ele.base.dumped = false
		fmt.Println("will drop this blob")
		return Blob{}
	}
	return blob
}

func (iter *FEOutPutIter) parseCommit(line string) Commit {
	return Commit{}
}

func (iter *FEOutPutIter) parseReset(line string) Reset {
	return Reset{}
}

func (iter *FEOutPutIter) parseTag(line string) Tag {
	return Tag{}
}

// pass some parameters into this for blob filter?
func (repo *Repository) Parser(opts Options) {
	iter, err := repo.NewFastExportIter()
	if err != nil {
		fmt.Fprint(os.Stdout, err)
	}
	for {
		line, _ := iter.Next()

		if strings.HasPrefix(line, "feature done") {
			// go to next line
			continue
		} else if strings.HasPrefix(line, "blob") {
			// pass user options to filter blob
			iter.parseBlob(opts, line)
		} else if strings.HasPrefix(line, "reset") {
			iter.parseReset(line)
		} else if strings.HasPrefix(line, "commit") {
			iter.parseCommit(line)
		} else if strings.HasPrefix(line, "tag") {
			iter.parseTag(line)
		} else if strings.HasPrefix(line, "done") {
			// parse done
			iter.Close()
			break
		}
		// continue next line
	}

}
