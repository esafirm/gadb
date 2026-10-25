package avd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Image is one installed system image, read from its source.properties.
type Image struct {
	Dir        string
	PkgID      string
	API        string
	APIMajor   int
	ABI        string
	TagID      string
	TagDisplay string
	Revision   string
	PageSizeKB int
	Complete   bool
	Desc       string
}

// InspectImage reads a system-image directory. source.properties is the
// source of truth: directory names lie, for example android-37.2-beta3
// reports AndroidVersion.ApiLevel=37.1.
func InspectImage(dir string) (*Image, error) {
	data, err := os.ReadFile(filepath.Join(dir, "source.properties"))
	if err != nil {
		return nil, err
	}
	props := parseProperties(data)

	image := &Image{
		Dir:        dir,
		API:        props["AndroidVersion.ApiLevel"],
		ABI:        props["SystemImage.Abi"],
		TagID:      props["SystemImage.TagId"],
		TagDisplay: props["SystemImage.TagDisplay"],
		Revision:   props["Pkg.Revision"],
		Desc:       props["Pkg.Desc"],
		PageSizeKB: 4,
	}

	image.APIMajor = apiMajor(image.API)
	if image.APIMajor == 0 {
		// Fall back to the android-<level> directory for images that omit it.
		platform := filepath.Base(filepath.Dir(filepath.Dir(dir)))
		image.APIMajor = apiMajor(strings.TrimPrefix(platform, "android-"))
	}
	if hasExactTag(image.TagID, "page_size_16kb") ||
		strings.Contains(strings.ToLower(image.Desc), "page size 16k") {
		image.PageSizeKB = 16
	}
	image.Complete = imageFileExists(filepath.Join(dir, "system.img")) &&
		imageFileExists(filepath.Join(dir, "ramdisk.img"))

	if sdk := SDKDir(); sdk != "" {
		if rel, err := filepath.Rel(sdk, dir); err == nil && !strings.HasPrefix(rel, "..") {
			image.PkgID = strings.ReplaceAll(filepath.ToSlash(rel), "/", ";")
		}
	}

	return image, nil
}

// InstalledImages lists every installed system image, unfiltered. Create uses
// it to explain why no suitable image qualified.
func InstalledImages() ([]Image, error) {
	sdk := SDKDir()
	if sdk == "" {
		return nil, fmt.Errorf("cannot determine the Android SDK directory")
	}

	matches, err := filepath.Glob(filepath.Join(sdk, "system-images", "*", "*", "*", "source.properties"))
	if err != nil {
		return nil, err
	}

	images := make([]Image, 0, len(matches))
	for _, match := range matches {
		image, err := InspectImage(filepath.Dir(match))
		if err != nil {
			continue
		}
		images = append(images, *image)
	}

	sortImages(images)
	return images, nil
}

// FindImages lists complete, 4 KB, plain Google APIs images for the host ABI,
// newest API first. api <= 0 accepts every major level.
func FindImages(api int) ([]Image, error) {
	all, err := InstalledImages()
	if err != nil {
		return nil, err
	}

	images := make([]Image, 0, len(all))
	for _, image := range all {
		if !hasExactTag(image.TagID, "google_apis") {
			continue // excludes google_apis_playstore and page_size_16kb
		}
		if image.ABI != HostABI() || !image.Complete {
			continue
		}
		if api > 0 && image.APIMajor != api {
			continue
		}
		images = append(images, image)
	}
	return images, nil
}

func sortImages(images []Image) {
	sort.Slice(images, func(i, j int) bool {
		if images[i].APIMajor != images[j].APIMajor {
			return images[i].APIMajor > images[j].APIMajor
		}
		return images[i].Revision > images[j].Revision
	})
}

// ImageForAVD resolves the image referenced by config.ini.
func ImageForAVD(config *Config) (*Image, error) {
	sysdir, ok := config.Get("image.sysdir.1")
	if !ok || strings.TrimSpace(sysdir) == "" {
		return nil, fmt.Errorf("image.sysdir.1 is missing from config.ini")
	}

	sysdir = filepath.FromSlash(strings.TrimSpace(sysdir))
	if !filepath.IsAbs(sysdir) {
		sdk := SDKDir()
		if sdk == "" {
			return nil, fmt.Errorf("cannot determine the Android SDK directory")
		}
		sysdir = filepath.Join(sdk, sysdir)
	}
	return InspectImage(filepath.Clean(sysdir))
}

// IsPlayStore reports whether this is a Play Store image, which cannot use
// adb root.
func (i *Image) IsPlayStore() bool {
	return i != nil && hasExactTag(i.TagID, "google_apis_playstore")
}

// EffectiveRAMFloor is the guest RAM the image enforces regardless of
// hw.ramSize. 16 KB page-size images reserve 4 GB in this emulator.
func (i *Image) EffectiveRAMFloor() int {
	if i != nil && i.PageSizeKB == 16 {
		return 4096
	}
	return 0
}

// ShortLabel is a compact image label for tables, e.g. "37/playstore+16KB".
func (i *Image) ShortLabel() string {
	if i == nil {
		return "unknown"
	}

	parts := make([]string, 0, 2)
	switch {
	case hasExactTag(i.TagID, "google_apis_playstore"):
		parts = append(parts, "playstore")
	case hasExactTag(i.TagID, "google_apis"):
		parts = append(parts, "google_apis")
	case i.TagID != "":
		parts = append(parts, i.TagID)
	default:
		parts = append(parts, "unknown")
	}
	if i.PageSizeKB == 16 {
		parts = append(parts, "16KB")
	}

	label := strings.Join(parts, "+")
	if i.APIMajor > 0 {
		label = strconv.Itoa(i.APIMajor) + "/" + label
	}
	return label
}

func hasExactTag(raw, want string) bool {
	for _, tag := range strings.Split(raw, ",") {
		if strings.TrimSpace(tag) == want {
			return true
		}
	}
	return false
}

func apiMajor(raw string) int {
	if raw == "" {
		return 0
	}
	if major, _, found := strings.Cut(raw, "."); found {
		raw = major
	}
	level, _ := strconv.Atoi(raw)
	return level
}

func parseProperties(data []byte) map[string]string {
	props := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		separator := strings.Index(line, "=")
		if separator <= 0 {
			continue
		}
		props[strings.TrimSpace(line[:separator])] = strings.TrimSpace(line[separator+1:])
	}
	return props
}

func imageFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Size() > 0
}
