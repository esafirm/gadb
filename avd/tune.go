package avd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

const (
	// DefaultRAM is the conservative guest RAM size for API 30+ images.
	DefaultRAM = 2048
	// BackupName is the untouched config.ini kept next to the live file.
	BackupName = "config.ini.gadb.bak"
)

// TuneOptions controls the host-side config.ini changes.
type TuneOptions struct {
	RAM        int
	GPU        string
	KeepAudio  bool
	KeepCamera bool
}

// TuneChange is one config.ini value that changed.
type TuneChange struct {
	Key     string
	From    string
	To      string
	Existed bool
}

// TuneResult reports what tune changed.
type TuneResult struct {
	AVD             AVD
	Image           *Image
	Changes         []TuneChange
	Skipped         []string
	Backup          string
	RemovedHardware bool
	Warnings        []string
}

// RestoreResult reports what a tune --restore put back.
type RestoreResult struct {
	AVD             AVD
	Restored        []TuneChange
	Skipped         []string
	RemovedManifest bool
}

// Tune writes config.ini for a stopped AVD and records the original values in
// the manifest. It never touches the guest.
func Tune(a AVD, opts TuneOptions) (*TuneResult, error) {
	if serial, running := runningEmulator(a.Name); running {
		return nil, fmt.Errorf("AVD %q is running (%s). Stop it first: adb -s %s emu kill", a.Name, serial, serial)
	}

	if opts.RAM <= 0 {
		opts.RAM = DefaultRAM
	}
	if opts.GPU == "" {
		opts.GPU = DefaultGPU()
	}

	config, err := LoadConfig(a.ConfigPath())
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", a.ConfigPath(), err)
	}

	manifest, err := LoadManifest(a)
	if err != nil {
		return nil, err
	}
	if manifest == nil {
		manifest = &Manifest{AVD: a.Name, Config: make(map[string]ConfigChange)}
	}
	if manifest.Config == nil {
		manifest.Config = make(map[string]ConfigChange)
	}

	backupPath := filepath.Join(a.Dir, BackupName)
	var backupConfig *Config
	if _, err := os.Stat(backupPath); err == nil {
		backupConfig, _ = LoadConfig(backupPath)
	} else {
		if err := os.WriteFile(backupPath, config.Bytes(), 0o644); err != nil {
			return nil, fmt.Errorf("cannot write %s: %w", backupPath, err)
		}
	}

	result := &TuneResult{
		AVD:    a,
		Backup: backupPath,
	}

	// --keep-audio and --keep-camera must undo a previous tune for those keys,
	// otherwise the flag would silently keep the old disabled values.
	if opts.KeepAudio {
		restored, skipped := restoreKeys(config, manifest, []string{"hw.audioInput", "hw.audioOutput"})
		result.Changes = append(result.Changes, restored...)
		result.Skipped = append(result.Skipped, skipped...)
	}
	if opts.KeepCamera {
		restored, skipped := restoreKeys(config, manifest, []string{"hw.camera.back", "hw.camera.front"})
		result.Changes = append(result.Changes, restored...)
		result.Skipped = append(result.Skipped, skipped...)
	}

	for _, item := range tuneValues(opts) {
		current, currentExists := config.Get(item.key)
		_, recorded := manifest.Config[item.key]

		// A value that is already correct and was never tuned needs no
		// manifest entry, so --restore stays quiet.
		if currentExists && current == item.value && !recorded {
			result.Changes = append(result.Changes, TuneChange{
				Key: item.key, From: current, To: item.value, Existed: true,
			})
			continue
		}

		old, existed := config.Set(item.key, item.value)

		if !recorded {
			original, originalExists := old, existed
			if backupConfig != nil {
				original, originalExists = backupConfig.Get(item.key)
			}
			manifest.Config[item.key] = ConfigChange{
				Original: original,
				Tuned:    item.value,
				Existed:  originalExists,
			}
		} else {
			change := manifest.Config[item.key]
			change.Tuned = item.value
			manifest.Config[item.key] = change
		}

		result.Changes = append(result.Changes, TuneChange{
			Key:     item.key,
			From:    current,
			To:      item.value,
			Existed: currentExists,
		})
	}

	if err := config.Save(); err != nil {
		return nil, fmt.Errorf("cannot write %s: %w", a.ConfigPath(), err)
	}

	if err := os.Remove(a.HardwarePath()); err == nil {
		result.RemovedHardware = true
	} else if !os.IsNotExist(err) {
		result.Warnings = append(result.Warnings, "could not remove hardware-qemu.ini: "+err.Error())
	}

	manifest.touch()
	manifest.AVD = a.Name
	manifest.Image = imageRef(config)
	if err := SaveManifest(a, manifest); err != nil {
		return nil, fmt.Errorf("config.ini was updated but the manifest could not be saved: %w", err)
	}

	result.Image, _ = ImageForAVD(config)
	if result.Image != nil && result.Image.PageSizeKB == 16 {
		result.Warnings = append(result.Warnings,
			"16 KB image: the guest still reserves 4096 MB, hw.ramSize is ignored")
	}
	if result.Image == nil {
		result.Warnings = append(result.Warnings, "could not inspect the system image")
	}
	if opts.RAM < DefaultRAM && (result.Image == nil || result.Image.APIMajor >= 30) {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("%d MB is aggressive for modern images, expect more app kills", opts.RAM))
	}

	return result, nil
}

