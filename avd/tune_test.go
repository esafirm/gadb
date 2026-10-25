package avd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleConfig = `# keep this comment
hw.ramSize=1536
hw.gpu.mode=auto
hw.gpu.enabled=no
hw.audioInput=yes
hw.audioOutput=yes
hw.camera.back=virtualscene
hw.camera.front=emulated
hw.keyboard=no
`

func testAVDWithConfig(t *testing.T, content string) AVD {
	t.Helper()

	a := testAVD(t)
	if err := os.WriteFile(a.ConfigPath(), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return a
}

func stubStopped(t *testing.T) {
	t.Helper()

	previous := runningEmulator
	runningEmulator = func(string) (string, bool) { return "", false }
	t.Cleanup(func() { runningEmulator = previous })
}

func readConfig(t *testing.T, a AVD) *Config {
	t.Helper()

	config, err := LoadConfig(a.ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	return config
}

func TestTuneWritesConfigAndManifest(t *testing.T) {
	a := testAVDWithConfig(t, sampleConfig)
	stubStopped(t)

	if err := os.WriteFile(a.HardwarePath(), []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Tune(a, TuneOptions{RAM: 2048, GPU: "host"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.RemovedHardware {
		t.Error("RemovedHardware = false, want true")
	}

	config := readConfig(t, a)
	want := map[string]string{
		"hw.ramSize":      "2048",
		"hw.gpu.mode":     "host",
		"hw.gpu.enabled":  "yes",
		"hw.audioInput":   "no",
		"hw.audioOutput":  "no",
		"hw.camera.back":  "none",
		"hw.camera.front": "none",
		"hw.keyboard":     "yes",
	}
	for key, value := range want {
		if got, _ := config.Get(key); got != value {
			t.Errorf("%s = %q, want %q", key, got, value)
		}
	}

	manifest, err := LoadManifest(a)
	if err != nil {
		t.Fatal(err)
	}
	if manifest == nil {
		t.Fatal("manifest = nil")
	}
	change := manifest.Config["hw.ramSize"]
	if change.Original != "1536" || change.Tuned != "2048" || !change.Existed {
		t.Fatalf("ram change = %+v", change)
	}
	if change := manifest.Config["hw.gpu.enabled"]; change.Original != "no" || change.Tuned != "yes" {
		t.Fatalf("gpu enabled change = %+v", change)
	}

	backup, err := os.ReadFile(filepath.Join(a.Dir, BackupName))
	if err != nil {
		t.Fatal(err)
	}
	if string(backup) != sampleConfig {
		t.Fatalf("backup =\n%s\nwant\n%s", backup, sampleConfig)
	}

	if _, err := os.Stat(a.HardwarePath()); !os.IsNotExist(err) {
		t.Fatal("hardware-qemu.ini was not removed")
	}
}

func TestTuneKeepsOriginalAcrossRuns(t *testing.T) {
	a := testAVDWithConfig(t, sampleConfig)
	stubStopped(t)

	if _, err := Tune(a, TuneOptions{RAM: 2048, GPU: "host"}); err != nil {
		t.Fatal(err)
	}
	if _, err := Tune(a, TuneOptions{RAM: 1536, GPU: "host"}); err != nil {
		t.Fatal(err)
	}

	manifest, err := LoadManifest(a)
	if err != nil {
		t.Fatal(err)
	}
	change := manifest.Config["hw.ramSize"]
	if change.Original != "1536" || change.Tuned != "1536" {
		t.Fatalf("ram change = %+v, want original 1536 and tuned 1536", change)
	}

	if got, _ := readConfig(t, a).Get("hw.ramSize"); got != "1536" {
		t.Fatalf("hw.ramSize = %q, want 1536", got)
	}
}

func TestTuneKeepAudioRestoresPreviousTune(t *testing.T) {
	a := testAVDWithConfig(t, sampleConfig)
	stubStopped(t)

	if _, err := Tune(a, TuneOptions{RAM: 2048, GPU: "host"}); err != nil {
		t.Fatal(err)
	}
	if _, err := Tune(a, TuneOptions{RAM: 2048, GPU: "host", KeepAudio: true, KeepCamera: true}); err != nil {
		t.Fatal(err)
	}

	config := readConfig(t, a)
	for key, value := range map[string]string{
		"hw.audioInput":   "yes",
		"hw.audioOutput":  "yes",
		"hw.camera.back":  "virtualscene",
		"hw.camera.front": "emulated",
	} {
		if got, _ := config.Get(key); got != value {
			t.Errorf("%s = %q, want %q", key, got, value)
		}
	}

	manifest, err := LoadManifest(a)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"hw.audioInput", "hw.audioOutput", "hw.camera.back", "hw.camera.front"} {
		if _, recorded := manifest.Config[key]; recorded {
			t.Errorf("%s is still recorded after --keep-audio/--keep-camera", key)
		}
	}
}

func TestRestoreConfig(t *testing.T) {
	a := testAVDWithConfig(t, sampleConfig)
	stubStopped(t)

	if _, err := Tune(a, TuneOptions{RAM: 2048, GPU: "host"}); err != nil {
		t.Fatal(err)
	}
	result, err := RestoreConfig(a)
	if err != nil {
		t.Fatal(err)
	}
	if !result.RemovedManifest {
		t.Error("RemovedManifest = false, want true")
	}

	got, err := os.ReadFile(a.ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != sampleConfig {
		t.Fatalf("restored config =\n%s\nwant\n%s", got, sampleConfig)
	}

	if _, err := os.Stat(a.ManifestPath()); !os.IsNotExist(err) {
		t.Fatal("manifest was not removed")
	}
	if _, err := os.Stat(filepath.Join(a.Dir, BackupName)); err != nil {
		t.Fatal("backup was removed")
	}
}

func TestRestoreConfigSkipsUserEdits(t *testing.T) {
	a := testAVDWithConfig(t, sampleConfig)
	stubStopped(t)

	if _, err := Tune(a, TuneOptions{RAM: 2048, GPU: "host"}); err != nil {
		t.Fatal(err)
	}

	config := readConfig(t, a)
	config.Set("hw.gpu.mode", "swiftshader_indirect")
	if err := config.Save(); err != nil {
		t.Fatal(err)
	}

	result, err := RestoreConfig(a)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := readConfig(t, a).Get("hw.gpu.mode"); got != "swiftshader_indirect" {
		t.Fatalf("hw.gpu.mode = %q, want the user's swiftshader_indirect", got)
	}
	if len(result.Skipped) != 1 || !strings.Contains(result.Skipped[0], "hw.gpu.mode") {
		t.Fatalf("Skipped = %+v, want hw.gpu.mode", result.Skipped)
	}
	if result.RemovedManifest {
		t.Fatal("manifest was removed even though hw.gpu.mode could not be restored")
	}
}

func TestRestoreConfigCleansUpAlreadyRestored(t *testing.T) {
	a := testAVDWithConfig(t, sampleConfig)
	stubStopped(t)

	if _, err := Tune(a, TuneOptions{RAM: 2048, GPU: "host"}); err != nil {
		t.Fatal(err)
	}

	config := readConfig(t, a)
	config.Set("hw.gpu.mode", "auto") // the original value, restored by hand
	if err := config.Save(); err != nil {
		t.Fatal(err)
	}

	result, err := RestoreConfig(a)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Skipped) != 0 {
		t.Fatalf("Skipped = %+v, want no conflicts", result.Skipped)
	}
	if !result.RemovedManifest {
		t.Fatal("manifest was not removed after everything was restored")
	}
	if got, _ := readConfig(t, a).Get("hw.gpu.mode"); got != "auto" {
		t.Fatalf("hw.gpu.mode = %q, want auto", got)
	}
}

func TestRestoreConfigRefusesRunningAVD(t *testing.T) {
	a := testAVDWithConfig(t, sampleConfig)
	stubStopped(t)

	if _, err := Tune(a, TuneOptions{RAM: 2048, GPU: "host"}); err != nil {
		t.Fatal(err)
	}

	previous := runningEmulator
	runningEmulator = func(string) (string, bool) { return "emulator-5554", true }
	t.Cleanup(func() { runningEmulator = previous })

	if _, err := RestoreConfig(a); err == nil || !strings.Contains(err.Error(), "is running") {
		t.Fatalf("RestoreConfig() error = %v, want a running AVD error", err)
	}
}

func TestTuneSkipsUnchangedKeys(t *testing.T) {
	a := testAVDWithConfig(t, "hw.ramSize=2048\nhw.gpu.mode=host\nhw.gpu.enabled=yes\nhw.keyboard=yes\nhw.audioInput=no\nhw.audioOutput=no\nhw.camera.back=none\nhw.camera.front=none\n")
	stubStopped(t)

	if _, err := Tune(a, TuneOptions{RAM: 2048, GPU: "host"}); err != nil {
		t.Fatal(err)
	}

	manifest, err := LoadManifest(a)
	if err != nil {
		t.Fatal(err)
	}
	if manifest == nil {
		t.Fatal("manifest = nil, want a marker even when nothing changed")
	}
	if len(manifest.Config) != 0 {
		t.Fatalf("manifest.Config = %+v, want no entries for unchanged values", manifest.Config)
	}
}

func TestTuneRefusesRunningAVD(t *testing.T) {
	a := testAVDWithConfig(t, sampleConfig)

	previous := runningEmulator
	runningEmulator = func(string) (string, bool) { return "emulator-5554", true }
	t.Cleanup(func() { runningEmulator = previous })

	_, err := Tune(a, TuneOptions{})
	if err == nil || !strings.Contains(err.Error(), "is running") {
		t.Fatalf("Tune() error = %v, want a running AVD error", err)
	}
}
