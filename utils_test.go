package main

import (
	"testing"
)

func TestUnitConvert(t *testing.T) {
	var Data_t = []struct {
		input    string
		expected uint64
	}{
		{"0B,", 0},
		{"123b", 123},
		{"1k", 1024},
		{"1000K", 1000 * 1024},
		{"1M", 1024 * 1024},
		{"1G", 1024 * 1024 * 1024},
		{"4g", 4 * 1024 * 1024 * 1024},
	}
	for _, data := range Data_t {
		out, _ := UnitConvert(data.input)
		if out != data.expected {
			t.Errorf("test UnitConvert error: expect: %v actual: %v", data.expected, out)
		}
	}
}

func TestShowScanResult(t *testing.T) {
	var Data_t = []HistoryRecord{
		// for test, the first one is the biggest one
		{"4fd1482c27853b935c11108688f63e3f4f0f9c80", 100000, "loooooooooooooooooooooooooong file"},
		{"e288c8793273629cf7a0679f4410eaf74c7108f0", 10, "short file1"},
		{"5266b09a8b363e8c50ec25488c821c441c3809a0", 100, "looooooong file"},
		{"4fd1482c27853b935c11108688f63e3f4f0f9c80", 1000, "looooooooooooooong file"},
	}
	ShowScanResult(Data_t)
}

func TestEndcodePath(t *testing.T) {
	var Data_t = []struct {
		input    string
		expected string
	}{
		{"dir/sub", "dir/sub"},
		{"顶级\\\\次级", "顶级\\次级"},
	}

	for _, data := range Data_t {
		actual := EndcodePath(data.input)
		if data.expected != actual {
			t.Errorf("test UnitConvert error: expect: %v actual: %v", data.expected, actual)
		}
	}
}

func TestTrimeDoubleQuote(t *testing.T) {
	var Data_t = []struct {
		input    string
		expected string
	}{
		{"\"quoted\"", "quoted"},
		{"non-quoted", "non-quoted"},
	}

	for _, data := range Data_t {
		actual := TrimeDoubleQuote(data.input)
		if data.expected != actual {
			t.Errorf("test UnitConvert error: expect: %v actual: %v", data.expected, actual)
		}
	}
}
