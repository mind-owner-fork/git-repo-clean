package main

// type Filter struct {
// 	parse_blob_   func()
// 	parse_reset_  func()
// 	parse_commit_ func()
// 	parse_tag_    func()
// }

/*Ids*/
type Ids struct {
	next_id      int32
	translations map[int32]int32
}

// create Ids object
func (ids Ids) NewIDs() Ids {
	return Ids{
		next_id:      1,
		translations: make(map[int32]int32),
	}
}

// return current next_id, then next_id + 1
func (ids Ids) nextID() int32 {
	id := ids.next_id
	ids.next_id += 1 // need atomaic operation?
	return id
}

// record map: old_id => new_id
func (ids Ids) record_rename(old_id, new_id int32) {
	if old_id != new_id {
		ids.translations[old_id] = new_id
	}
}

func (ids Ids) has_renames() bool {
	return len(ids.translations) == 0
}

// query from translations map, if find return new_id, else return old_id
func (ids Ids) translate(old_id int32) int32 {
	if new_id, ok := ids.translations[old_id]; ok {
		return new_id
	} else {
		return old_id
	}
}

//  Global OID and tables
var (
	IDs            = Ids{}.NewIDs()
	Id_hash        map[int32]string
	Hash_id        map[string]int32
	Skiped_commits []int32
)

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
	base   GitElements
	id     int32 // mark id
	old_id int32 // previous mark id
}

// new Git element has new mark id, and its previous id is 0 as default,
// but will set properly on the other place
func NewGitElementsWithID() GitElementsWithID {
	return GitElementsWithID{
		base:   NewGitElement(),
		id:     IDs.nextID(),
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
