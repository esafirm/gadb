package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExtractErrorMessage(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "adb failure",
			output: "Performing Streamed Install\nadb: failed to install app.apk: Failure [INSTALL_FAILED_ALREADY_EXISTS]\n",
			want:   "adb: failed to install app.apk: Failure [INSTALL_FAILED_ALREADY_EXISTS]",
		},
		{
			name:   "pm failure without adb prefix",
			output: "Failure [INSTALL_FAILED_VERSION_DOWNGRADE]",
			want:   "Failure [INSTALL_FAILED_VERSION_DOWNGRADE]",
		},
		{
			name:   "no failure detail",
			output: "Performing Streamed Install\n",
			want:   "",
		},
		{
			name:   "empty output",
			output: "",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractErrorMessage(tt.output); got != tt.want {
				t.Fatalf("extractErrorMessage(%q) = %q, want %q", tt.output, got, tt.want)
			}
		})
	}
}

func TestCanRecoverVersionDowngradeSkipsUnrelatedErrors(t *testing.T) {
	// Should not touch the APK or apkanalyzer when the install failed for a
	// reason other than a downgrade.
	if canRecoverVersionDowngrade("missing.apk", "Failure [INSTALL_FAILED_INSUFFICIENT_STORAGE]") {
		t.Fatal("expected no recovery for an unrelated install failure")
	}
}

func TestInstallAPKUsesAndroidAndDoesNotFallbackOnFailure(t *testing.T) {
	bin := t.TempDir()
	log := filepath.Join(bin, "log")
	for name, body := range map[string]string{
		"android": "echo android \"$@\" >> '" + log + "'; exit 1\n",
		"adb":     "echo adb \"$@\" >> '" + log + "'\n",
	} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+body), 0755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin)
	result := installAPK("", "app.apk", false, 10*time.Second)
	if result.Error == nil {
		t.Fatal("expected Android CLI failure")
	}
	got, err := os.ReadFile(log)
	if err != nil {
		t.Fatalf("command error %v; log: %v", result.Error, err)
	}
	if string(got) == "" || containsLine(string(got), "adb") {
		t.Fatalf("unexpected commands: %s", got)
	}
}

func TestInstallAPKFallsBackOnlyWhenAndroidIsMissing(t *testing.T) {
	bin := t.TempDir()
	log := filepath.Join(bin, "args")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > '" + log + "'\n"
	if err := os.WriteFile(filepath.Join(bin, "adb"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	result := installAPK("", "app.apk", false, 10*time.Second)
	if result.Error != nil {
		t.Fatal(result.Error)
	}
	got, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "install\n-r\napp.apk\n" {
		t.Fatalf("adb args = %q", got)
	}
}

func TestInstallAPKFullStillUsesAndroidAndInheritedSerial(t *testing.T) {
	bin := t.TempDir()
	argsFile := filepath.Join(bin, "args")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > '" + argsFile + "'\n"
	if err := os.WriteFile(filepath.Join(bin, "android"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	t.Setenv("ANDROID_SERIAL", "inherited-device")
	result := installAPK("", "app.apk", true, 10*time.Second)
	if result.Error != nil {
		t.Fatal(result.Error)
	}
	got, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	want := "install\n--install-options=-r\n--device=inherited-device\n--use-delta-install=false\napp.apk\n"
	if string(got) != want {
		t.Fatalf("Android args = %q, want %q", got, want)
	}
}

func TestInstallAPKSelectedDeviceFallbackUsesReplacement(t *testing.T) {
	bin := t.TempDir()
	argsFile := filepath.Join(bin, "args")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > '" + argsFile + "'\n"
	if err := os.WriteFile(filepath.Join(bin, "adb"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	result := installAPK("chosen-device", "app.apk", false, 10*time.Second)
	if result.Error != nil {
		t.Fatal(result.Error)
	}
	got, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "-s\nchosen-device\ninstall\n-r\napp.apk\n" {
		t.Fatalf("adb args = %q", got)
	}
}

func TestInstallTimeoutDoesNotFallback(t *testing.T) {
	bin := t.TempDir()
	log := filepath.Join(bin, "log")
	android := "#!/bin/sh\necho android >> '" + log + "'\nexec /bin/sleep 2\n"
	adbScript := "#!/bin/sh\necho adb >> '" + log + "'\n"
	for name, script := range map[string]string{"android": android, "adb": adbScript} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(script), 0755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin)
	result := installAPK("", "app.apk", true, 20*time.Millisecond)
	if result.Error == nil || !strings.Contains(result.Error.Error(), "timed out") {
		t.Fatalf("expected timeout, got %v", result.Error)
	}
	got, err := os.ReadFile(log)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	// The deadline can expire before the shell starts under load.
	if len(got) > 0 && string(got) != "android\n" {
		t.Fatalf("commands = %q", got)
	}
}

func TestDowngradeUninstallUsesSelectedDevice(t *testing.T) {
	bin := t.TempDir()
	argsFile := filepath.Join(bin, "args")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > '" + argsFile + "'\n"
	if err := os.WriteFile(filepath.Join(bin, "adb"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	previous := installDevice
	installDevice = "selected-device"
	t.Cleanup(func() { installDevice = previous })
	if !uninstall("example.app") {
		t.Fatal("uninstall failed")
	}
	got, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "-s\nselected-device\nuninstall\nexample.app\n" {
		t.Fatalf("adb args = %q", got)
	}
}

func containsLine(text, prefix string) bool {
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, prefix+" ") {
			return true
		}
	}
	return false
}
