package avd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSnapshot(t *testing.T, a AVD) {
	t.Helper()

	dir := a.SnapshotDir(CleanSnapshotName)
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ram.img"), []byte("12345"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "state.img"), []byte("1234567"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func snapshotFixture(t *testing.T) (AVD, *Manifest) {
	t.Helper()

	sdk := useSDK(t)
	abi := HostABI()
	writeImage(t, sdk, "android-35", "google_apis", abi, map[string]string{
		"AndroidVersion.ApiLevel": "35.0",
		"SystemImage.Abi":         abi,
		"SystemImage.TagId":       "google_apis",
		"Pkg.Revision":            "9",
	}, true)

	a := testAVDWithConfig(t, "hw.ramSize=2048\nhw.gpu.mode=host\nhw.audioOutput=no\n"+
		"image.sysdir.1=system-images/android-35/google_apis/"+abi+"/\n")
	config, err := LoadConfig(a.ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	image, err := ImageForAVD(config)
	if err != nil {
		t.Fatal(err)
	}

	return a, &Manifest{
		Version: 1,
		AVD:     a.Name,
		Snapshot: &SnapshotRef{
			Name:             CleanSnapshotName,
			ImageFingerprint: ImageFingerprint(image),
			ConfigHash:       ConfigHash(config),
		},
	}
}

func TestCheckSnapshotValid(t *testing.T) {
	a, manifest := snapshotFixture(t)
	writeSnapshot(t, a)

	status := CheckSnapshot(a, manifest)
	if !status.Valid || !status.Exists {
		t.Fatalf("status = %+v, want a valid snapshot", status)
	}
	if status.SizeBytes != 12 {
		t.Fatalf("SizeBytes = %d, want 12", status.SizeBytes)
	}
}

func TestCheckSnapshotMissing(t *testing.T) {
	a, manifest := snapshotFixture(t)

	status := CheckSnapshot(a, manifest)
	if status.Exists || status.Valid {
		t.Fatalf("status = %+v, want missing", status)
	}
	if status.Reason != "not found" {
		t.Fatalf("Reason = %q", status.Reason)
	}
}

func TestCheckSnapshotNotRecorded(t *testing.T) {
	a, _ := snapshotFixture(t)
	writeSnapshot(t, a)

	status := CheckSnapshot(a, nil)
	if status.Valid || !status.Exists {
		t.Fatalf("status = %+v", status)
	}
	if !strings.Contains(status.Reason, "not recorded") {
		t.Fatalf("Reason = %q", status.Reason)
	}
}

func TestCheckSnapshotImageChanged(t *testing.T) {
	a, manifest := snapshotFixture(t)
	writeSnapshot(t, a)
	manifest.Snapshot.ImageFingerprint = "old-image"

	status := CheckSnapshot(a, manifest)
	if status.Valid {
		t.Fatal("snapshot considered valid after the image changed")
	}
	if !strings.Contains(status.Reason, "image changed") {
		t.Fatalf("Reason = %q", status.Reason)
	}
}

func TestCheckSnapshotConfigChanged(t *testing.T) {
	a, manifest := snapshotFixture(t)
	writeSnapshot(t, a)
	manifest.Snapshot.ConfigHash = "old-config"

	status := CheckSnapshot(a, manifest)
	if status.Valid {
		t.Fatal("snapshot considered valid after config.ini changed")
	}
	if !strings.Contains(status.Reason, "config.ini") {
		t.Fatalf("Reason = %q", status.Reason)
	}
}

func TestImageFingerprintChanges(t *testing.T) {
	base := &Image{PkgID: "system-images;android-35;google_apis;arm64-v8a", Revision: "9", PageSizeKB: 4, API: "35", ABI: "arm64-v8a"}
	same := *base
	if ImageFingerprint(base) != ImageFingerprint(&same) {
		t.Fatal("identical images have different fingerprints")
	}
	for name, mutate := range map[string]func(*Image){
		"revision":  func(i *Image) { i.Revision = "10" },
		"page size": func(i *Image) { i.PageSizeKB = 16 },
		"api":       func(i *Image) { i.API = "36" },
		"abi":       func(i *Image) { i.ABI = "x86_64" },
	} {
		changed := *base
		mutate(&changed)
		if ImageFingerprint(base) == ImageFingerprint(&changed) {
			t.Errorf("%s change did not change the fingerprint", name)
		}
	}
}

func TestConfigHashChanges(t *testing.T) {
	base := &Config{lines: []string{"hw.ramSize=2048", "hw.gpu.mode=host"}}
	same := &Config{lines: []string{"hw.ramSize=2048", "hw.gpu.mode=host"}}
	if ConfigHash(base) != ConfigHash(same) {
		t.Fatal("identical configs have different hashes")
	}
	changed := &Config{lines: []string{"hw.ramSize=1536", "hw.gpu.mode=host"}}
	if ConfigHash(base) == ConfigHash(changed) {
		t.Fatal("RAM change did not change the config hash")
	}
}

func TestDeleteSnapshot(t *testing.T) {
	a, _ := snapshotFixture(t)
	writeSnapshot(t, a)

	if err := DeleteSnapshot(a, CleanSnapshotName); err != nil {
		t.Fatal(err)
	}
	if a.HasSnapshot(CleanSnapshotName) {
		t.Fatal("snapshot still exists")
	}
}

func TestDeleteSnapshotRefusesOtherNames(t *testing.T) {
	a, _ := snapshotFixture(t)
	if err := DeleteSnapshot(a, "default_boot"); err == nil {
		t.Fatal("DeleteSnapshot accepted a name other than gadb_clean")
	}
}

func TestHumanSize(t *testing.T) {
	cases := map[int64]string{
		512:         "512 B",
		1024:        "1.0 KB",
		1024 * 1024: "1.0 MB",
	}
	for bytes, want := range cases {
		if got := HumanSize(bytes); got != want {
			t.Errorf("HumanSize(%d) = %q, want %q", bytes, got, want)
		}
	}
}
