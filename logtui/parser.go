package logtui

import (
	"hash/fnv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var tagColors = []lipgloss.Color{
	lipgloss.Color("#FF6B6B"), // Red
	lipgloss.Color("#51CF66"), // Green
	lipgloss.Color("#FCC419"), // Yellow
	lipgloss.Color("#339AF0"), // Blue
	lipgloss.Color("#CC5DE8"), // Magenta
	lipgloss.Color("#20C997"), // Cyan
	lipgloss.Color("#FF922B"), // Orange
	lipgloss.Color("#94D82D"), // Lime
	lipgloss.Color("#FFD43B"), // Light Yellow
	lipgloss.Color("#74C0FC"), // Light Blue
	lipgloss.Color("#F06595"), // Pink
	lipgloss.Color("#63E6BE"), // Light Teal
}

// TagColor returns a deterministic color from the palette for a tag
func TagColor(tag string) lipgloss.Color {
	if tag == "" {
		return lipgloss.Color("#888888")
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(tag))
	idx := int(h.Sum32()) % len(tagColors)
	if idx < 0 {
		idx = -idx
	}
	return tagColors[idx]
}

// ParseLogLine parses a raw line from adb logcat
func ParseLogLine(line string) *LogEntry {
	trimmed := strings.TrimRight(line, "\r\n")
	if trimmed == "" {
		return nil
	}

	// Handle logcat system marker lines, e.g. "--------- beginning of main"
	if strings.HasPrefix(trimmed, "---------") {
		return &LogEntry{
			Raw:              trimmed,
			Timestamp:        "",
			Date:             "",
			PID:              "",
			TID:              "",
			Level:            "I",
			Tag:              "system",
			Message:          trimmed,
			IsCrashStart:     false,
			IsCrashContinue:  false,
			IsStackTrace:     false,
			CachedSearchText: strings.ToLower(trimmed),
		}
	}

	fields := strings.Fields(trimmed)
	// Expect threadtime: Date Time PID TID Level Tag: Message
	// e.g. "09-30 17:26:34.145   649   806 D WifiScoreCard: txRate: 1 txSpeed: 1"
	if len(fields) >= 5 {
		date := fields[0]
		ts := fields[1]

		// Format timestamp to 12 chars if possible (e.g., "17:26:34.145")
		if len(ts) > 12 {
			ts = ts[:12]
		}

		pid := fields[2]
		tid := fields[3]
		level := fields[4]

		// Check if field 4 looks like a valid logcat level
		if isLogLevel(level) && len(fields) >= 6 {
			// Find tag and message in remainder
			tagAndMsg := strings.Join(fields[5:], " ")
			var tag, msg string
			if idx := strings.Index(tagAndMsg, ": "); idx != -1 {
				tag = strings.TrimSpace(tagAndMsg[:idx])
				msg = tagAndMsg[idx+2:]
			} else if idx := strings.Index(tagAndMsg, ":"); idx != -1 {
				tag = strings.TrimSpace(tagAndMsg[:idx])
				msg = strings.TrimSpace(tagAndMsg[idx+1:])
			} else {
				tag = fields[5]
				if len(fields) > 6 {
					msg = strings.Join(fields[6:], " ")
				}
			}

			entry := &LogEntry{
				Raw:              trimmed,
				Timestamp:        ts,
				Date:             date,
				PID:              pid,
				TID:              tid,
				Level:            level,
				Tag:              tag,
				Message:          msg,
				CachedSearchText: strings.ToLower(tag + " " + level + " " + msg + " " + trimmed),
			}
			entry.IsStackTrace = IsStackTrace(msg) || IsStackTrace(trimmed)
			entry.IsCrashStart = IsCrashStart(entry)
			entry.IsCrashContinue = IsCrashContinuation(entry)
			return entry
		}

		// Check if it's standard "time" format (Date Time Level/Tag(PID): Message)
		// e.g. "09-30 17:26:34.145 I/Tag(  649): message"
		if len(fields) >= 3 && strings.Contains(fields[2], "/") {
			parts := strings.SplitN(fields[2], "/", 2)
			lvl := parts[0]
			tagWithPID := parts[1]
			tag := tagWithPID
			pid := ""
			if pStart := strings.Index(tagWithPID, "("); pStart != -1 {
				tag = tagWithPID[:pStart]
				if pEnd := strings.Index(tagWithPID, ")"); pEnd > pStart {
					pid = strings.TrimSpace(tagWithPID[pStart+1 : pEnd])
				}
			}
			tag = strings.TrimSuffix(tag, ":")
			msg := strings.Join(fields[3:], " ")
			entry := &LogEntry{
				Raw:              trimmed,
				Timestamp:        ts,
				Date:             date,
				PID:              pid,
				TID:              "",
				Level:            lvl,
				Tag:              tag,
				Message:          msg,
				CachedSearchText: strings.ToLower(tag + " " + lvl + " " + msg + " " + trimmed),
			}
			entry.IsStackTrace = IsStackTrace(msg) || IsStackTrace(trimmed)
			entry.IsCrashStart = IsCrashStart(entry)
			entry.IsCrashContinue = IsCrashContinuation(entry)
			return entry
		}
	}

	// Fallback generic line
	entry := &LogEntry{
		Raw:              trimmed,
		Timestamp:        "",
		Date:             "",
		PID:              "",
		TID:              "",
		Level:            "I",
		Tag:              "",
		Message:          trimmed,
		CachedSearchText: strings.ToLower(trimmed),
	}
	entry.IsStackTrace = IsStackTrace(trimmed)
	entry.IsCrashStart = IsCrashStart(entry)
	entry.IsCrashContinue = IsCrashContinuation(entry)
	return entry
}

