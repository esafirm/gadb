package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestBackendCommandsForwardArgumentsAndStreams(t *testing.T) {
	for _, tc := range []struct {
		name string
		cmd  func() *cobra.Command
		want []string
		args []string
	}{
		{"run", func() *cobra.Command { return runCmd }, []string{"run", "app", "--", "--offline", "-Pfoo=bar"}, []string{"app", "--", "--offline", "-Pfoo=bar"}},
		{"profiler", func() *cobra.Command { return profilerCmd }, []string{"profiler", "session", "list", "--json"}, []string{"session", "list", "--json"}},
		{"artifact", func() *cobra.Command { return artifactCmd }, []string{"analyze", "--format=json", "app.aab"}, []string{"--format=json", "app.aab"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			capture := filepath.Join(dir, "args")
			android := filepath.Join(dir, "android")
			if err := os.WriteFile(android, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$CAPTURE\"\nprintf stdout\nprintf stderr >&2\n"), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir)
			t.Setenv("CAPTURE", capture)
			cmd := &cobra.Command{Use: tc.cmd().Use, Short: tc.cmd().Short, Long: tc.cmd().Long, DisableFlagParsing: true, RunE: tc.cmd().RunE}
			var stdout, stderr bytes.Buffer
			cmd.SetArgs(tc.args)
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			cmd.SetIn(strings.NewReader("input"))
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(capture)
			if err != nil {
				t.Fatal(err)
			}
			if actual := strings.Split(strings.TrimSuffix(string(got), "\n"), "\n"); !reflect.DeepEqual(actual, tc.want) {
				t.Fatalf("args = %#v, want %#v", actual, tc.want)
			}
			if stdout.String() != "stdout" || stderr.String() != "stderr" {
				t.Fatalf("streams stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
		})
	}
}

func TestBackendCommandsForwardBackendHelp(t *testing.T) {
	dir := t.TempDir()
	capture := filepath.Join(dir, "args")
	if err := os.WriteFile(filepath.Join(dir, "android"), []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$CAPTURE\"\nprintf 'backend-specific help'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("CAPTURE", capture)
	for _, tc := range []struct {
		original *cobra.Command
		args     []string
		want     []string
	}{
		{runCmd, []string{"--help"}, []string{"run", "--help"}},
		{profilerCmd, []string{"session", "--help"}, []string{"profiler", "session", "--help"}},
		{artifactCmd, []string{"-h"}, []string{"analyze", "-h"}},
	} {
		original := tc.original
		cmd := &cobra.Command{Use: original.Use, Short: original.Short, Long: original.Long, DisableFlagParsing: true, RunE: original.RunE}
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetArgs(tc.args)
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if out.String() != "backend-specific help" {
			t.Errorf("%s output = %q", cmd.Name(), out.String())
		}
		got, err := os.ReadFile(capture)
		if err != nil {
			t.Fatal(err)
		}
		if actual := strings.Split(strings.TrimSuffix(string(got), "\n"), "\n"); !reflect.DeepEqual(actual, tc.want) {
			t.Errorf("args = %#v, want %#v", actual, tc.want)
		}
	}
}

func TestBackendCommandHelpFallsBackWhenAndroidMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	cmd := &cobra.Command{Use: runCmd.Use, Short: runCmd.Short, Long: runCmd.Long, DisableFlagParsing: true, RunE: runCmd.RunE}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Usage: android run") {
		t.Fatalf("fallback help missing usage: %q", out.String())
	}
}

func TestBackendCommandMissingAndroidIsActionable(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	cmd := &cobra.Command{Use: runCmd.Use, DisableFlagParsing: true, RunE: runCmd.RunE}
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "Android CLI is required") {
		t.Fatalf("error = %v", err)
	}
}
