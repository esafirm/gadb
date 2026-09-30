package logtui

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/atotto/clipboard"
)

// CopyToClipboard copies a string to system clipboard
func CopyToClipboard(text string) error {
	err := clipboard.WriteAll(text)
	if err == nil {
		return nil
	}

	// Fallback to platform-specific CLI tools
	switch runtime.GOOS {
	case "darwin":
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	case "linux":
		cmd := exec.Command("xclip", "-selection", "clipboard")
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err == nil {
			return nil
		}
		cmdWl := exec.Command("wl-copy")
		cmdWl.Stdin = strings.NewReader(text)
		return cmdWl.Run()
	case "windows":
		cmd := exec.Command("clip")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}

	return err
}

// ExportLogsToFile writes log entries to a timestamped file
func ExportLogsToFile(entries []LogEntry, prefix string) (string, error) {
	if prefix == "" {
		prefix = "gadb-log"
	}
	ts := time.Now().Format("2006-01-02-150405")
	filename := fmt.Sprintf("%s-%s.log", prefix, ts)

	var sb strings.Builder
	for _, e := range entries {
		sb.WriteString(e.Raw)
		sb.WriteByte('\n')
	}

	err := os.WriteFile(filename, []byte(sb.String()), 0644)
	if err != nil {
		return "", err
	}
	return filename, nil
}

// ExportCrashToFile writes crash details to crash-<timestamp>.log
func ExportCrashToFile(crash CrashEvent) (string, error) {
	ts := time.Now().Format("2006-01-02-150405")
	filename := fmt.Sprintf("crash-%s.log", ts)

	var sb strings.Builder
	sb.WriteString("=== Crash Event ===\n")
	sb.WriteString(fmt.Sprintf("Timestamp: %s\n", crash.Timestamp))
	sb.WriteString(fmt.Sprintf("Summary: %s\n\n", crash.Summary))
	sb.WriteString("=== Stack Trace ===\n")
	sb.WriteString(crash.FullStackTrace())

	err := os.WriteFile(filename, []byte(sb.String()), 0644)
	if err != nil {
		return "", err
	}
	return filename, nil
}

// FormatAIAgentPrompt formats a crash for AI coding assistants
func FormatAIAgentPrompt(crash CrashEvent, pkg string) string {
	var sb strings.Builder
	sb.WriteString("Help fix this Android crash:\n\n")
	if pkg != "" {
		sb.WriteString(fmt.Sprintf("Package: %s\n", pkg))
	}
	sb.WriteString(fmt.Sprintf("Summary: %s\n", crash.Summary))
	sb.WriteString(fmt.Sprintf("Timestamp: %s\n\n", crash.Timestamp))
	sb.WriteString("Stack Trace:\n```\n")
	sb.WriteString(crash.FullStackTrace())
	sb.WriteString("```\n\nPlease analyze the cause of this crash and explain how to resolve it.")
	return sb.String()
}

// SearchCrashOnline opens the default web browser with a Google search for the exception
func SearchCrashOnline(summary string) error {
	query := summary
	// Strip common prefixes
	for _, prefix := range []string{"FATAL EXCEPTION: main", "FATAL EXCEPTION:", "AndroidRuntime: FATAL EXCEPTION:"} {
		query = strings.TrimPrefix(query, prefix)
	}
	query = strings.TrimSpace(query)
	if query == "" {
		query = summary
	}

	searchURL := "https://www.google.com/search?q=" + url.QueryEscape(query)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", searchURL)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", searchURL)
	default:
		cmd = exec.Command("xdg-open", searchURL)
	}

	return cmd.Start()
}
