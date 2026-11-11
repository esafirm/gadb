package adb

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestBuildAvdRunArgs(t *testing.T) {
	tests := []struct {
		name         string
		avdName      string
		coldBoot     bool
		emulatorArgs []string
		want         []string
	}{
		{
			name:    "default",
			avdName: "Pixel_7_Pro_API_34",
			want:    []string{"@Pixel_7_Pro_API_34"},
		},
		{
			name:     "cold boot",
			avdName:  "Pixel_7_Pro_API_34",
			coldBoot: true,
			want:     []string{"@Pixel_7_Pro_API_34", "-no-snapshot"},
		},
		{
			name:         "extra emulator args",
			avdName:      "Pixel_7_Pro_API_34",
			emulatorArgs: []string{"-no-window", "-no-audio"},
			want:         []string{"@Pixel_7_Pro_API_34", "-no-window", "-no-audio"},
		},
		{
			name:         "cold boot with extra emulator args",
			avdName:      "Pixel_7_Pro_API_34",
			coldBoot:     true,
			emulatorArgs: []string{"-gpu", "swiftshader_indirect"},
			want:         []string{"@Pixel_7_Pro_API_34", "-no-snapshot", "-gpu", "swiftshader_indirect"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildAvdRunArgs(tt.avdName, tt.coldBoot, tt.emulatorArgs)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("buildAvdRunArgs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestAndroidInstallArgs(t *testing.T) {
	bin := t.TempDir()
	argsFile := filepath.Join(bin, "args")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"" + argsFile + "\"\n"
	if err := os.WriteFile(filepath.Join(bin, "android"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	result := AndroidInstall("serial-1", "/tmp/app.apk", false, 10*time.Second)
	if result.Error != nil {
		t.Fatal(result.Error)
	}
	got, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	want := "install\n--install-options=-r\n--device=serial-1\n--use-delta-install=true\n/tmp/app.apk\n"
	if strings.ReplaceAll(string(got), "\r\n", "\n") != want {
		t.Fatalf("args = %q, want %q", got, want)
	}
}
