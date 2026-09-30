package logtui

import (
	"testing"
)

func TestFilterInclude(t *testing.T) {
	filter := &LogFilter{
		IncludeQuery: "Network",
		AllPackages:  true,
	}

	entry1 := &LogEntry{
		Tag:              "OkHttp",
		Message:          "Network call succeeded",
		CachedSearchText: "okhttp i network call succeeded",
	}
	if !filter.Allows(entry1) {
		t.Errorf("expected entry1 to pass include filter")
	}

	entry2 := &LogEntry{
		Tag:              "UI",
		Message:          "View clicked",
		CachedSearchText: "ui i view clicked",
	}
	if filter.Allows(entry2) {
		t.Errorf("expected entry2 to be filtered out")
	}
}

func TestFilterExclude(t *testing.T) {
	filter := &LogFilter{
		ExcludeQuery: "chatty",
		AllPackages:  true,
	}

	entry1 := &LogEntry{
		Tag:              "chatty",
		Message:          "uid=1000 expire 4 lines",
		CachedSearchText: "chatty i uid=1000 expire 4 lines",
	}
	if filter.Allows(entry1) {
		t.Errorf("expected chatty line to be excluded")
	}

	entry2 := &LogEntry{
		Tag:              "MyTag",
		Message:          "normal log",
		CachedSearchText: "mytag i normal log",
	}
	if !filter.Allows(entry2) {
		t.Errorf("expected normal log to pass")
	}
}

func TestFilterLevel(t *testing.T) {
	filter := &LogFilter{
		Level:       "W",
		AllPackages: true,
	}

	entryDebug := &LogEntry{Level: "D"}
	entryInfo := &LogEntry{Level: "I"}
	entryWarn := &LogEntry{Level: "W"}
	entryError := &LogEntry{Level: "E"}

	if filter.Allows(entryDebug) {
		t.Errorf("Debug should be filtered out by Warn+ filter")
	}
	if filter.Allows(entryInfo) {
		t.Errorf("Info should be filtered out by Warn+ filter")
	}
	if !filter.Allows(entryWarn) {
		t.Errorf("Warn should pass Warn+ filter")
	}
	if !filter.Allows(entryError) {
		t.Errorf("Error should pass Warn+ filter")
	}
}

func TestFilterPackageAndPID(t *testing.T) {
	filter := &LogFilter{
		PackageName: "com.example.app",
		AllPackages: false,
		PIDs: map[string]bool{
			"1234": true,
		},
	}

	entryFromPid := &LogEntry{
		PID:              "1234",
		CachedSearchText: "sometag i message",
	}
	if !filter.Allows(entryFromPid) {
		t.Errorf("expected entry matching PID to pass")
	}

	entryDiffPidWithPkgMention := &LogEntry{
		PID:              "5678",
		CachedSearchText: "activitymanager i start proc for com.example.app",
	}
	if !filter.Allows(entryDiffPidWithPkgMention) {
		t.Errorf("expected entry mentioning package name to pass")
	}

	entryUnrelated := &LogEntry{
		PID:              "9999",
		CachedSearchText: "otherapp i unrelated",
	}
	if filter.Allows(entryUnrelated) {
		t.Errorf("expected unrelated entry to be filtered out")
	}
}
