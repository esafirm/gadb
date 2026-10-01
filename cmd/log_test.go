package cmd

import (
	"testing"
)

func TestLogCmdRegistration(t *testing.T) {
	if logCmd == nil {
		t.Fatalf("logCmd should be registered")
	}

	if logCmd.Name() != "log" {
		t.Errorf("expected Name to be 'log', got %s", logCmd.Name())
	}

	if logsCmd == nil {
		t.Fatalf("logsCmd should be registered")
	}

	if logsCmd.Name() != "logs" {
		t.Errorf("expected Name to be 'logs', got %s", logsCmd.Name())
	}

	expectedAliases := map[string]bool{"logs": true, "logcat": true}
	for _, a := range logCmd.Aliases {
		if !expectedAliases[a] {
			t.Errorf("unexpected alias %s", a)
		}
	}

	flags := []string{"package", "serial", "level", "all", "filter", "exclude"}
	for _, f := range flags {
		if logCmd.Flags().Lookup(f) == nil {
			t.Errorf("flag --%s not registered on logCmd", f)
		}
		if logsCmd.Flags().Lookup(f) == nil {
			t.Errorf("flag --%s not registered on logsCmd", f)
		}
	}
}

func TestRootCmdContainsLogAndLogs(t *testing.T) {
	foundLog := false
	foundLogs := false

	for _, c := range rootCmd.Commands() {
		if c.Name() == "log" {
			foundLog = true
		}
		if c.Name() == "logs" {
			foundLogs = true
		}
	}

	if !foundLog {
		t.Errorf("rootCmd missing 'log' subcommand")
	}
	if !foundLogs {
		t.Errorf("rootCmd missing 'logs' subcommand")
	}
}

func TestParseLogArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		flagPkg     string
		flagFilter  string
		flagAll     bool
		expectedPkg string
		expectedFil string
		expectedAll bool
	}{
		{
			name:        "no args, no flags",
			args:        []string{},
			flagPkg:     "",
			flagFilter:  "",
			flagAll:     false,
			expectedPkg: "",
			expectedFil: "",
			expectedAll: true, // defaults to all if no config and no package
		},
		{
			name:        "positional package only",
			args:        []string{"com.example.app"},
			flagPkg:     "",
			flagFilter:  "",
			flagAll:     false,
			expectedPkg: "com.example.app",
			expectedFil: "",
			expectedAll: false,
		},
		{
			name:        "positional package and filter",
			args:        []string{"com.example.app", "Retrofit"},
			flagPkg:     "",
			flagFilter:  "",
			flagAll:     false,
			expectedPkg: "com.example.app",
			expectedFil: "Retrofit",
			expectedAll: false,
		},
		{
			name:        "positional package and multi-word filter",
			args:        []string{"com.example.app", "Http", "error", "500"},
			flagPkg:     "",
			flagFilter:  "",
			flagAll:     false,
			expectedPkg: "com.example.app",
			expectedFil: "Http error 500",
			expectedAll: false,
		},
		{
			name:        "flag package with positional filter",
			args:        []string{"NetworkError"},
			flagPkg:     "com.example.app",
			flagFilter:  "",
			flagAll:     false,
			expectedPkg: "com.example.app",
			expectedFil: "NetworkError",
			expectedAll: false,
		},
		{
			name:        "flag all with positional filter",
			args:        []string{"fatal", "exception"},
			flagPkg:     "",
			flagFilter:  "",
			flagAll:     true,
			expectedPkg: "",
			expectedFil: "fatal exception",
			expectedAll: true,
		},
		{
			name:        "flag filter takes precedence over empty args",
			args:        []string{"com.example.app"},
			flagPkg:     "",
			flagFilter:  "ExistingFilter",
			flagAll:     false,
			expectedPkg: "com.example.app",
			expectedFil: "ExistingFilter",
			expectedAll: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pkg, filter, all := parseLogArgs(tc.args, tc.flagPkg, tc.flagFilter, tc.flagAll)
			if pkg != tc.expectedPkg {
				t.Errorf("expected pkg %q, got %q", tc.expectedPkg, pkg)
			}
			if filter != tc.expectedFil {
				t.Errorf("expected filter %q, got %q", tc.expectedFil, filter)
			}
			if all != tc.expectedAll {
				t.Errorf("expected all %v, got %v", tc.expectedAll, all)
			}
		})
	}
}
