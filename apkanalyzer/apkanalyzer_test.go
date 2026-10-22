package adb

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackageNameWithoutAnalyzer(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("ANDROID_HOME", "")
	t.Setenv("ANDROID_SDK_ROOT", "")

	_, err := PackageName("app.apk")
	if err == nil || !strings.Contains(err.Error(), "not in the path") {
		t.Fatalf("expected missing analyzer error, got %v", err)
	}
}

func TestPackageNameFindsAnalyzerInSDK(t *testing.T) {
	for _, env := range []string{"ANDROID_HOME", "ANDROID_SDK_ROOT"} {
		t.Run(env, func(t *testing.T) {
			sdk := t.TempDir()
			bin := filepath.Join(sdk, "cmdline-tools", "22.0", "bin")
			if err := os.MkdirAll(bin, 0o755); err != nil {
				t.Fatal(err)
			}
			script := "#!/bin/sh\necho '<manifest package=\"com.example.fake\" />'\n"
			if err := os.WriteFile(filepath.Join(bin, ANALYZER), []byte(script), 0o755); err != nil {
				t.Fatal(err)
			}

			t.Setenv("PATH", t.TempDir())
			t.Setenv("ANDROID_HOME", "")
			t.Setenv("ANDROID_SDK_ROOT", "")
			t.Setenv(env, sdk)

			name, err := PackageName("app.apk")
			if err != nil {
				t.Fatal(err)
			}
			if name != "com.example.fake" {
				t.Fatalf("PackageName = %q, want %q", name, "com.example.fake")
			}
		})
	}
}
