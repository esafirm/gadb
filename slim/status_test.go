package slim

import (
	"testing"

	"github.com/esafirm/gadb/avd"
)

func TestStatusClean(t *testing.T) {
	a := testAVD(t)
	device := newFakeDevice("emulator-5554",
		"com.google.android.googlequicksearchbox",
		"com.google.android.apps.photos",
	)

	if _, err := On(a, device, Options{}); err != nil {
		t.Fatal(err)
	}
	status, err := Status(a, device)
	if err != nil {
		t.Fatal(err)
	}

	if !status.Running || status.Serial != "emulator-5554" {
		t.Fatalf("runtime = %+v", status)
	}
	if status.Packages != 2 || status.Disabled != 2 {
		t.Fatalf("packages = %d, disabled = %d", status.Packages, status.Disabled)
	}
	if status.Settings != 2 || status.SettingsOK != 2 {
		t.Fatalf("settings = %d, ok = %d", status.Settings, status.SettingsOK)
	}
	if len(status.ReEnabled) != 0 || len(status.SettingsDrift) != 0 || len(status.ConfigDrift) != 0 {
		t.Fatalf("unexpected drift: %+v", status)
	}
}

func TestStatusDetectsDrift(t *testing.T) {
	a := testAVD(t)
	device := newFakeDevice("emulator-5554",
		"com.google.android.googlequicksearchbox",
		"com.google.android.apps.photos",
	)

	if _, err := On(a, device, Options{}); err != nil {
		t.Fatal(err)
	}
	delete(device.disabled, "com.google.android.apps.photos")
	if err := device.PutSetting("global", "background_process_limit", "20"); err != nil {
		t.Fatal(err)
	}

	status, err := Status(a, device)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(status.ReEnabled, "com.google.android.apps.photos") {
		t.Fatalf("ReEnabled = %v", status.ReEnabled)
	}
	if !contains(status.SettingsDrift, "global/background_process_limit") {
		t.Fatalf("SettingsDrift = %v", status.SettingsDrift)
	}
	if status.Disabled != 1 {
		t.Fatalf("Disabled = %d, want 1", status.Disabled)
	}
	if status.SettingsOK != 1 {
		t.Fatalf("SettingsOK = %d, want 1", status.SettingsOK)
	}
}

func TestStatusStopped(t *testing.T) {
	a := testAVD(t)
	device := newFakeDevice("emulator-5554",
		"com.google.android.googlequicksearchbox",
		"com.google.android.apps.photos",
	)

	if _, err := On(a, device, Options{}); err != nil {
		t.Fatal(err)
	}
	status, err := Status(a, nil)
	if err != nil {
		t.Fatal(err)
	}
	if status.Running {
		t.Fatal("Running = true")
	}
	if status.Packages != 2 || status.Settings != 2 {
		t.Fatalf("status = %+v", status)
	}
	if status.Disabled != 0 || status.SettingsOK != 0 {
		t.Fatalf("guest values were checked while stopped: %+v", status)
	}
}

func TestStatusNoManifest(t *testing.T) {
	a := testAVD(t)
	status, err := Status(a, nil)
	if err != nil {
		t.Fatal(err)
	}
	if status.Manifest != nil {
		t.Fatalf("Manifest = %+v, want nil", status.Manifest)
	}
}

func TestStatusConfigDrift(t *testing.T) {
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
	config, err := avd.LoadConfig(a.ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	config.Set("hw.ramSize", "4096")
	if err := config.Save(); err != nil {
		t.Fatal(err)
	}

	status, err := Status(a, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(status.ConfigDrift, "hw.ramSize") {
		t.Fatalf("ConfigDrift = %v", status.ConfigDrift)
	}
}
