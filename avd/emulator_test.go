package avd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestLaunchArgs(t *testing.T) {
	a := AVD{Name: "Pixel_Test"}

	args := launchArgs(a, LaunchOptions{
		RAM:        1536,
		GPU:        "host",
		LowRAM:     true,
		Headless:   true,
		NoAudio:    true,
		NoBootAnim: true,
		Snapshot:   CleanSnapshotName,
	}, 5556)
	joined := strings.Join(args, " ")
	for _, want := range []string{
		"-avd Pixel_Test",
		"-memory 1536",
		"-gpu host",
		"-port 5556",
		"-lowram",
		"-no-window",
		"-no-audio",
		"-no-boot-anim",
		"-snapshot " + CleanSnapshotName,
		"-no-snapshot-save",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("launch args missing %q: %s", want, joined)
		}
	}

	args = launchArgs(a, LaunchOptions{RAM: 2048, GPU: "auto"}, 5554)
	joined = strings.Join(args, " ")
	if strings.Contains(joined, "-snapshot "+CleanSnapshotName) {
		t.Errorf("cold launch still uses a snapshot: %s", joined)
	}
	if !strings.Contains(joined, "-no-snapshot-load") {
		t.Errorf("cold launch is missing -no-snapshot-load: %s", joined)
	}
	for _, unwanted := range []string{"-lowram", "-no-window", "-no-audio", "-no-boot-anim"} {
		if strings.Contains(joined, unwanted) {
			t.Errorf("cold launch unexpectedly has %q: %s", unwanted, joined)
		}
	}
}

func TestFreeEmulatorPort(t *testing.T) {
	port, err := FreeEmulatorPort()
	if err != nil {
		t.Fatal(err)
	}
	if port < 5554 || port > 5682 || port%2 != 0 {
		t.Fatalf("FreeEmulatorPort() = %d, want an even port in 5554..5682", port)
	}
}

func TestFindEmulatorUsesSDK(t *testing.T) {
	sdk := t.TempDir()
	t.Setenv("ANDROID_HOME", sdk)
	t.Setenv("ANDROID_SDK_ROOT", "")
	t.Setenv("PATH", t.TempDir())

	name := "emulator"
	if runtime.GOOS == "windows" {
		name = "emulator.exe"
	}
	path := filepath.Join(sdk, "emulator", name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	if got := FindEmulator(); got != path {
		t.Fatalf("FindEmulator() = %q, want %q", got, path)
	}
}

func TestTailLog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "emulator.log")
	var lines []string
	for i := 1; i <= 100; i++ {
		lines = append(lines, fmt.Sprintf("line%d", i))
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got, want := TailLog(path, 3), "line98\nline99\nline100"; got != want {
		t.Fatalf("TailLog() = %q, want %q", got, want)
	}
	if got := TailLog(filepath.Join(t.TempDir(), "missing.log"), 3); got != "" {
		t.Fatalf("TailLog(missing) = %q, want empty", got)
	}
}

func TestLaunchStartsDetached(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake emulator is a shell script")
	}

	dir := t.TempDir()
	script := "#!/bin/sh\necho started\nexec /bin/sleep 30\n"
	if err := os.WriteFile(filepath.Join(dir, "emulator"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	a := AVD{Name: "Test", Dir: t.TempDir()}
	process, err := Launch(a, LaunchOptions{RAM: 2048, GPU: "host", Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if !process.Exited() {
			_ = process.Cmd.Process.Kill()
		}
	})

	if process.Serial != "emulator-"+strconv.Itoa(process.Port) {
		t.Fatalf("Serial = %q, port = %d", process.Serial, process.Port)
	}
	if _, err := os.Stat(process.LogPath); err != nil {
		t.Fatalf("log file: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if data, err := os.ReadFile(process.LogPath); err == nil && strings.Contains(string(data), "started") {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if data, _ := os.ReadFile(process.LogPath); !strings.Contains(string(data), "started") {
		t.Fatalf("log does not contain the emulator output: %q", data)
	}

	if process.Exited() {
		t.Fatal("emulator exited early")
	}
	if err := process.Cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-process.done:
	case <-time.After(5 * time.Second):
		t.Fatal("emulator did not exit after Kill")
	}
}
