package slim

import (
	"sort"
	"strings"
)

// The package groups below are derived from avdslim
// (https://github.com/kdbhalala/avdslim), Copyright (c) 2026 Krunal Bhalala,
// MIT License. See NOTICE for the full license text.

// Group is a named set of packages that --skip can leave alone.
type Group struct {
	Name     string
	Packages []string
}

// Groups are the standard, non-aggressive package groups.
var Groups = []Group{
	{
		Name: "assistant",
		Packages: []string{
			"com.google.android.googlequicksearchbox",
			"com.google.android.as",
			"com.google.android.as.oss",
		},
	},
	{
		Name: "peripherals",
		Packages: []string{
			"com.android.bluetoothmidiservice",
		},
	},
	{
		Name: "privacy",
		Packages: []string{
			"com.google.android.adservices.api",
			"com.google.mainline.adservices",
		},
	},
	{
		Name: "wellbeing",
		Packages: []string{
			"com.google.android.apps.wellbeing",
			"com.google.android.settings.intelligence",
			"com.android.emulator.multidisplay",
			"com.google.android.apps.bard",
			"com.google.android.apps.restore",
			"com.google.android.apps.pulse",
			"com.google.android.configupdater",
			"com.google.android.projection.gearhead",
			"com.google.android.healthconnect.controller",
			"com.android.imsserviceentitlement",
			"com.android.carrierdefaultapp",
			"com.google.android.ondevicepersonalization.services",
			"com.google.android.federatedcompute",
			"com.google.android.glasses.companion",
			"com.google.android.glasses.core",
			"com.google.android.apps.safetyhub",
			"com.google.android.markup",
		},
	},
	{
		Name: "media",
		Packages: []string{
			"com.google.android.apps.photos",
			"com.google.android.youtube",
			"com.google.android.videos",
			"com.google.android.music",
			"com.google.android.apps.youtube.music",
			"com.google.android.apps.maps",
			"com.google.android.gm",
			"com.google.android.apps.docs",
			"com.android.gallery3d",
			"com.android.music",
			"com.android.camera2",
			"com.android.cameraextensions",
			"com.android.DeviceAsWebcam",
		},
	},
	{
		Name: "telephony",
		Packages: []string{
			"com.google.android.apps.messaging",
			"com.google.android.dialer",
			"com.google.android.contacts",
			"com.android.mms",
			"com.android.dialer",
			"com.android.contacts",
			"com.google.android.cellbroadcastservice",
			"com.android.cellbroadcastreceiver",
			"com.google.android.telephony.satellite",
			"com.android.stk",
		},
	},
	{
		Name: "accessibility",
		Packages: []string{
			"com.google.android.tts",
			"com.google.android.marvin.talkback",
			"com.google.android.apps.accessibility.voiceaccess",
		},
	},
	{
		Name: "printing",
		Packages: []string{
			"com.android.printspooler",
			"com.android.bips",
			"com.google.android.printservice.recommendation",
		},
	},
	{
		Name: "telemetry",
		Packages: []string{
			"com.google.android.feedback",
			"com.android.traceur",
			"com.android.dreams.basic",
			"com.android.dreams.phototable",
			"com.android.wallpaper.livepicker",
			"com.google.android.apps.wallpaper",
			"com.android.emergency",
			"com.android.calendar",
			"com.google.android.calendar",
			"com.android.deskclock",
			"com.google.android.deskclock",
			"com.android.calculator2",
			"com.google.android.calculator",
		},
	},
}

// AggressivePackages are added by --aggressive.
var AggressivePackages = []string{
	"com.android.vending",
	"com.android.chrome",
	"com.google.android.partnersetup",
	"com.google.android.setupwizard",
	"com.android.setupwizard",
}

// bootCritical must never be disabled: Android 16+ crash-loops on boot when
// the Bluetooth APK is disabled while the feature flag is set.
var bootCritical = map[string]bool{
	"com.google.android.bluetooth": true,
	"com.android.bluetooth":        true,
}

// protectedPrefixes keep core OS, Play Services, WebView, input, launcher and
// package management working. Matching is by prefix for GMS and friends,
// which have many packages that share the prefix. Bluetooth is matched
// exactly above so the MIDI service can still be pruned.
var protectedPrefixes = []string{
	"com.google.android.gms",
	"com.google.android.gsf",
	"com.google.android.webview",
	"com.android.webview",
	"com.google.android.trichromelibrary",
	"com.android.systemui",
	"com.android.settings",
	"com.android.phone",
	"com.android.shell",
	"com.android.inputmethod",
	"com.google.android.inputmethod",
	"com.android.launcher",
	"com.google.android.apps.nexuslauncher",
	"com.android.permissioncontroller",
	"com.android.packageinstaller",
	"com.google.android.packageinstaller",
	"com.android.providers",
}

// Selection is the package plan for one On run.
type Selection struct {
	Disable []string
	Kept    []string
}

// IsProtected reports whether a package must never be disabled.
func IsProtected(pkg string) bool {
	if bootCritical[pkg] {
		return true
	}
	for _, prefix := range protectedPrefixes {
		if strings.HasPrefix(pkg, prefix) {
			return true
		}
	}
	return false
}

// SkipGroups lists every value --skip accepts.
func SkipGroups() []string {
	groups := make([]string, 0, len(Groups)+3)
	for _, group := range Groups {
		groups = append(groups, group.Name)
	}
	groups = append(groups, "aggressive", "animations", "bglimit")
	sort.Strings(groups)
	return groups
}

// ValidSkip reports whether group is a known --skip value.
func ValidSkip(group string) bool {
	for _, candidate := range SkipGroups() {
		if candidate == group {
			return true
		}
	}
	return false
}

// Select decides which installed, currently enabled packages to disable.
func Select(installed, disabled map[string]bool, opts Options) Selection {
	skip := make(map[string]bool, len(opts.Skip))
	for _, group := range opts.Skip {
		skip[group] = true
	}
	keep := make(map[string]bool, len(opts.Keep))
	for _, pkg := range opts.Keep {
		keep[pkg] = true
	}

	seen := make(map[string]bool)
	var candidates []string
	for _, group := range Groups {
		if skip[group.Name] {
			continue
		}
		for _, pkg := range group.Packages {
			if seen[pkg] {
				continue
			}
			seen[pkg] = true
			candidates = append(candidates, pkg)
		}
	}
	if opts.Aggressive && !skip["aggressive"] {
		for _, pkg := range AggressivePackages {
			if seen[pkg] {
				continue
			}
			seen[pkg] = true
			candidates = append(candidates, pkg)
		}
	}

	selection := Selection{}
	for _, pkg := range candidates {
		if !installed[pkg] || disabled[pkg] {
			continue
		}
		if keep[pkg] || IsProtected(pkg) {
			selection.Kept = append(selection.Kept, pkg)
			continue
		}
		selection.Disable = append(selection.Disable, pkg)
	}

	sort.Strings(selection.Disable)
	sort.Strings(selection.Kept)
	return selection
}