func isLogLevel(s string) bool {
	switch s {
	case "V", "D", "I", "W", "E", "F", "A":
		return true
	default:
		return false
	}
}

// IsCrashStart detects the initial line of a crash, ANR, or fatal signal
func IsCrashStart(entry *LogEntry) bool {
	m := strings.ToLower(entry.Message)
	r := strings.ToLower(entry.Raw)

	return strings.Contains(m, "fatal exception") ||
		strings.Contains(r, "fatal exception") ||
		strings.Contains(m, "anr in ") ||
		strings.Contains(r, "anr in ") ||
		strings.Contains(m, "crash:") ||
		strings.Contains(r, "crash:") ||
		strings.Contains(r, "sigsegv") ||
		strings.Contains(r, "sigabrt") ||
		strings.Contains(r, "sigbus") ||
		(strings.Contains(r, "debug:") && strings.Contains(r, "*** ***"))
}

// IsCrashContinuation detects subsequent lines belonging to an active crash stack trace
func IsCrashContinuation(entry *LogEntry) bool {
	if IsCrashStart(entry) {
		return false
	}

	m := entry.Message
	r := entry.Raw
	trimM := strings.TrimSpace(m)

	if IsStackTrace(m) || IsStackTrace(r) {
		return true
	}
	if strings.HasPrefix(trimM, "at ") {
		return true
	}
	if strings.Contains(m, "Caused by:") || strings.Contains(r, "Caused by:") {
		return true
	}
	if strings.Contains(m, "Suppressed:") || strings.Contains(r, "Suppressed:") {
		return true
	}
	if entry.Tag == "AndroidRuntime" || entry.Tag == "System.err" || entry.Tag == "DEBUG" {
		return true
	}

	return false
}

// IsStackTrace tests if a string looks like a Java/Kotlin or native stack trace line
func IsStackTrace(line string) bool {
	trim := strings.TrimSpace(line)
	if strings.HasPrefix(trim, "at ") {
		return true
	}
	if strings.Contains(line, "Exception") || strings.Contains(line, "Error:") {
		return true
	}
	if strings.Contains(line, "Caused by:") {
		return true
	}
	if strings.Contains(line, "backtrace:") || strings.Contains(line, "Abort message:") {
		return true
	}
	if strings.Contains(line, " pc ") && strings.Contains(line, ".so") {
		return true
	}
	return false
}
