package avd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// CleanSnapshotName is the named snapshot `gadb slim snapshot` writes and
// `gadb slim start` restores.
const CleanSnapshotName = "gadb_clean"

// snapshotConfigKeys are the config.ini values a golden snapshot depends on.
// Changing any of them invalidates the snapshot.
var snapshotConfigKeys = []string{
	"hw.ramSize",
	"hw.gpu.mode",
	"hw.gpu.enabled",
	"hw.audioInput",
	"hw.audioOutput",
	"hw.camera.back",
	"hw.camera.front",
	"hw.keyboard",
}

// SnapshotStatus describes the golden snapshot and whether it still matches the
// current image and config.
type SnapshotStatus struct {
	Name      string
	Exists    bool
	SizeBytes int64
	Valid     bool
	Reason    string
	CreatedAt string
}

// HasSnapshot reports whether the named snapshot directory exists.
func (a AVD) HasSnapshot(name string) bool {
	return dirExists(a.SnapshotDir(name))
}

// ImageFingerprint changes when the system image package or revision changes.
func ImageFingerprint(image *Image) string {
	if image == nil {
		return ""
	}
	hash := sha256.New()
	fmt.Fprintf(hash, "%s|%s|%d|%s|%s\n",
		image.PkgID, image.Revision, image.PageSizeKB, image.API, image.ABI)
	return hex.EncodeToString(hash.Sum(nil))
}

// ConfigHash changes when any snapshot-relevant config.ini value changes.
func ConfigHash(config *Config) string {
	if config == nil {
		return ""
	}
	hash := sha256.New()
	for _, key := range snapshotConfigKeys {
		value, _ := config.Get(key)
		fmt.Fprintf(hash, "%s=%s\n", key, value)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// CheckSnapshot compares the snapshot on disk with the manifest and the current
// image/config. A missing or stale snapshot is reported, never deleted.
func CheckSnapshot(a AVD, manifest *Manifest) *SnapshotStatus {
	status := &SnapshotStatus{Name: CleanSnapshotName}
	status.Exists = a.HasSnapshot(CleanSnapshotName)
	if !status.Exists {
		status.Reason = "not found"
		return status
	}

	status.SizeBytes, _ = SnapshotSize(a, CleanSnapshotName)
	if manifest == nil || manifest.Snapshot == nil {
		status.Reason = "not recorded in gadb-slim.json"
		return status
	}
	status.CreatedAt = manifest.Snapshot.CreatedAt

	config, err := LoadConfig(a.ConfigPath())
	if err != nil {
		status.Reason = "cannot read config.ini"
		return status
	}
	image, err := ImageForAVD(config)
	if err != nil || image == nil {
		status.Reason = "cannot inspect the system image"
		return status
	}
	if manifest.Snapshot.ImageFingerprint == "" ||
		manifest.Snapshot.ImageFingerprint != ImageFingerprint(image) {
		status.Reason = "the system image changed"
		return status
	}
	if manifest.Snapshot.ConfigHash == "" ||
		manifest.Snapshot.ConfigHash != ConfigHash(config) {
		status.Reason = "config.ini changed"
		return status
	}

	status.Valid = true
	return status
}

// SnapshotSize sums the files in a snapshot directory.
func SnapshotSize(a AVD, name string) (int64, error) {
	var size int64
	err := filepath.WalkDir(a.SnapshotDir(name), func(_ string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		size += info.Size()
		return nil
	})
	return size, err
}

// DeleteSnapshot removes gadb's named snapshot. The name is checked so a
// malformed manifest can never delete an arbitrary directory.
func DeleteSnapshot(a AVD, name string) error {
	if name != CleanSnapshotName {
		return fmt.Errorf("refusing to delete snapshot %q", name)
	}

	root := filepath.Join(a.Dir, "snapshots")
	target := filepath.Join(root, name)
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == "." || relative == ".." ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("refusing to delete %s", target)
	}
	return os.RemoveAll(target)
}

// HumanSize formats a byte count for CLI output.
func HumanSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for value := bytes / unit; value >= unit; value /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
