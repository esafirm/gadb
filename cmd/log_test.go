package cmd

import (
	"testing"
)

func TestLogCmdRegistration(t *testing.T) {
	if logCmd == nil {
		t.Fatalf("logCmd should be registered")
	}

	if logCmd.Use != "log" {
		t.Errorf("expected Use to be 'log', got %s", logCmd.Use)
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
	}
}
