package main

import (
	"fmt"

	colorable "github.com/mattn/go-colorable"
)

const (
	FORMAT_RED    = "\033[31m%s\033[0m"
	FORMAT_GREEN  = "\033[32m%s\033[0m"
	FORMAT_YELLOW = "\033[33m%s\033[0m"
	FORMAT_BLUE   = "\033[34m%s\033[0m"
)

var out = colorable.NewColorableStdout()

func PrintRed(msg string) {
	fmt.Fprintf(out, FORMAT_RED, msg)
}

func PrintGreen(msg string) {
	fmt.Fprintf(out, FORMAT_GREEN, msg)
}

func PrintYellow(msg string) {
	fmt.Fprintf(out, FORMAT_YELLOW, msg)
}

func PrintBlue(msg string) {
	fmt.Fprintf(out, FORMAT_BLUE, msg)
}

func PrintPlain(msg string) {
	fmt.Print(msg)
}

func PrintRedln(msg string) {
	fmt.Fprintf(out, FORMAT_RED, msg)
	fmt.Println()
}

func PrintGreenln(msg string) {
	fmt.Fprintf(out, FORMAT_GREEN, msg)
	fmt.Println()
}

func PrintYellowln(msg string) {
	fmt.Fprintf(out, FORMAT_YELLOW, msg)
	fmt.Println()
}

func PrintBlueln(msg string) {
	fmt.Fprintf(out, FORMAT_BLUE, msg)
	fmt.Println()
}

func PrintPlainln(msg string) {
	fmt.Println(msg)
}
