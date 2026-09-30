package logtui

import (
	"strings"
)

// LogEntry represents a single parsed logcat line
type LogEntry struct {
	Raw              string
	Timestamp        string // e.g. "17:26:34.145"
	Date             string // e.g. "09-30"
	PID              string
	TID              string
	Level            string // "V", "D", "I", "W", "E", "F"
	Tag              string
	Message          string
	IsCrashStart     bool
	IsCrashContinue  bool
	IsStackTrace     bool
	CachedSearchText string // lowercased tag + level + message + raw for fast live filtering
}

// CrashEvent represents a captured crash or ANR event
type CrashEvent struct {
	Timestamp string
	Summary   string
	Lines     []LogEntry
}

// LevelFilterOption represents available log level filters
type LevelFilterOption struct {
	Code  string
	Title string
}

var LevelFilterOptions = []LevelFilterOption{
	{Code: "ALL", Title: "All (Verbose+)"},
	{Code: "D", Title: "Debug+"},
	{Code: "I", Title: "Info+"},
	{Code: "W", Title: "Warn+"},
	{Code: "E", Title: "Error+"},
}

// FullStackTrace returns the combined raw text of the crash event
func (c *CrashEvent) FullStackTrace() string {
	var b strings.Builder
	for _, l := range c.Lines {
		b.WriteString(l.Raw)
		b.WriteByte('\n')
	}
	return b.String()
}
