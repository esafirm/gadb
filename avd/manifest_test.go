package avd

import (
	"os"
	"path/filepath"
	"testing"
)

func testAVD(t *testing.T) AVD {
	t.Helper()

	base := t.TempDir()
	t.Setenv("ANDROID_AVD_HOME", base)
	dir := filepath.Join(base, "Test.avd")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return AVD{Name: "Test", Dir: dir}
}

func TestManifestRoundTrip(t *testing.T) {
	a := testAVD(t)

	if manifest, err := LoadManifest(a); err != nil || manifest != nil {
		t.Fatalf("LoadManifest() = %+v, %v, want nil, nil", manifest, err)
	}

	manifest := &Manifest{
		Version: 1,
		AVD:     a.Name,
		Config: map[string]ConfigChange{
			"hw.ramSize": {Original: "1536", Tuned: "2048", Existed: true},
		},
		Packages: []PackageChange{{Name: "com.example.app", WasEnabled: true}},
	}
	if err := SaveManifest(a, manifest); err != nil {
		t.Fatal(err)
	}

	got, err := LoadManifest(a)
	if err != nil {
		t.Fatal(err)
	}
	if got.AVD != a.Name {
		t.Fatalf("AVD = %q, want %q", got.AVD, a.Name)
	}
	if got.Config["hw.ramSize"].Original != "1536" {
		t.Fatalf("ram change = %+v", got.Config["hw.ramSize"])
	}
	if len(got.Packages) != 1 || got.Packages[0].Name != "com.example.app" {
		t.Fatalf("packages = %+v", got.Packages)
	}

	if err := RemoveManifest(a); err != nil {
		t.Fatal(err)
	}
	if manifest, err := LoadManifest(a); err != nil || manifest != nil {
		t.Fatalf("after RemoveManifest: %+v, %v", manifest, err)
	}
}

func TestManifestRejectsNewerSchema(t *testing.T) {
	a := testAVD(t)
	if err := os.WriteFile(a.ManifestPath(), []byte(`{"version":99}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadManifest(a); err == nil {
		t.Fatal("expected an error for a newer manifest schema")
	}
}
