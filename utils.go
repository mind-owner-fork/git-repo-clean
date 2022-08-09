package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Convert number to bytes according to Uint
// e.g. 10 Kib => (10 * 1024) bytes
// valid unit: b, B, k, K, m, M, g, G
func UnitConvert(input string) (uint64, error) {
	if len(input) == 0 {
		return 0, fmt.Errorf("expected a value followed by --limit options, but you are: %s", input)
	}
	v := input[:len(input)-1]
	u := input[len(input)-1:]
	// set bitsize to 64, means max single blob size is 4 GiB
	cv, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse uint error: %s", err)
	}
	if strings.ToLower(u) == "b" {
		return cv, nil
	} else if strings.ToLower(u) == "k" {
		return cv * 1024, nil
	} else if strings.ToLower(u) == "m" {
		return cv * 1024 * 1024, nil
	} else if strings.ToLower(u) == "g" {
		return cv * 1024 * 1024 * 1024, nil
	} else {
		err := fmt.Errorf("expected format: --limit=<n>b|k|m|g, but you are: --limit=%s", input)
		return 0, err
	}
}

func ShowScanResult(list BlobList) {
	PrintLocalWithGreenln("scan done!")
	PrintLocalWithYellowln("note that there may be multiple versions of the same file")

	// if maxNameLen = 58 maxUTF8NameLen = 34, then ActualLen = (58-34)/2
	maxNameLen, maxUTF8NameLen := maxLenBlobName(list)
	ActualLen := maxUTF8NameLen + (maxNameLen-maxUTF8NameLen)/2

	// fix for too short file name
	if ActualLen < 9 {
		ActualLen = 9
	}
	maxSizeLen := maxLenBlobSize(list)
	// fix for too small file size
	if maxSizeLen < 4 {
		maxSizeLen = 4
	}
	fmt.Println()
	fmt.Printf("|-%-*s | %-*s------ | %-*s-|\n", 40, strings.Repeat("-", 40), maxSizeLen, strings.Repeat("-", maxSizeLen), ActualLen, strings.Repeat("-", ActualLen))
	fmt.Printf("| %-*s | %-*s bytes | %-*s |\n", 40, "Blob ID", maxSizeLen, "SIZE", ActualLen, "File Name")
	fmt.Printf("|-%-*s | %-*s------ | %-*s-|\n", 40, strings.Repeat("-", 40), maxSizeLen, strings.Repeat("-", maxSizeLen), ActualLen, strings.Repeat("-", ActualLen))
	for _, item := range list {
		d := len(item.objectName) - len([]rune(item.objectName))
		if d != 0 {
			fmt.Printf("| %.*s | %.*d bytes | %-*s |\n", 40, item.oid, maxSizeLen, item.objectSize, ActualLen-d/2, item.objectName)
		} else {
			fmt.Printf("| %.*s | %.*d bytes | %-*s |\n", 40, item.oid, maxSizeLen, item.objectSize, ActualLen, item.objectName)
		}
	}
	fmt.Printf("|-%-*s | %-*s------ | %-*s-|\n", 40, strings.Repeat("-", 40), maxSizeLen, strings.Repeat("-", maxSizeLen), ActualLen, strings.Repeat("-", ActualLen))
	fmt.Println()
}

func maxLenBlobName(list BlobList) (int, int) {
	// fix Chinese Character issue: a Chinese Character is 3 bytes, but a English Letter is 1 byte.
	// s1 := "abcd" b1 := []byte(s1) fmt.Println(b1) // [97 98 99 100]
	// s2 := "中文" b2 := []byte(s2) fmt.Println(b2) // [228 184 173 230 150 135]
	// r3 := []rune(s2)  fmt.Println(r3) // [20013 25991]
	var maxNameLen = 0
	var maxUTF8NameLen = 0
	for _, item := range list {
		if len(item.objectName) > maxNameLen {
			maxNameLen = len(item.objectName)
			maxUTF8NameLen = len([]rune(item.objectName))
		}
	}
	return maxNameLen, maxUTF8NameLen
}

func maxLenBlobSize(list BlobList) int {
	// the first one is the biggest one
	return len(strconv.Itoa(int(list[0].objectSize)))
}

func EndcodePath(path string) string {
	path = strings.ReplaceAll(path, "\\\\", "\\")
	return path
}

func TrimeDoubleQuote(s string) string {
	if len(s) > 0 && s[0] == '"' {
		s = s[1:]
	}
	if len(s) > 0 && s[len(s)-1] == '"' {
		s = s[:len(s)-1]
	}
	return s
}
