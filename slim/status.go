package slim

import (
	"sort"

	"github.com/esafirm/gadb/avd"
)

// Status reports what a manifest says should be applied and, when an emulator
// is running, whether it still matches. It never changes anything.
func Status(a avd.AVD, dev Device) (*StatusReport, error) {
	status := &StatusReport{AVD: a}
	if dev != nil {
		status.Serial = dev.Serial()
		status.Running = true
	}

	manifest, err := avd.LoadManifest(a)
	if err != nil {
		return nil, err
	}
	status.Manifest = manifest
	if manifest == nil {
		return status, nil
	}

	status.Packages = len(manifest.Packages)
	status.Settings = len(manifest.Settings)
	status.OlderList = manifest.PackageListVersion != 0 &&
		manifest.PackageListVersion < PackageListVersion
	snapshot := avd.CheckSnapshot(a, manifest)
	if snapshot.Exists {
		status.Snapshot = snapshot.Name
		status.SnapshotValid = snapshot.Valid
		status.SnapshotReason = snapshot.Reason
	}

	if config, err := avd.LoadConfig(a.ConfigPath()); err == nil {
		for key, change := range manifest.Config {
			current, exists := config.Get(key)
			if !exists || current != change.Tuned {
				status.ConfigDrift = append(status.ConfigDrift, key)
			}
		}
		sort.Strings(status.ConfigDrift)
	}

	if dev == nil {
		return status, nil
	}

	disabled, err := dev.DisabledPackages()
	if err != nil {
		return nil, err
	}
	installed, err := dev.InstalledPackages()
	if err != nil {
		return nil, err
	}

	for _, change := range manifest.Packages {
		if !installed[change.Name] {
			status.Missing = append(status.Missing, change.Name)
			continue
		}
		if !disabled[change.Name] {
			status.ReEnabled = append(status.ReEnabled, change.Name)
		}
	}
	status.Disabled = len(manifest.Packages) - len(status.Missing) - len(status.ReEnabled)
	sort.Strings(status.Missing)
	sort.Strings(status.ReEnabled)

	for _, setting := range manifest.Settings {
		current, err := dev.GetSetting(setting.Namespace, setting.Key)
		if err != nil || current == nil || *current != setting.Tuned {
			status.SettingsDrift = append(status.SettingsDrift, setting.Namespace+"/"+setting.Key)
			continue
		}
		status.SettingsOK++
	}
	sort.Strings(status.SettingsDrift)

	return status, nil
}
