package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunToolCommandStreamsSeparately(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "android")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nread line\nprintf '%s\\n' \"$line\" \"$@\"\nprintf 'diagnostic' >&2\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	var out, stderr bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader("input\n"))
	cmd.SetOut(&out)
	cmd.SetErr(&stderr)
	if err := runToolCommand(cmd, "android", "layout", "a; b"); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "input\nlayout\na; b\n" {
		t.Fatalf("stdout = %q", got)
	}
	if stderr.String() != "diagnostic" {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunToolCommandMissingAndroid(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	err := runToolCommand(&cobra.Command{}, "android", "layout")
	if err == nil || !strings.Contains(err.Error(), "https://developer.android.com/cli") {
		t.Fatalf("expected actionable installation error, got %v", err)
	}
}

func TestRunToolCommandPropagatesFailure(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "android"), []byte("#!/bin/sh\nexit 7\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	if err := runToolCommand(&cobra.Command{}, "android", "layout"); err == nil {
		t.Fatal("expected backend failure")
	}
}
