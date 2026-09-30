package logtui

import (
	"strings"
)

// LogFilter holds all active filtering criteria
type LogFilter struct {
	IncludeQuery string
	ExcludeQuery string
	Level        string // "ALL", "D", "I", "W", "E"
	PackageName  string
	AllPackages  bool
	PIDs         map[string]bool
}

// Allows checks whether entry passes the filter criteria
func (f *LogFilter) Allows(entry *LogEntry) bool {
	if entry == nil {
		return false
	}

	// 1. Package / PID filter
	if !f.AllPackages && f.PackageName != "" {
		pidMatch := false
		if len(f.PIDs) > 0 && entry.PID != "" {
			if f.PIDs[entry.PID] {
				pidMatch = true
			}
		}
		if !pidMatch {
			// Check if package name is mentioned in message, tag, or raw
			if !strings.Contains(entry.CachedSearchText, strings.ToLower(f.PackageName)) {
				return false
			}
		}
	}

	// 2. Level filter
	if !MatchesLevel(f.Level, entry.Level) {
		return false
	}

	// 3. Exclude query
	if f.ExcludeQuery != "" {
		if strings.Contains(entry.CachedSearchText, strings.ToLower(f.ExcludeQuery)) {
			return false
		}
	}

	// 4. Include query
	if f.IncludeQuery != "" {
		if !strings.Contains(entry.CachedSearchText, strings.ToLower(f.IncludeQuery)) {
			return false
		}
	}

	return true
}

// MatchesLevel checks if entryLevel meets targetLevel threshold
func MatchesLevel(targetLevel string, entryLevel string) bool {
	if targetLevel == "" || targetLevel == "ALL" {
		return true
	}

	order := map[string]int{
		"V": 1,
		"D": 2,
		"I": 3,
		"W": 4,
		"E": 5,
		"F": 6,
		"A": 6, // Assert
	}

	targetVal, ok1 := order[targetLevel]
	if !ok1 {
		return true
	}
	entryVal, ok2 := order[entryLevel]
	if !ok2 {
		return true
	}

	return entryVal >= targetVal
}
