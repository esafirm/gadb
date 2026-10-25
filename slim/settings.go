package slim

import (
	"strings"

	"github.com/esafirm/gadb/avd"
)

type tweak struct {
	Namespace string
	Key       string
	Value     string
	Group     string
}

var defaultTweaks = []tweak{
	{"global", "background_process_limit", "4", "bglimit"},
	{"global", "activity_manager_constants", "max_cached_processes=4", "bglimit"},
}

var animationTweaks = []tweak{
	{"global", "window_animation_scale", "0", "animations"},
	{"global", "transition_animation_scale", "0", "animations"},
	{"global", "animator_duration_scale", "0", "animations"},
}

type plannedSetting struct {
	Namespace string
	Key       string
	Value     string
	Original  *string
	Existed   bool
}

func planSettings(dev Device, manifest *avd.Manifest, opts Options) ([]plannedSetting, []string) {
	skip := make(map[string]bool, len(opts.Skip))
	for _, group := range opts.Skip {
		skip[group] = true
	}

	tweaks := append([]tweak{}, defaultTweaks...)
	if opts.NoAnim {
		tweaks = append(tweaks, animationTweaks...)
	}

	var plan []plannedSetting
	var warnings []string
	for _, item := range tweaks {
		if skip[item.Group] {
			continue
		}

		current, err := dev.GetSetting(item.Namespace, item.Key)
		if err != nil {
			warnings = append(warnings,
				"cannot read "+item.Namespace+"/"+item.Key+": "+err.Error())
			continue
		}

		value := item.Value
		if item.Key == "activity_manager_constants" {
			key, desired, _ := strings.Cut(item.Value, "=")
			var changed bool
			value, changed = mergeConstants(current, key, desired)
			if !changed {
				continue
			}
		} else if current != nil && *current == value {
			continue
		}

		recordSetting(manifest, item, current, value)
		plan = append(plan, plannedSetting{
			Namespace: item.Namespace,
			Key:       item.Key,
			Value:     value,
			Original:  current,
			Existed:   current != nil,
		})
	}
	return plan, warnings
}

func recordSetting(manifest *avd.Manifest, item tweak, original *string, value string) {
	var copy *string
	if original != nil {
		saved := *original
		copy = &saved
	}

	for i := range manifest.Settings {
		if manifest.Settings[i].Namespace == item.Namespace && manifest.Settings[i].Key == item.Key {
			manifest.Settings[i].Tuned = value
			return
		}
	}
	manifest.Settings = append(manifest.Settings, avd.SettingChange{
		Namespace: item.Namespace,
		Key:       item.Key,
		Existed:   original != nil,
		Original:  copy,
		Tuned:     value,
	})
}

// mergeConstants preserves the other activity_manager_constants pairs and only
// replaces max_cached_processes. Replacing the whole string would silently drop
// constants the image or user set.
func mergeConstants(current *string, key, value string) (string, bool) {
	if current == nil || strings.TrimSpace(*current) == "" {
		return key + "=" + value, true
	}

	parts := strings.Split(*current, ",")
	found, changed := false, false
	for i, part := range parts {
		k, v, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok || strings.TrimSpace(k) != key {
			continue
		}
		found = true
		if strings.TrimSpace(v) != value {
			parts[i] = key + "=" + value
			changed = true
		}
	}
	if !found {
		parts = append(parts, key+"="+value)
		changed = true
	}
	if !changed {
		return *current, false
	}
	return strings.Join(parts, ","), true
}

func restoreSetting(dev Device, setting avd.SettingChange) error {
	if setting.Existed {
		value := ""
		if setting.Original != nil {
			value = *setting.Original
		}
		return dev.PutSetting(setting.Namespace, setting.Key, value)
	}
	return dev.DeleteSetting(setting.Namespace, setting.Key)
}
