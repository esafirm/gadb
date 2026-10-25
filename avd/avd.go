package avd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// AVD is one installed Android Virtual Device.
type AVD struct {
	Name string
	Dir  string
}

// List returns installed AVDs sorted by name. A missing AVD directory is not
// an error, it just means no AVD has been created yet.
func List() ([]AVD, error) {
	base := BaseDir()
	if base == "" {
		return nil, fmt.Errorf("cannot determine the AVD directory")
	}

	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	avds := make([]AVD, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasSuffix(entry.Name(), ".avd") {
			continue
		}
		avds = append(avds, AVD{
			Name: strings.TrimSuffix(entry.Name(), ".avd"),
			Dir:  filepath.Join(base, entry.Name()),
		})
	}
	sort.Slice(avds, func(i, j int) bool {
		return strings.ToLower(avds[i].Name) < strings.ToLower(avds[j].Name)
	})
	return avds, nil
}

// Find resolves an AVD name case-insensitively.
func Find(name string) (AVD, error) {
	avds, err := List()
	if err != nil {
		return AVD{}, err
	}
	for _, candidate := range avds {
		if strings.EqualFold(candidate.Name, name) {
			return candidate, nil
		}
	}
	return AVD{}, fmt.Errorf("AVD %q not found in %s", name, BaseDir())
}

// ConfigPath is the emulator's config.ini.
func (a AVD) ConfigPath() string { return filepath.Join(a.Dir, "config.ini") }

// ManifestPath is gadb's reversible-state file for this AVD.
func (a AVD) ManifestPath() string { return filepath.Join(a.Dir, "gadb-slim.json") }

// HardwarePath is the generated runtime hardware file. It can hold stale RAM
// and GPU values, so tuning removes it and lets the emulator regenerate it.
func (a AVD) HardwarePath() string { return filepath.Join(a.Dir, "hardware-qemu.ini") }

// SnapshotDir is where the emulator stores a named snapshot.
func (a AVD) SnapshotDir(name string) string {
	return filepath.Join(a.Dir, "snapshots", name)
}

// ClearHardwareFile removes the generated runtime hardware file so the next
// launch uses the current config.ini and launch flags.
func (a AVD) ClearHardwareFile() {
	_ = os.Remove(a.HardwarePath())
}
