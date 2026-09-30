package logtui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderHeaderResponsiveness(t *testing.T) {
	widths := []int{60, 80, 100, 120, 160, 200}

	for _, w := range widths {
		m := NewModel(Config{
			DeviceSerial: "emulator-5554",
			PackageName:  "com.example.super.long.application.identifier.for.android",
		})
		m.deviceModel = "Android SDK built for arm64 v8a"
		m.androidRelease = "16"
		m.sdkVersion = "36"

		header := RenderHeader(m, w)
		lines := strings.Split(header, "\n")

		if len(lines) != 2 {
			t.Errorf("expected header to have exactly 2 lines at width %d, got %d lines:\n%s", w, len(lines), header)
		}

		for i, line := range lines {
			lineWidth := lipgloss.Width(line)
			if lineWidth > w {
				t.Errorf("header line %d exceeds width %d: actual width %d", i, w, lineWidth)
			}
		}
	}
}

func TestRenderFooterResponsiveness(t *testing.T) {
	// Test various terminal widths from small to very wide
	widths := []int{50, 70, 85, 100, 110, 125, 150, 200}

	for _, w := range widths {
		m := NewModel(Config{
			DeviceSerial: "emulator-5554",
		})

		footer := RenderFooter(m, w)
		lines := strings.Split(footer, "\n")

		// footerBlockStyle adds 1 border line at top + 1 status line + 1 keycaps line = 3 lines max
		if len(lines) > 3 {
			t.Errorf("footer wrapped at width %d! got %d lines (expected <= 3):\n%s", w, len(lines), footer)
		}

		for i, line := range lines {
			lineWidth := lipgloss.Width(line)
			if lineWidth > w {
				t.Errorf("footer line %d exceeds width %d: actual %d", i, w, lineWidth)
			}
		}
	}
}

func TestRenderPackageModalNoCarriageReturns(t *testing.T) {
	m := NewModel(Config{
		DeviceSerial: "emulator-5554",
	})
	m.installedPackages = []string{
		"com.google.android.youtube\r",
		"com.google.android.apps.messaging\r",
		"com.google.android.apps.turbo\r",
	}
	m.filterPackageSuggestions()

	modal := RenderPackageModal(m, 120, 30)

	// Verify no \r in the rendered output
	if strings.Contains(modal, "\r") {
		t.Fatalf("RenderPackageModal contains \\r carriage return!")
	}

	// Verify package names are present without \r
	if !strings.Contains(modal, "com.google.android.youtube") {
		t.Errorf("modal missing expected package suggestion")
	}
}

func TestRenderCrashPopupAndHelp(t *testing.T) {
	m := NewModel(Config{})
	entry := &LogEntry{
		Timestamp:    "10:00:00.123",
		Level:        "E",
		Tag:          "AndroidRuntime",
		Message:      "FATAL EXCEPTION: main",
		Raw:          "10:00:00.123 1234 1234 E AndroidRuntime: FATAL EXCEPTION: main",
		IsCrashStart: true,
	}
	m.crashes = append(m.crashes, CrashEvent{
		Timestamp: entry.Timestamp,
		Summary:   entry.Message,
		Lines:     []LogEntry{*entry},
	})

	popup := RenderCrashPopup(m, 100, 30)
	if popup == "" {
		t.Fatalf("expected non-empty crash popup")
	}
	if strings.Contains(popup, "\r") {
		t.Fatalf("crash popup contains \\r")
	}

	help := RenderHelpModal(100, 30)
	if help == "" {
		t.Fatalf("expected non-empty help modal")
	}
	if strings.Contains(help, "\r") {
		t.Fatalf("help modal contains \\r")
	}
}
