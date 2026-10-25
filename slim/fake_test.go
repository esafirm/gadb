package slim

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/esafirm/gadb/avd"
)

type fakeDevice struct {
	serial      string
	installed   map[string]bool
	disabled    map[string]bool
	settings    map[string]*string
	failDisable map[string]bool
	failEnable  map[string]bool
	failPut     map[string]bool
}

func newFakeDevice(serial string, installed ...string) *fakeDevice {
	device := &fakeDevice{
		serial:      serial,
		installed:   make(map[string]bool),
		disabled:    make(map[string]bool),
		settings:    make(map[string]*string),
		failDisable: make(map[string]bool),
		failEnable:  make(map[string]bool),
		failPut:     make(map[string]bool),
	}
	for _, pkg := range installed {
		device.installed[pkg] = true
	}
	return device
}

func cloneSet(input map[string]bool) map[string]bool {
	out := make(map[string]bool, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func (d *fakeDevice) Serial() string { return d.serial }

func (d *fakeDevice) InstalledPackages() (map[string]bool, error) { return cloneSet(d.installed), nil }

func (d *fakeDevice) DisabledPackages() (map[string]bool, error) { return cloneSet(d.disabled), nil }

func (d *fakeDevice) DisablePackage(pkg string) error {
	if d.failDisable[pkg] {
		return fmt.Errorf("cannot disable %s", pkg)
	}
	d.disabled[pkg] = true
	return nil
}

func (d *fakeDevice) EnablePackage(pkg string) error {
	if d.failEnable[pkg] {
		return fmt.Errorf("cannot enable %s", pkg)
	}
	delete(d.disabled, pkg)
	return nil
}

func (d *fakeDevice) GetSetting(namespace, key string) (*string, error) {
	value := d.settings[namespace+"/"+key]
	if value == nil {
		return nil, nil
	}
	copied := *value
	return &copied, nil
}

func (d *fakeDevice) PutSetting(namespace, key, value string) error {
	if d.failPut[namespace+"/"+key] {
		return fmt.Errorf("cannot put %s/%s", namespace, key)
	}
	copied := value
	d.settings[namespace+"/"+key] = &copied
	return nil
}

func (d *fakeDevice) DeleteSetting(namespace, key string) error {
	delete(d.settings, namespace+"/"+key)
	return nil
}

func (d *fakeDevice) TrimMemory() {}

func testAVD(t *testing.T) avd.AVD {
	t.Helper()

	base := t.TempDir()
	t.Setenv("ANDROID_AVD_HOME", base)
	dir := filepath.Join(base, "Test.avd")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	config := "hw.ramSize=2048\nhw.gpu.mode=host\n"
	if err := os.WriteFile(filepath.Join(dir, "config.ini"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	return avd.AVD{Name: "Test", Dir: dir}
}

func set(items ...string) map[string]bool {
	out := make(map[string]bool, len(items))
	for _, item := range items {
		out[item] = true
	}
	return out
}

func assertSet(t *testing.T, got []string, want map[string]bool) {
	t.Helper()

	gotSet := make(map[string]bool, len(got))
	for _, item := range got {
		gotSet[item] = true
	}
	if len(gotSet) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for item := range want {
		if !gotSet[item] {
			t.Fatalf("missing %s in %v", item, got)
		}
	}
}
