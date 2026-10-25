// Package slim applies reversible memory tuning to a running emulator:
// package pruning, a small set of settings, and cache trimming.
package slim

import "github.com/esafirm/gadb/avd"

// PackageListVersion changes whenever the curated package list changes.
const PackageListVersion = 1

// Options controls On.
type Options struct {
	// Aggressive adds the Play Store, Chrome and setup packages.
	Aggressive bool
	// Keep lists exact package names that must not be disabled.
	Keep []string
	// Skip lists group names to leave alone, for packages and settings.
	Skip []string
	// NoAnim sets the three animation scales to 0. They are untouched by default.
	NoAnim bool
}

// OnResult reports what On changed.
type OnResult struct {
	AVD            avd.AVD
	Serial         string
	Disabled       []string
	Kept           []string
	FailedPackages []string
	FailedSettings []string
	Settings       []avd.SettingChange
	Warnings       []string
	Trimmed        bool
}

// OffResult reports what Off restored.
type OffResult struct {
	AVD              avd.AVD
	Serial           string
	Running          bool
	RestoredPackages []string
	RestoredSettings []string
	FailedPackages   []string
	FailedSettings   []string
	Config           *avd.RestoreResult
	RemovedManifest  bool
	Warnings         []string
}

// StatusReport is a read-only view of a slimmed AVD.
type StatusReport struct {
	AVD            avd.AVD
	Serial         string
	Running        bool
	Manifest       *avd.Manifest
	Packages       int
	Disabled       int
	ReEnabled      []string
	Missing        []string
	Settings       int
	SettingsOK     int
	SettingsDrift  []string
	ConfigDrift    []string
	Snapshot       string
	SnapshotValid  bool
	SnapshotReason string
	OlderList      bool
}
