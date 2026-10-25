package avd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// CreateOptions controls AVD creation.
type CreateOptions struct {
	Name       string
	API        int
	Device     string
	RAM        int
	GPU        string
	KeepAudio  bool
	KeepCamera bool
}

// CreateResult reports a created and tuned AVD.
type CreateResult struct {
	AVD        AVD
	Image      Image
	Device     string
	AVDManager string
	Tune       *TuneResult
	Warnings   []string
}

// ValidAVDName mirrors avdmanager's accepted characters.
func ValidAVDName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' ||
			r == '.' || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

// FindAVDManager locates avdmanager on PATH, then in the SDK's cmdline-tools.
func FindAVDManager() string {
	names := []string{"avdmanager"}
	if runtime.GOOS == "windows" {
		names = []string{"avdmanager.bat", "avdmanager"}
	}
	for _, name := range names {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}

	sdk := SDKDir()
	if sdk == "" {
		return ""
	}

	candidates := make([]string, 0, 4)
	for _, name := range names {
		candidates = append(candidates, filepath.Join(sdk, "cmdline-tools", "latest", "bin", name))
	}
	for _, name := range names {
		versioned, _ := filepath.Glob(filepath.Join(sdk, "cmdline-tools", "*", "bin", name))
		sort.Sort(sort.Reverse(sort.StringSlice(versioned)))
		candidates = append(candidates, versioned...)
	}
	for _, name := range names {
		candidates = append(candidates, filepath.Join(sdk, "tools", "bin", name))
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

// Create builds an AVD from the newest qualifying image and tunes it.
func Create(opts CreateOptions) (*CreateResult, error) {
	if !ValidAVDName(opts.Name) {
		return nil, fmt.Errorf("invalid AVD name %q (letters, digits, '.', '_' or '-')", opts.Name)
	}

	installed, err := List()
	if err != nil {
		return nil, err
	}
	for _, existing := range installed {
		if strings.EqualFold(existing.Name, opts.Name) {
			return nil, fmt.Errorf("AVD %q already exists; tune it instead: gadb slim tune %s", existing.Name, existing.Name)
		}
	}

	if opts.Device == "" {
		opts.Device = "pixel"
	}

	images, err := FindImages(opts.API)
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, noImageError(opts.API)
	}
	image := images[0]

	avdmanager := FindAVDManager()
	if avdmanager == "" {
		return nil, errors.New(
			"avdmanager not found. Install the SDK Command-line Tools " +
				"(Android Studio > SDK Manager > SDK Tools) or put avdmanager on PATH",
		)
	}

	args := []string{"create", "avd", "-n", opts.Name, "-k", image.PkgID, "-d", opts.Device}
	command := exec.Command(avdmanager, args...)
	if runtime.GOOS == "windows" {
		switch strings.ToLower(filepath.Ext(avdmanager)) {
		case ".bat", ".cmd":
			command = exec.Command("cmd", append([]string{"/c", avdmanager}, args...)...)
		}
	}
	command.Stdin = strings.NewReader("no\n") // decline the custom hardware profile prompt
	output, err := command.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message != "" {
			return nil, fmt.Errorf("avdmanager failed: %w\n%s\navdmanager needs Java 17+", err, message)
		}
		return nil, fmt.Errorf("avdmanager failed: %w (it needs Java 17+)", err)
	}

	created, err := Find(opts.Name)
	if err != nil {
		return nil, fmt.Errorf("avdmanager reported success but %s was not created: %w", opts.Name, err)
	}

	result := &CreateResult{
		AVD:        created,
		Image:      image,
		Device:     opts.Device,
		AVDManager: avdmanager,
	}
	if image.APIMajor > 0 && image.APIMajor < 30 {
		result.Warnings = append(result.Warnings, fmt.Sprintf(
			"API %d is old; install a newer google_apis image for modern app testing", image.APIMajor,
		))
	}

	tune, err := Tune(created, TuneOptions{
		RAM:        opts.RAM,
		GPU:        opts.GPU,
		KeepAudio:  opts.KeepAudio,
		KeepCamera: opts.KeepCamera,
	})
	if err != nil {
		return nil, fmt.Errorf("created %q but tuning failed: %w", created.Name, err)
	}
	result.Tune = tune
	return result, nil
}

// noImageError explains what is installed and prints the exact install command.
func noImageError(api int) error {
	abi := HostABI()
	level := 35
	if api > 0 {
		level = api
	}

	var message strings.Builder
	fmt.Fprintf(&message, "no complete 4 KB google_apis image for %s", abi)
	if api > 0 {
		fmt.Fprintf(&message, " at API %d", api)
	}
	message.WriteString("\n")

	all, _ := InstalledImages()
	var rows []string
	for _, image := range all {
		if image.ABI != abi {
			continue
		}
		if !hasExactTag(image.TagID, "google_apis") && !hasExactTag(image.TagID, "google_apis_playstore") {
			continue
		}
		status := "complete"
		if !image.Complete {
			status = "incomplete (system.img missing)"
		}
		if image.PageSizeKB == 16 {
			status += ", 16 KB pages"
		}
		if hasExactTag(image.TagID, "google_apis_playstore") {
			status += ", Play Store"
		}
		rows = append(rows, fmt.Sprintf("  API %d %s: %s", image.APIMajor, image.TagID, status))
	}
	if len(rows) > 0 {
		message.WriteString("installed Google images:\n")
		message.WriteString(strings.Join(rows, "\n"))
		message.WriteString("\n")
	}
	fmt.Fprintf(&message, "install one:\n  sdkmanager \"system-images;android-%d;google_apis;%s\"", level, abi)
	return errors.New(message.String())
}
