package avd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// ManifestVersion is the on-disk schema version.
const ManifestVersion = 1

// Manifest records everything gadb changed on an AVD so it can be undone.
// It lives next to config.ini as gadb-slim.json.
type Manifest struct {
	Version            int                     `json:"version"`
	PackageListVersion int                     `json:"packageListVersion,omitempty"`
	AVD                string                  `json:"avd"`
	AppliedAt          string                  `json:"appliedAt,omitempty"`
	Image              *ImageRef               `json:"image,omitempty"`
	Config             map[string]ConfigChange `json:"config,omitempty"`
	Packages           []PackageChange         `json:"packages,omitempty"`
	Settings           []SettingChange         `json:"settings,omitempty"`
	Snapshot           *SnapshotRef            `json:"snapshot,omitempty"`
}

// ConfigChange is one config.ini key before and after tuning.
type ConfigChange struct {
	Original string `json:"original"`
	Tuned    string `json:"tuned"`
	Existed  bool   `json:"existed"`
}

// PackageChange is one guest package gadb disabled.
type PackageChange struct {
	Name       string `json:"name"`
	WasEnabled bool   `json:"wasEnabled"`
}

// SettingChange is one Android setting gadb changed.
type SettingChange struct {
	Namespace string  `json:"namespace"`
	Key       string  `json:"key"`
	Existed   bool    `json:"existed"`
	Original  *string `json:"original,omitempty"`
	Tuned     string  `json:"tuned"`
}

// ImageRef identifies the image a manifest was written against.
type ImageRef struct {
	SysDir     string `json:"sysDir,omitempty"`
	API        string `json:"api,omitempty"`
	APIMajor   int    `json:"apiMajor,omitempty"`
	ABI        string `json:"abi,omitempty"`
	TagID      string `json:"tagId,omitempty"`
	PageSizeKB int    `json:"pageSizeKB,omitempty"`
	Revision   string `json:"revision,omitempty"`
}

// SnapshotRef identifies the golden snapshot tied to this AVD.
type SnapshotRef struct {
	Name               string `json:"name"`
	CreatedAt          string `json:"createdAt"`
	ImageFingerprint   string `json:"imageFingerprint,omitempty"`
	ConfigHash         string `json:"configHash,omitempty"`
	PackageListVersion int    `json:"packageListVersion,omitempty"`
}

// LoadManifest returns nil when the AVD has never been tuned.
func LoadManifest(a AVD) (*Manifest, error) {
	data, err := os.ReadFile(a.ManifestPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	manifest := &Manifest{}
	if err := json.Unmarshal(data, manifest); err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", a.ManifestPath(), err)
	}
	if manifest.Version > ManifestVersion {
		return nil, fmt.Errorf(
			"%s was written by a newer gadb (schema %d, this build understands %d)",
			a.ManifestPath(), manifest.Version, ManifestVersion,
		)
	}
	if manifest.Version == 0 {
		manifest.Version = ManifestVersion
	}
	return manifest, nil
}

// SaveManifest writes the manifest atomically.
func SaveManifest(a AVD, manifest *Manifest) error {
	if manifest.Version == 0 {
		manifest.Version = ManifestVersion
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(a.ManifestPath(), append(data, '\n'), 0o644)
}

// RemoveManifest deletes the manifest, ignoring a missing file.
func RemoveManifest(a AVD) error {
	err := os.Remove(a.ManifestPath())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (m *Manifest) touch() {
	m.Version = ManifestVersion
	m.AppliedAt = time.Now().UTC().Format(time.RFC3339)
}

// Empty reports whether the manifest has no reversible state left.
func (m *Manifest) Empty() bool {
	return len(m.Config) == 0 && len(m.Packages) == 0 && len(m.Settings) == 0 && m.Snapshot == nil
}

func imageRef(config *Config) *ImageRef {
	image, err := ImageForAVD(config)
	if err != nil {
		return nil
	}
	return &ImageRef{
		SysDir:     image.Dir,
		API:        image.API,
		APIMajor:   image.APIMajor,
		ABI:        image.ABI,
		TagID:      image.TagID,
		PageSizeKB: image.PageSizeKB,
		Revision:   image.Revision,
	}
}
