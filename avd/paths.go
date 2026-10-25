// Package avd owns the host side of Android Virtual Device management:
// paths, config.ini, system images, tuned state and snapshots. It never
// changes the guest. Guest-side slimming lives in the slim package.
package avd

import (
	"os"
	"path/filepath"
	"runtime"
)

// BaseDir is the directory holding AVD definitions (<name>.ini and <name>.avd).
// It honours ANDROID_AVD_HOME, then ANDROID_USER_HOME, then the legacy
// ANDROID_SDK_HOME, then ~/.android/avd.
func BaseDir() string {
	if dir := os.Getenv("ANDROID_AVD_HOME"); dir != "" {
		return dir
	}
	if dir := os.Getenv("ANDROID_USER_HOME"); dir != "" {
		return filepath.Join(dir, "avd")
	}
	if dir := os.Getenv("ANDROID_SDK_HOME"); dir != "" {
		return filepath.Join(dir, ".android", "avd")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".android", "avd")
}

// SDKDir is the Android SDK root. Empty when it cannot be determined.
func SDKDir() string {
	for _, env := range []string{"ANDROID_HOME", "ANDROID_SDK_ROOT"} {
		if dir := os.Getenv(env); dir != "" {
			return dir
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Android", "sdk")
	case "windows":
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			return filepath.Join(local, "Android", "Sdk")
		}
		return filepath.Join(home, "AppData", "Local", "Android", "Sdk")
	default:
		return filepath.Join(home, "Android", "Sdk")
	}
}

// HostABI is the system-image ABI that runs natively on this machine.
func HostABI() string {
	if runtime.GOARCH == "arm64" {
		return "arm64-v8a"
	}
	return "x86_64"
}

// DefaultGPU picks the best emulator GPU mode for this host. Headless Linux
// has no display server, so it falls back to software rendering.
func DefaultGPU() string {
	switch runtime.GOOS {
	case "darwin", "windows":
		return "host"
	}
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		return "swiftshader_indirect"
	}
	return "auto"
}
