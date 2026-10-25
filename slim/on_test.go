package slim

import (
	"testing"

	"github.com/esafirm/gadb/avd"
)

func TestOnAppliesAndRecords(t *testing.T) {
	a := testAVD(t)
	device := newFakeDevice("emulator-5554",
		"com.google.android.googlequicksearchbox",
		"com.google.android.apps.photos",
		"com.google.android.gms",
	)

	result, err := On(a, device, Options{})
	if err != nil {
		t.Fatal(err)
	}

	assertSet(t, result.Disabled, set(
		"com.google.android.googlequicksearchbox",
		"com.google.android.apps.photos",
	))
	if !device.disabled["com.google.android.apps.photos"] {
		t.Error("photos was not disabled on the device")
	}
	if device.disabled["com.google.android.gms"] {
		t.Error("protected GMS package was disabled")
	}
	if len(result.Settings) != 2 {
		t.Fatalf("settings = %+v, want 2", result.Settings)
	}

	manifest, err := avd.LoadManifest(a)
	if err != nil {
		t.Fatal(err)
	}
	if manifest == nil || len(manifest.Packages) != 2 || len(manifest.Settings) != 2 {
		t.Fatalf("manifest = %+v", manifest)
	}
	if manifest.PackageListVersion != PackageListVersion {
		t.Fatalf("PackageListVersion = %d", manifest.PackageListVersion)
	}
}

func TestOnIsIdempotent(t *testing.T) {
	a := testAVD(t)
	device := newFakeDevice("emulator-5554",
		"com.google.android.googlequicksearchbox",
		"com.google.android.apps.photos",
	)

	if _, err := On(a, device, Options{}); err != nil {
		t.Fatal(err)
	}
	result, err := On(a, device, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Disabled) != 0 {
		t.Fatalf("second run disabled %v", result.Disabled)
	}
	if len(result.Settings) != 0 {
		t.Fatalf("second run changed settings %v", result.Settings)
	}

	manifest, _ := avd.LoadManifest(a)
	if len(manifest.Packages) != 2 {
		t.Fatalf("manifest packages = %+v, want 2 unique entries", manifest.Packages)
	}
}

func TestOnRespectsKeep(t *testing.T) {
	a := testAVD(t)
	device := newFakeDevice("emulator-5554",
		"com.google.android.googlequicksearchbox",
		"com.google.android.apps.photos",
		"com.google.android.apps.maps",
	)

	result, err := On(a, device, Options{Keep: []string{"com.google.android.apps.maps"}})
	if err != nil {
		t.Fatal(err)
	}
	if device.disabled["com.google.android.apps.maps"] {
		t.Error("--keep package was disabled")
	}
	if !contains(result.Kept, "com.google.android.apps.maps") {
		t.Errorf("Kept = %v", result.Kept)
	}
	if !device.disabled["com.google.android.apps.photos"] {
		t.Error("sibling media package was not disabled")
	}
}

func TestOnRespectsSkip(t *testing.T) {
	a := testAVD(t)
	device := newFakeDevice("emulator-5554",
		"com.google.android.googlequicksearchbox",
		"com.google.android.apps.photos",
		"com.google.android.apps.maps",
	)

	result, err := On(a, device, Options{Skip: []string{"media"}})
	if err != nil {
		t.Fatal(err)
	}
	if device.disabled["com.google.android.apps.photos"] || device.disabled["com.google.android.apps.maps"] {
		t.Error("--skip=media still disabled a media package")
	}
	if !contains(result.Disabled, "com.google.android.googlequicksearchbox") {
		t.Errorf("Disabled = %v, want the assistant package", result.Disabled)
	}
}

func TestOnReportsFailures(t *testing.T) {
	a := testAVD(t)
	device := newFakeDevice("emulator-5554",
		"com.google.android.googlequicksearchbox",
		"com.google.android.apps.photos",
	)
	device.failDisable["com.google.android.apps.photos"] = true
	device.failPut["global/background_process_limit"] = true

	result, err := On(a, device, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.FailedPackages) != 1 {
		t.Fatalf("FailedPackages = %v", result.FailedPackages)
	}
	if len(result.FailedSettings) != 1 {
		t.Fatalf("FailedSettings = %v", result.FailedSettings)
	}

	manifest, _ := avd.LoadManifest(a)
	if len(manifest.Packages) != 2 || len(manifest.Settings) != 2 {
		t.Fatalf("manifest does not keep the full restore plan: %+v", manifest)
	}
}

func TestOnRequiresDevice(t *testing.T) {
	a := testAVD(t)
	if _, err := On(a, nil, Options{}); err == nil {
		t.Fatal("On() succeeded without a device")
	}
}
