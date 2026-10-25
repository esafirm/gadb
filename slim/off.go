package slim

import (
	"fmt"

	"github.com/esafirm/gadb/avd"
)

// Off restores packages and settings when dev is supplied, and always restores
// the host config.ini values. Failed packages and settings stay in the manifest
// so Off can be re-run.
func Off(a avd.AVD, dev Device, running bool) (*OffResult, error) {
	manifest, err := avd.LoadManifest(a)
	if err != nil {
		return nil, err
	}
	if manifest == nil {
		return nil, fmt.Errorf("nothing to restore for %q", a.Name)
	}

	result := &OffResult{AVD: a, Running: running}
	if running {
		if dev == nil {
			return nil, fmt.Errorf("a running emulator is required to restore packages and settings")
		}
		result.Serial = dev.Serial()

		installed, err := dev.InstalledPackages()
		if err != nil {
			return nil, fmt.Errorf("cannot list installed packages: %w", err)
		}

		keptPackages := manifest.Packages[:0]
		for _, change := range manifest.Packages {
			if !installed[change.Name] {
				result.Warnings = append(result.Warnings, change.Name+" is no longer installed")
				continue
			}
			if err := dev.EnablePackage(change.Name); err != nil {
				result.FailedPackages = append(result.FailedPackages, err.Error())
				keptPackages = append(keptPackages, change)
				continue
			}
			result.RestoredPackages = append(result.RestoredPackages, change.Name)
		}
		manifest.Packages = keptPackages

		keptSettings := manifest.Settings[:0]
		for _, setting := range manifest.Settings {
			if err := restoreSetting(dev, setting); err != nil {
				result.FailedSettings = append(result.FailedSettings,
					fmt.Sprintf("%s/%s: %v", setting.Namespace, setting.Key, err))
				keptSettings = append(keptSettings, setting)
				continue
			}
			result.RestoredSettings = append(result.RestoredSettings,
				setting.Namespace+"/"+setting.Key)
		}
		manifest.Settings = keptSettings
	} else {
		result.Warnings = append(result.Warnings,
			"the emulator is not running, packages and settings were not restored")
	}

	configResult, err := avd.RestoreConfigEntries(a, manifest, running)
	if err != nil {
		return nil, err
	}
	result.Config = configResult
	if running && len(configResult.Restored) > 0 {
		result.Warnings = append(result.Warnings,
			"config.ini was restored, restart the emulator for it to take effect")
	}

	if manifest.Snapshot != nil {
		result.Warnings = append(result.Warnings,
			"the "+manifest.Snapshot.Name+" snapshot still has the slim state; remove it with gadb slim unbake "+a.Name)
	}

	if manifest.Empty() {
		if err := avd.RemoveManifest(a); err != nil {
			return nil, err
		}
		result.RemovedManifest = true
	} else {
		if err := avd.SaveManifest(a, manifest); err != nil {
			return nil, err
		}
	}

	return result, nil
}
