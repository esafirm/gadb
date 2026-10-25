package slim

import (
	"testing"

	"github.com/esafirm/gadb/avd"
)

func ptr(value string) *string { return &value }

func TestMergeConstants(t *testing.T) {
	cases := []struct {
		name    string
		current *string
		want    string
		changed bool
	}{
		{"unset", nil, "max_cached_processes=4", true},
		{"empty", ptr(""), "max_cached_processes=4", true},
		{"same", ptr("max_cached_processes=4"), "max_cached_processes=4", false},
		{"replace", ptr("max_cached_processes=8"), "max_cached_processes=4", true},
		{"preserve", ptr("foo=bar,max_cached_processes=8"), "foo=bar,max_cached_processes=4", true},
		{"append", ptr("foo=bar"), "foo=bar,max_cached_processes=4", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, changed := mergeConstants(tc.current, "max_cached_processes", "4")
			if got != tc.want || changed != tc.changed {
				t.Fatalf("mergeConstants() = %q, %v, want %q, %v", got, changed, tc.want, tc.changed)
			}
		})
	}
}

func TestPlanSettings(t *testing.T) {
	device := newFakeDevice("emulator-5554")
	manifest := &avd.Manifest{Version: 1, AVD: "Test"}

	plan, warnings := planSettings(device, manifest, Options{})
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if len(plan) != 2 {
		t.Fatalf("plan = %+v, want 2 entries", plan)
	}
	if len(manifest.Settings) != 2 {
		t.Fatalf("manifest settings = %+v, want 2 entries", manifest.Settings)
	}
	if plan[0].Key != "background_process_limit" || plan[0].Value != "4" {
		t.Fatalf("first setting = %+v", plan[0])
	}
	if plan[1].Key != "activity_manager_constants" || plan[1].Value != "max_cached_processes=4" {
		t.Fatalf("second setting = %+v", plan[1])
	}

	for _, item := range plan {
		if err := device.PutSetting(item.Namespace, item.Key, item.Value); err != nil {
			t.Fatal(err)
		}
	}
	plan, _ = planSettings(device, manifest, Options{})
	if len(plan) != 0 {
		t.Fatalf("second plan = %+v, want no changes", plan)
	}
}

func TestPlanSettingsSkip(t *testing.T) {
	device := newFakeDevice("emulator-5554")
	manifest := &avd.Manifest{Version: 1, AVD: "Test"}

	plan, _ := planSettings(device, manifest, Options{Skip: []string{"bglimit"}})
	if len(plan) != 0 {
		t.Fatalf("plan = %+v, want none for --skip=bglimit", plan)
	}
}

func TestPlanSettingsNoAnim(t *testing.T) {
	device := newFakeDevice("emulator-5554")
	manifest := &avd.Manifest{Version: 1, AVD: "Test"}

	plan, _ := planSettings(device, manifest, Options{NoAnim: true})
	if len(plan) != 5 {
		t.Fatalf("plan = %+v, want 2 settings plus 3 animation scales", plan)
	}
}

func TestPlanSettingsMergesConstants(t *testing.T) {
	device := newFakeDevice("emulator-5554")
	device.settings["global/activity_manager_constants"] = ptr("foo=bar")
	manifest := &avd.Manifest{Version: 1, AVD: "Test"}

	plan, _ := planSettings(device, manifest, Options{})
	var merged *plannedSetting
	for i := range plan {
		if plan[i].Key == "activity_manager_constants" {
			merged = &plan[i]
		}
	}
	if merged == nil {
		t.Fatal("activity_manager_constants was not planned")
	}
	if merged.Value != "foo=bar,max_cached_processes=4" {
		t.Fatalf("merged value = %q", merged.Value)
	}

	var recorded *avd.SettingChange
	for i := range manifest.Settings {
		if manifest.Settings[i].Key == "activity_manager_constants" {
			recorded = &manifest.Settings[i]
		}
	}
	if recorded == nil || recorded.Original == nil || *recorded.Original != "foo=bar" {
		t.Fatalf("recorded setting = %+v", recorded)
	}
}
