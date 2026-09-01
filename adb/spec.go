package adb

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// DeviceSpec holds hardware and OS info about a connected device
type DeviceSpec struct {
	Model        string `json:"model"`
	Manufacturer string `json:"manufacturer"`
	OSVersion    string `json:"os_version"`
	SDKVersion   string `json:"sdk_version"`
	DensityDpi   string `json:"density_dpi"`
	WidthPx      string `json:"width_px"`
	HeightPx     string `json:"height_px"`
	WidthDp      string `json:"width_dp"`
	HeightDp     string `json:"height_dp"`
}

// DeviceSpecCommandReturn have the gathered DeviceSpec and error
type DeviceSpecCommandReturn struct {
	Spec  DeviceSpec
	Error error
}

var wmSizeRegex = regexp.MustCompile(`Physical size:\s*(\d+)x(\d+)`)
var wmDensityRegex = regexp.MustCompile(`Physical density:\s*(\d+)`)

// Spec gathers device model, OS version, density and screen size
func Spec() DeviceSpecCommandReturn {
	model, err := getProp("ro.product.model")
	if err != nil {
		return DeviceSpecCommandReturn{Error: err}
	}

	manufacturer, err := getProp("ro.product.manufacturer")
	if err != nil {
		return DeviceSpecCommandReturn{Error: err}
	}

	osVersion, err := getProp("ro.build.version.release")
	if err != nil {
		return DeviceSpecCommandReturn{Error: err}
	}

	sdkVersion, err := getProp("ro.build.version.sdk")
	if err != nil {
		return DeviceSpecCommandReturn{Error: err}
	}

	sizeReturn := runOnly("adb", "shell", "wm", "size")
	if sizeReturn.Error != nil {
		return DeviceSpecCommandReturn{Error: sizeReturn.Error}
	}
	widthPx, heightPx := parseWmSize(string(sizeReturn.Output))

	densityReturn := runOnly("adb", "shell", "wm", "density")
	if densityReturn.Error != nil {
		return DeviceSpecCommandReturn{Error: densityReturn.Error}
	}
	densityDpi := parseWmDensity(string(densityReturn.Output))

	return DeviceSpecCommandReturn{
		Spec: DeviceSpec{
			Model:        model,
			Manufacturer: manufacturer,
			OSVersion:    osVersion,
			SDKVersion:   sdkVersion,
			DensityDpi:   densityDpi,
			WidthPx:      widthPx,
			HeightPx:     heightPx,
			WidthDp:      computeLogicalPixels(widthPx, densityDpi),
			HeightDp:     computeLogicalPixels(heightPx, densityDpi),
		},
	}
}

func getProp(key string) (string, error) {
	commandReturn := runOnly("adb", "shell", "getprop", key)
	if commandReturn.Error != nil {
		return "", commandReturn.Error
	}
	return strings.TrimSpace(string(commandReturn.Output)), nil
}

func parseWmSize(raw string) (widthPx string, heightPx string) {
	matches := wmSizeRegex.FindStringSubmatch(raw)
	if len(matches) < 3 {
		return "", ""
	}
	return matches[1], matches[2]
}

func parseWmDensity(raw string) string {
	matches := wmDensityRegex.FindStringSubmatch(raw)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

// computeLogicalPixels converts a physical pixel dimension to
// density-independent pixels (dp), i.e. px / (densityDpi / 160)
func computeLogicalPixels(px string, densityDpi string) string {
	pxValue, err := strconv.Atoi(px)
	if err != nil {
		return ""
	}
	densityValue, err := strconv.Atoi(densityDpi)
	if err != nil || densityValue == 0 {
		return ""
	}
	dp := float64(pxValue) / (float64(densityValue) / 160)
	return fmt.Sprintf("%.0f", dp)
}