// RestoreConfig puts back every config.ini value recorded in the manifest. It
// leaves values the user changed after tuning untouched and reports them.
func RestoreConfig(a AVD) (*RestoreResult, error) {
	if serial, running := runningEmulator(a.Name); running {
		return nil, fmt.Errorf("AVD %q is running (%s). Stop it first: adb -s %s emu kill", a.Name, serial, serial)
	}

	manifest, err := LoadManifest(a)
	if err != nil {
		return nil, err
	}
	if manifest == nil {
		return nil, fmt.Errorf(
			"no gadb manifest for %q, nothing to restore (backup: %s)",
			a.Name, filepath.Join(a.Dir, BackupName),
		)
	}

	result, err := RestoreConfigEntries(a, manifest, false)
	if err != nil {
		return nil, err
	}

	if manifest.Empty() {
		if err := RemoveManifest(a); err != nil {
			return nil, err
		}
		result.RemovedManifest = true
	} else {
		if err := SaveManifest(a, manifest); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// RestoreConfigEntries restores the config.ini values recorded in manifest and
// removes those entries from it. The caller owns saving or removing the
// manifest. When running is true, the generated hardware-qemu.ini is left
// alone so a live emulator is not disturbed.
func RestoreConfigEntries(a AVD, manifest *Manifest, running bool) (*RestoreResult, error) {
	config, err := LoadConfig(a.ConfigPath())
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", a.ConfigPath(), err)
	}

	result := &RestoreResult{AVD: a}
	keys := make([]string, 0, len(manifest.Config))
	for key := range manifest.Config {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		change := manifest.Config[key]
		current, exists := config.Get(key)

		if !exists {
			if !change.Existed {
				// The key was added by gadb and is already gone again.
				delete(manifest.Config, key)
				continue
			}
			result.Skipped = append(result.Skipped, key+" (missing since tuning)")
			continue
		}

		if current != change.Tuned {
			if change.Existed && current == change.Original {
				// Already back at the original value.
				delete(manifest.Config, key)
				continue
			}
			result.Skipped = append(result.Skipped, key+" (changed since tuning)")
			continue
		}

		if change.Existed {
			config.Set(key, change.Original)
			result.Restored = append(result.Restored, TuneChange{
				Key: key, From: current, To: change.Original, Existed: true,
			})
		} else {
			config.Delete(key)
			result.Restored = append(result.Restored, TuneChange{
				Key: key, From: current, To: "", Existed: true,
			})
		}
		delete(manifest.Config, key)
	}

	if len(result.Restored) > 0 {
		if err := config.Save(); err != nil {
			return nil, fmt.Errorf("cannot write %s: %w", a.ConfigPath(), err)
		}
	}

	if !running {
		_ = os.Remove(a.HardwarePath()) // regenerated by the emulator on the next boot
	}

	return result, nil
}

type tuneValue struct {
	key   string
	value string
}

func tuneValues(opts TuneOptions) []tuneValue {
	values := []tuneValue{
		{"hw.ramSize", strconv.Itoa(opts.RAM)},
		{"hw.gpu.mode", opts.GPU},
		{"hw.gpu.enabled", "yes"},
		{"hw.keyboard", "yes"},
	}
	if !opts.KeepAudio {
		values = append(values,
			tuneValue{"hw.audioInput", "no"},
			tuneValue{"hw.audioOutput", "no"},
		)
	}
	if !opts.KeepCamera {
		values = append(values,
			tuneValue{"hw.camera.back", "none"},
			tuneValue{"hw.camera.front", "none"},
		)
	}
	return values
}

// restoreKeys reverts only the listed keys, used by --keep-audio/--keep-camera.
func restoreKeys(config *Config, manifest *Manifest, keys []string) (restored []TuneChange, skipped []string) {
	for _, key := range keys {
		change, recorded := manifest.Config[key]
		if !recorded {
			continue
		}

		current, exists := config.Get(key)
		if !exists || current != change.Tuned {
			skipped = append(skipped, key+" (changed since tuning)")
			continue
		}

		if change.Existed {
			config.Set(key, change.Original)
			restored = append(restored, TuneChange{Key: key, From: current, To: change.Original, Existed: true})
		} else {
			config.Delete(key)
			restored = append(restored, TuneChange{Key: key, From: current, To: "", Existed: true})
		}
		delete(manifest.Config, key)
	}
	return restored, skipped
}
