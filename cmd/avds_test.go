package cmd

import (
	"reflect"
	"testing"
)

func TestAvdsEmulatorOptions(t *testing.T) {
	t.Cleanup(func() {
		isNoWindow = false
		avdEmulatorArgs = nil
	})

	err := avdsCmd.Flags().Parse([]string{"--no-window", "-e", "-no-audio", "-e", "-gpu"})
	if err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	want := []string{"-no-window", "-no-audio", "-gpu"}
	if got := emulatorOptions(); !reflect.DeepEqual(got, want) {
		t.Fatalf("emulatorOptions() = %#v, want %#v", got, want)
	}
}
