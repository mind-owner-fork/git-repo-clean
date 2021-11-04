package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cli/safeexec"
	mapset "github.com/deckarep/golang-set"
)

//  Global OID and ID tables
var (
	IDs             = NewIDs()
	ID_HASH         = make(map[int32]string)
	HASH_ID         = make(map[string]int32)
	SKIPPED_COMMITS = mapset.NewSet()
)

/*Ids*/
type Ids struct {
	next_id      int32
	translations map[int32]int32
}

// create Ids object instance
func NewIDs() Ids {
	return Ids{
		next_id:      1,
		translations: make(map[int32]int32),
	}
}

// return current next_id, then next_id + 1
func (ids *Ids) New() int32 {
	id := ids.next_id
	ids.next_id += 1
	return id
}

// record map: old_id => new_id
func (ids *Ids) record_rename(old_id, new_id int32) {
	if old_id != new_id {
		ids.translations[old_id] = new_id
	}
}

func (ids Ids) has_renames() bool {
	return len(ids.translations) == 0
}

// query from translations map, if find return new_id, else return old_id
func (ids *Ids) translate(old_id int32) int32 {
	if new_id, ok := ids.translations[old_id]; ok {
		return new_id
	} else {
		return old_id
	}
}

// Git element basically contain type and dumped field
type GitElements struct {
	types  string
	dumped bool
}

// return element types and dump status
func NewGitElement() GitElements {
	return GitElements{
		types:  "none",
		dumped: true, // true means to dump out, which is the default behavior
	}
}

func (ele GitElements) skip() {
	ele.dumped = false // false means to skip it
}

// base represents type and dumped,
// id represents int32 short mark id,
// old_id represents previous short mark id
type GitElementsWithID struct {
	base   *GitElements
	id     int32 // mark id
	old_id int32 // previous mark id
}

// new Git element has new mark id, and its previous id is 0 as default,
// but will set properly on the other place
func NewGitElementsWithID() GitElementsWithID {
	ele := NewGitElement()
	return GitElementsWithID{
		base:   &ele,
		id:     IDs.New(),
		old_id: 0, // mark id must > 0, so 0 just means it haven't initialized
	}
}

func (ele GitElementsWithID) skip(new_id int32) {
	ele.base.dumped = false
	if ele.old_id != 0 {
		IDs.record_rename(ele.old_id, new_id)
	} else {
		IDs.record_rename(ele.id, new_id)
	}
}

func findGitBin() (string, error) {
	gitBin, err := safeexec.LookPath("git")
	if err != nil {
		return "", err
	}

	gitBin, err = filepath.Abs(gitBin)
	if err != nil {
		return "", err
	}

	return gitBin, nil
}

func (repo Repository) GitCommand(callerArgs ...string) *exec.Cmd {
	args := []string{
		"--no-replace-objects",
		"-c",
		"advice.graftFileDeprecated=false",
		"-C",
		repo.path,
	}

	args = append(args, callerArgs...)

	cmd := exec.Command(repo.gitBin, args...)
	cmd.Env = append(
		os.Environ(),
		// "GIT_DIR"+repo.gitDir, // fix Windows issue
		// Disable grafts when running our commands:
		"GIT_GRAFT_FILE="+os.DevNull,
	)

	return cmd
}

// CanonicalizePath returns absolute repo path
func CanonicalizePath(path, relPath string) string {
	if filepath.IsAbs(relPath) {
		return relPath
	}
	return filepath.Join(path, relPath)
}

func GitDir(gitbin, path string) (string, error) {

	cmd := exec.Command(gitbin, "-C", path, "rev-parse", "--git-dir")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf(
			"could not run 'git rev-parse --git-dir': %s", err,
		)
	}
	// git object dir: ${repo-path}/${git-dir}
	gitDir := CanonicalizePath(path, string(bytes.TrimSpace(out)))

	return gitDir, nil
}
