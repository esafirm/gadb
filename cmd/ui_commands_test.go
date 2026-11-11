package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func fakeTool(t *testing.T, name, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
	return path
}

func TestLayoutForwardsOptions(t *testing.T) {
	argsFile := filepath.Join(t.TempDir(), "args")
	fakeTool(t, "android", "printf '%s\\n' \"$@\" > "+argsFile)
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	layoutDevice, layoutFull, layoutFlat, layoutNoIdle, layoutPretty, layoutOutput = "emulator-1", true, true, true, true, "tree.json"
	defer func() {
		layoutDevice = ""
		layoutFull = false
		layoutFlat = false
		layoutNoIdle = false
		layoutPretty = false
		layoutOutput = ""
	}()
	if err := layoutCmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(argsFile)
	for _, want := range []string{"layout", "--device\nemulator-1", "--full", "--flat", "--no-idle", "--pretty", "--output\ntree.json"} {
		if !strings.Contains(string(got), want) {
			t.Fatalf("arguments %q missing %q", got, want)
		}
	}
}

func TestSemanticInputForwardsDeviceAndRawArgs(t *testing.T) {
	argsFile := filepath.Join(t.TempDir(), "args")
	fakeTool(t, "android", "printf '%s\\n' \"$@\" > "+argsFile)
	cmd := &cobra.Command{}
	inputDevice = "serial"
	defer func() { inputDevice = "" }()
	if err := semanticInputCmd.RunE(cmd, []string{"tap", "text=Save", "--wait", "10"}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(argsFile)
	if string(got) != "input\n--device\nserial\ntap\ntext=Save\n--wait\n10\n" {
		t.Fatalf("unexpected args: %q", got)
	}
}

func TestCoordinateInputUsesADBAndScreenshotStreamsBinary(t *testing.T) {
	argsFile := filepath.Join(t.TempDir(), "args")
	fakeTool(t, "adb", "printf '%s\\n' \"$@\" > "+argsFile+"\nprintf '\\211PNG\\r\\n\\032\\n'\n")
	inputDevice = "serial"
	defer func() { inputDevice = "" }()
	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)
	coordinate := coordinateOperation("tap", 2)
	if err := coordinate.RunE(cmd, []string{"12", "34"}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(argsFile)
	if string(got) != "-s\nserial\nshell\ninput\ntap\n12\n34\n" {
		t.Fatalf("unexpected args: %q", got)
	}
	if err := screenshotCmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out.Bytes(), []byte("PNG")) {
		t.Fatalf("binary screenshot missing: %v", out.Bytes())
	}
}

func TestCoordinateInputQuotesTextAndRejectsShellSyntaxNumbers(t *testing.T) {
	argsFile := filepath.Join(t.TempDir(), "args")
	fakeTool(t, "adb", "printf '%s\\n' \"$@\" > "+argsFile)
	cmd := &cobra.Command{}
	if err := coordinateOperation("text", 1).RunE(cmd, []string{"a b; $(touch /tmp/nope) 'quoted'"}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(argsFile)
	if !strings.Contains(string(got), "'a%sb;%s$(touch%s/tmp/nope)%s'\\''quoted'\\'''\n") {
		t.Fatalf("text not safely quoted and space-converted: %q", got)
	}
	if shellQuote("") != "''" {
		t.Fatalf("empty shell argument quoted as %q", shellQuote(""))
	}
	for _, tc := range []struct {
		operation string
		args      []string
	}{{"tap", []string{"1;id", "2"}}, {"swipe", []string{"1", "2", "3", "4", "$(id)"}}, {"keyevent", []string{"1;id"}}} {
		before, _ := os.ReadFile(argsFile)
		count := map[string]int{"tap": 2, "swipe": 4, "keyevent": 1}[tc.operation]
		if err := coordinateOperation(tc.operation, count).RunE(cmd, tc.args); err == nil {
			t.Fatalf("%s accepted unsafe argument %q", tc.operation, tc.args)
		}
		after, _ := os.ReadFile(argsFile)
		if !bytes.Equal(before, after) {
			t.Fatalf("%s ran adb after invalid arguments", tc.operation)
		}
	}
	if err := coordinateOperation("swipe", 4).RunE(cmd, []string{"1", "2", "3", "4", "250"}); err != nil {
		t.Fatalf("swipe duration rejected: %v", err)
	}
}

func TestSemanticDeviceFlagBeforeAndAfterSubcommand(t *testing.T) {
	argsFile := filepath.Join(t.TempDir(), "args")
	fakeTool(t, "android", "printf '%s\\n' \"$@\" > "+argsFile)
	for _, tc := range []struct {
		argv []string
		want string
	}{
		{[]string{"input", "-s", "SERIAL", "semantic", "tap", "text=Save"}, "input\n--device\nSERIAL\ntap\ntext=Save\n"},
		{[]string{"input", "--device=SERIAL", "semantic", "tap", "text=Save"}, "input\n--device\nSERIAL\ntap\ntext=Save\n"},
		{[]string{"input", "semantic", "tap", "--device", "SERIAL", "text=Save"}, "input\ntap\n--device\nSERIAL\ntext=Save\n"},
		{[]string{"input", "semantic", "text", "", "--", "--device"}, "input\ntext\n\n--\n--device\n"},
	} {
		inputDevice = ""
		rootCmd.SetArgs(tc.argv)
		rootCmd.SetOut(&bytes.Buffer{})
		rootCmd.SetErr(&bytes.Buffer{})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("input %v: %v", tc.argv, err)
		}
		got, _ := os.ReadFile(argsFile)
		if string(got) != tc.want {
			t.Fatalf("input %v args: %q, want %q", tc.argv, got, tc.want)
		}
	}
	inputDevice = ""
	if err := semanticInputCmd.RunE(&cobra.Command{}, []string{"--device"}); err == nil || !strings.Contains(err.Error(), "requires a device serial") {
		t.Fatalf("missing prefix device value error = %v", err)
	}
}

func TestAnnotatedScreenshotWritesStdoutAndPreservesExistingFileOnFailure(t *testing.T) {
	dir := t.TempDir()
	android := filepath.Join(dir, "android")
	if err := os.WriteFile(android, []byte("#!/bin/sh\nfor last do :; done\ncase \"$FAIL\" in 1) exit 9;; esac\nprintf '\\211PNG\\r\\n\\032\\n' > \"$last\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
	t.Setenv("FAIL", "")
	screenshotAnnotated, screenshotOutput = true, ""
	defer func() { screenshotAnnotated = false; screenshotOutput = "" }()
	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)
	if err := screenshotCmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(out.Bytes(), []byte("\x89PNG\r\n\x1a\n")) {
		t.Fatalf("annotated screenshot not sent to stdout: %q", out.Bytes())
	}
	path := filepath.Join(dir, "existing.png")
	if err := os.WriteFile(path, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	screenshotOutput = path
	t.Setenv("FAIL", "1")
	if err := screenshotCmd.RunE(cmd, nil); err == nil {
		t.Fatal("expected android failure")
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "keep" {
		t.Fatalf("existing output changed: %q, %v", got, err)
	}
}

func TestScreenshotPreservesExistingFileOnADBFailure(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "adb"), []byte("#!/bin/sh\nexit 9\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
	path := filepath.Join(dir, "existing.png")
	if err := os.WriteFile(path, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	screenshotAnnotated, screenshotOutput = false, path
	defer func() { screenshotOutput = "" }()
	if err := screenshotCmd.RunE(&cobra.Command{}, nil); err == nil {
		t.Fatal("expected adb failure")
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "keep" {
		t.Fatalf("existing output changed: %q, %v", got, err)
	}
}
