package utils

import (
	"bufio"
	"io"
	"strings"
	"testing"
)

func TestReadInputLine(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"lf", "yes\n", "yes"},
		{"crlf", "yes\r\n", "yes"},
		{"bare cr", "yes\r", "yes"},
		{"eof without terminator", "yes", "yes"},
		{"empty line", "\n", ""},
		{"backspace", "yn\x7f\n", "y"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(c.input))
			got, err := readInputLine(reader)
			if err != nil {
				t.Fatalf("readInputLine(%q) error: %v", c.input, err)
			}
			if got != c.want {
				t.Errorf("readInputLine(%q) = %q, want %q", c.input, got, c.want)
			}
		})
	}
}

func TestReadInputLineKeepsFollowingInput(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("y\r\nno\r"))

	first, err := readInputLine(reader)
	if err != nil || first != "y" {
		t.Fatalf("first line = (%q, %v), want (\"y\", nil)", first, err)
	}

	second, err := readInputLine(reader)
	if err != nil || second != "no" {
		t.Fatalf("second line = (%q, %v), want (\"no\", nil)", second, err)
	}

	if _, err := readInputLine(reader); err != io.EOF {
		t.Fatalf("expected io.EOF after input is exhausted, got %v", err)
	}
}

func TestReadInputLineEOF(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader(""))

	line, err := readInputLine(reader)
	if err != io.EOF || line != "" {
		t.Fatalf("readInputLine on empty input = (%q, %v), want (\"\", io.EOF)", line, err)
	}
}

func TestInterpretYesNo(t *testing.T) {
	cases := []struct {
		input    string
		expected bool
		valid    bool
	}{
		{"y", true, true},
		{"Y", true, true},
		{"yes", true, true},
		{"YES", true, true},
		{" y ", true, true},
		{"n", false, true},
		{"N", false, true},
		{"no", false, true},
		{"No", false, true},
		{"", false, true}, // Enter answers the default [y/N]
		{"  ", false, true},
		{"maybe", false, false},
		{"yy", false, false},
		{"nah", false, false},
		{"1", false, false},
	}

	for _, c := range cases {
		answer, valid := interpretYesNo(c.input)
		if answer != c.expected || valid != c.valid {
			t.Errorf("interpretYesNo(%q) = (%v, %v), want (%v, %v)",
				c.input, answer, valid, c.expected, c.valid)
		}
	}
}
