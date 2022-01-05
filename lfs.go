package main

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

var (
	LFSVER = "https://git-lfs.github.com/spec/v1"
)

type Pointer struct {
	Version string
	Oid     string // sha256
	Size    int64
}

func NewLFSPointer(blob *Blob) Pointer {
	p := Pointer{
		Version: LFSVER,
		Oid:     blob.sha256,
		Size:    blob.data_size,
	}
	return p
}

func GenerateHash(data []byte, alg string) (hash string) {
	if alg == "sha256sum" {
		sha256 := sha256.New()
		sha256.Write(data)
		return hex.EncodeToString(sha256.Sum(nil))
	}
	if alg == "sha1sum" {
		sha1 := sha1.New()
		sha1.Write(data)
		return hex.EncodeToString(sha1.Sum(nil))
	}
	return
}

// version https://git-lfs.github.com/spec/v1
// oid sha256:$(sha256)
// size $(old size)
func CreatePointerFile(blob *Blob) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "version %s\n", LFSVER)
	fmt.Fprintf(&buf, "oid sha256:%s\n", blob.sha256)
	fmt.Fprintf(&buf, "size %d\n", len(blob.data))
	if buf.Len() > 200 {
		fmt.Println("bad LFS Pointer file")
		return nil
	}
	return buf.Bytes()
}

func UpdateBlob(blob *Blob) *Blob {
	pf := CreatePointerFile(blob)
	newblob := blob
	newblob.original_oid = GenerateHash(pf, "sha1sum")
	newblob.data_size = int64(len(pf))
	newblob.data = pf

	return newblob
}

// convert to Git LFS object
func ConvertToLFSObj(blob *Blob) {
	f := OpenLFSFile(blob)
	defer f.Close()
	n, err := f.Write(blob.data)
	if err != nil {
		fmt.Printf("write data into %s error: %s", f.Name(), err)
	}
	if n != int(blob.data_size) {
		fmt.Println("write error")
	}
}

func OpenLFSFile(blob *Blob) (f *os.File) {
	dir := CreateLFSDir(blob.sha256)
	file := dir + "/" + blob.sha256
	f, err := os.Create(file)
	if err != nil {
		fmt.Printf("create file error: %s\n", err)
		return nil
	}
	return f
}

func CreateLFSDir(sha256 string) (name string) {
	lfspath := ".git/lfs/objects/" + sha256[0:2] + "/" + sha256[2:4]
	absdir, err := filepath.Abs(lfspath)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	// if filepath exists, don't create new one
	_, err = os.Stat(absdir)
	if err == nil {
		return absdir
	}
	err = os.MkdirAll(absdir, 0777)
	if err != nil {
		fmt.Printf("create directory error: %s\n", err)
		return ""
	}
	return absdir
}
