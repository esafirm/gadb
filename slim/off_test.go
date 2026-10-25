package slim

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/esafirm/gadb/avd"
)

func TestOffRestoresEverything(t *testing.T) {
	a := testAVD(t)
	device := newFakeDevice("emulator-5554",
		"com.google.android.googlequicksearchbox",
		"com.google.android.apps.photos",
	)

	if _, err := On(a, device, Options{}); err != nil {
		t.Fatal(err)
	}
	result, err := Off(a, device, true)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.RestoredPackages) != 2 {
		t.Fatalf("RestoredPackages = %v", result.RestoredPackages)
	}
	if len(device.disabled) != 0 {
		t.Fatalf("device still has disabled packages: %v", device.disabled)
	}
	if len(result.RestoredSettings) != 2 {
		t.Fatalf("RestoredSettings = %v", result.RestoredSettings)
	}
	if len(device.settings) != 0 {
		t.Fatalf("device still has settings: %v", device.settings)
	}
	if !result.RemovedManifest {
		t.Fatal("manifest was not removed")
	}
	if manifest, err := avd.LoadManifest(a); err != nil || manifest != nil {
		t.Fatalf("manifest = %+v, %v", manifest, err)
	}
}

func TestOffRestoresOriginalSetting(t *testing.T) {
	a := testAVD(t)
	device := newFakeDevice("emulator-5554")
	device.settings["global/background_process_limit"] = ptr("20")

	if _, err := On(a, device, Options{}); err != nil {
		t.Fatal(err)
	}
	if got := device.settings["global/background_process_limit"]; got == nil || *got != "4" {
		t.Fatalf("after On, background_process_limit = %v", got)
	}

	if _, err := Off(a, device, true); err != nil {
		t.Fatal(err)
	}
	if got := device.settings["global/background_process_limit"]; got == nil || *got != "20" {
		t.Fatalf("after Off, background_process_limit = %v, want 20", got)
	}
}

func TestOffStoppedKeepsGuestState(t *testing.T) {
	a := testAVD(t)
	device := newFakeDevice("emulator-5554",
		"com.google.android.googlequicksearchbox",
		"com.google.android.apps.photos",
	)

	if _, err := On(a, device, Options{}); err != nil {
		t.Fatal(err)
	}
	result, err := Off(a, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	if result.Running {
		t.Fatal("Running = true")
	}
	if len(result.Warnings) == 0 || !strings.Contains(result.Warnings[0], "not running") {
		t.Fatalf("Warnings = %v", result.Warnings)
	}
	if len(device.disabled) != 2 {
		t.Fatalf("device state changed while stopped: %v", device.disabled)
	}
	manifest, err := avd.LoadManifest(a)
	if err != nil {
		t.Fatal(err)
	}
	if manifest == nil || len(manifest.Packages) != 2 {
		t.Fatalf("manifest = %+v", manifest)
	}
}

func TestOffKeepsFailedPackage(t *testing.T) {
	a := testAVD(t)
	device := newFakeDevice("emulator-5554",
		"com.google.android.googlequicksearchbox",
		"com.google.android.apps.photos",
	)

	if _, err := On(a, device, Options{}); err != nil {
		t.Fatal(err)
	}
	device.failEnable["com.google.android.apps.photos"] = true

	result, err := Off(a, device, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.FailedPackages) != 1 {
		t.Fatalf("FailedPackages = %v", result.FailedPackages)
	}
	if result.RemovedManifest {
		t.Fatal("manifest was removed even though a package could not be restored")
	}
	manifest, err := avd.LoadManifest(a)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Packages) != 1 || manifest.Packages[0].Name != "com.google.android.apps.photos" {
		t.Fatalf("manifest packages = %+v", manifest.Packages)
	}
}

func TestOffRestoresConfigWhileRunning(t *testing.T) {
	a := testAVD(t)
	manifest := &avd.Manifest{
		Version: 1,
		AVD:     a.Name,
		Config: map[string]avd.ConfigChange{
			"hw.ramSize": {Original: "4096", Tuned: "2048", Existed: true},
		},
	}
	if err := avd.SaveManifest(a, manifest); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(a.HardwarePath(), []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	device := newFakeDevice("emulator-5554")
	result, err := Off(a, device, true)
	if err != nil {
		t.Fatal(err)
	}

	config, err := avd.LoadConfig(a.ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := config.Get("hw.ramSize"); got != "4096" {
		t.Fatalf("hw.ramSize = %q, want 4096", got)
	}
	if _, err := os.Stat(filepath.Join(a.Dir, "hardware-qemu.ini")); err != nil {
		t.Fatal("hardware-qemu.ini was removed while the emulator is running")
	}
	if len(result.Warnings) == 0 || !strings.Contains(strings.Join(result.Warnings, "\n"), "restart") {
		t.Fatalf("Warnings = %v", result.Warnings)
	}
	if !result.RemovedManifest {
		t.Fatal("manifest was not removed")
	}
}

func TestOffNothingToRestore(t *testing.T) {
	a := testAVD(t)
	if _, err := Off(a, nil, false); err == nil {
		t.Fatal("Off() succeeded without a manifest")
	}
}
