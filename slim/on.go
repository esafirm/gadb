package slim

import (
	"fmt"
	"time"

	"github.com/esafirm/gadb/avd"
)

// On prunes packages, applies the settings plan and trims memory on a running
// emulator. The restore record is saved before anything is changed, so a crash
// halfway through still leaves Off able to undo the whole plan.
func On(a avd.AVD, dev Device, opts Options) (*OnResult, error) {
	if dev == nil {
		return nil, fmt.Errorf("a running emulator is required")
	}

	installed, err := dev.InstalledPackages()
	if err != nil {
		return nil, fmt.Errorf("cannot list installed packages: %w", err)
	}
	disabled, err := dev.DisabledPackages()
	if err != nil {
		return nil, fmt.Errorf("cannot list disabled packages: %w", err)
	}

	selection := Select(installed, disabled, opts)

	manifest, err := avd.LoadManifest(a)
	if err != nil {
		return nil, err
	}
	if manifest == nil {
		manifest = &avd.Manifest{Version: avd.ManifestVersion, AVD: a.Name}
	}

	recorded := make(map[string]bool, len(manifest.Packages))
	for _, change := range manifest.Packages {
		recorded[change.Name] = true
	}
	for _, pkg := range selection.Disable {
		if recorded[pkg] {
			continue
		}
		manifest.Packages = append(manifest.Packages, avd.PackageChange{Name: pkg, WasEnabled: true})
		recorded[pkg] = true
	}

	plan, warnings := planSettings(dev, manifest, opts)
	manifest.PackageListVersion = PackageListVersion
	manifest.AppliedAt = time.Now().UTC().Format(time.RFC3339)

	if err := avd.SaveManifest(a, manifest); err != nil {
		return nil, fmt.Errorf("cannot save the restore record: %w", err)
	}

	result := &OnResult{
		AVD:      a,
		Serial:   dev.Serial(),
		Kept:     selection.Kept,
		Warnings: warnings,
	}

	for _, pkg := range selection.Disable {
		if err := dev.DisablePackage(pkg); err != nil {
			result.FailedPackages = append(result.FailedPackages, err.Error())
			continue
		}
		result.Disabled = append(result.Disabled, pkg)
	}

	for _, item := range plan {
		if err := dev.PutSetting(item.Namespace, item.Key, item.Value); err != nil {
			result.FailedSettings = append(result.FailedSettings,
				fmt.Sprintf("%s/%s: %v", item.Namespace, item.Key, err))
			continue
		}
		result.Settings = append(result.Settings, avd.SettingChange{
			Namespace: item.Namespace,
			Key:       item.Key,
			Existed:   item.Existed,
			Original:  item.Original,
			Tuned:     item.Value,
		})
	}

	dev.TrimMemory()
	result.Trimmed = true
	return result, nil
}
