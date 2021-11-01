package main

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	FORMAT_RED    = "\033[31m%s\033[0m\n"
	FORMAT_GREEN  = "\033[32m%s\033[0m\n"
	FORMAT_YELLOW = "\033[33m%s\033[0m\n"
	FORMAT_BLUE   = "\033[34m%s\033[0m\n"
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
	cv, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		return 0, err
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
		err := fmt.Errorf("expected format: --limit=<n>k|m|g, but you are: --limit=%s", input)
		return 0, err
	}
}

func PrintRed(msg string) {
	fmt.Printf(FORMAT_RED, msg)
}

func PrintGreen(msg string) {
	fmt.Printf(FORMAT_GREEN, msg)
}

func PrintYellow(msg string) {
	fmt.Printf(FORMAT_YELLOW, msg)
}

func PrintBlue(msg string) {
	fmt.Printf(FORMAT_BLUE, msg)
}
