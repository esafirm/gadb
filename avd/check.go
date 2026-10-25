package avd

import (
	"fmt"
	"os"
	"strconv"
)

// CheckReport is a read-only audit of one AVD.
type CheckReport struct {
	AVD         AVD
	Config      *Config
	Image       *Image
	ImageErr    error
	Runtime     *Runtime
	Manifest    *Manifest
	ManifestErr error
	Snapshot    *SnapshotStatus
	GPU         string
	RAM         string
	RAMFloor    int
	Audio       string
	Camera      string
	Verdict     string
	Notes       []string
	Next        []string
}

// Check gathers host config and, when the AVD is running, the real guest
// values. It never changes anything.
func Check(a AVD) (*CheckReport, error) {
	config, err := LoadConfig(a.ConfigPath())
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", a.ConfigPath(), err)
	}

	report := &CheckReport{AVD: a, Config: config}
	report.GPU = configValue(config, "hw.gpu.mode", "unknown")
	report.RAM = configValue(config, "hw.ramSize", "unknown")
	report.Audio = "on"
	if value, ok := config.Get("hw.audioOutput"); ok && value == "no" {
		report.Audio = "off"
	}
	report.Camera = "on"
	if back, _ := config.Get("hw.camera.back"); back == "none" {
		if front, _ := config.Get("hw.camera.front"); front == "none" {
			report.Camera = "off"
		}
	}

	report.Image, report.ImageErr = ImageForAVD(config)
	report.RAMFloor = report.Image.EffectiveRAMFloor()

	if serial, running := runningEmulator(a.Name); running {
		report.Runtime = probeRuntime(serial)
	}

	report.Manifest, report.ManifestErr = LoadManifest(a)
	report.Snapshot = CheckSnapshot(a, report.Manifest)
	report.evaluate()
	return report, nil
}

func (r *CheckReport) evaluate() {
	high, medium := false, false
	var notes, next []string

	if r.ManifestErr != nil {
		notes = append(notes, "cannot read gadb-slim.json: "+r.ManifestErr.Error())
		medium = true
	}

	if r.ImageErr != nil {
		notes = append(notes, "cannot inspect the system image: "+r.ImageErr.Error())
		medium = true
	}

	unsuitableImage := false
	if r.ImageErr != nil {
		unsuitableImage = true
	}

	if r.Image != nil {
		if !r.Image.Complete {
			notes = append(notes, "the system image is incomplete (system.img or ramdisk.img is missing)")
			high = true
			unsuitableImage = true
		}
		if r.Image.PageSizeKB == 16 {
			notes = append(notes,
				"16 KB page-size image: hw.ramSize is ignored, the guest reserves 4096 MB")
			high = true
			unsuitableImage = true
		}
		if hasExactTag(r.Image.TagID, "google_apis_playstore") {
			notes = append(notes,
				"Play Store image: adb root is unavailable, so cache compaction is skipped")
			high = true
			unsuitableImage = true
		}
		if unsuitableImage {
			next = append(next,
				fmt.Sprintf("install a 4 KB image: sdkmanager \"system-images;android-35;google_apis;%s\"", HostABI()),
				fmt.Sprintf("create a slim AVD: gadb slim create %s_slim --api=35", r.AVD.Name),
			)
		}
	}
	if r.ImageErr != nil {
		next = append(next, fmt.Sprintf("create a slim AVD: gadb slim create %s_slim --api=35", r.AVD.Name))
	}

	if r.Runtime != nil {
		if r.Runtime.Renderer == "software" {
			notes = append(notes, "the running emulator uses software rendering")
			high = true
			next = append(next, "launch with -gpu host instead of -gpu swiftshader_indirect")
		}
		if r.RAMFloor > 0 && r.Runtime.MemTotalMB > 0 {
			if configured, err := strconv.Atoi(r.RAM); err == nil && configured < r.RAMFloor {
				notes = append(notes, fmt.Sprintf(
					"the guest sees %d MB because the image floor overrides hw.ramSize=%d",
					r.Runtime.MemTotalMB, configured,
				))
			}
		}
		if r.Runtime.LowRAM {
			notes = append(notes, "the guest is in low-RAM mode (isLowRamDevice() is true)")
		}
	}

	if r.GPU == "swiftshader" || r.GPU == "swiftshader_indirect" {
		notes = append(notes, "config.ini selects software rendering")
		high = true
	}

	switch {
	case r.ManifestErr != nil:
		// already reported above
	case r.Manifest == nil && !unsuitableImage:
		medium = true
		next = append(next, "run gadb slim tune "+r.AVD.Name)
	default:
		notes = append(notes, "config.ini is tuned by gadb")
	}

	if r.Snapshot != nil && r.Snapshot.Exists {
		if r.Snapshot.Valid {
			notes = append(notes, "gadb_clean snapshot is valid")
		} else {
			notes = append(notes, "gadb_clean snapshot is stale: "+r.Snapshot.Reason)
		}
	}

	switch {
	case high:
		r.Verdict = "high"
	case medium:
		r.Verdict = "medium"
	default:
		r.Verdict = "low"
	}

	r.Notes = dedupe(notes)
	r.Next = dedupe(next)
}

func configValue(config *Config, key, fallback string) string {
	if value, ok := config.Get(key); ok && value != "" {
		return value
	}
	return fallback
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func dedupe(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := values[:0]
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}
