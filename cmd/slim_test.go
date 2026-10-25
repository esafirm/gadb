package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/esafirm/gadb/avd"
	"github.com/spf13/cobra"
)

func slimTestCommand() *cobra.Command {
	command := &cobra.Command{}
	command.Flags().Int("ram", 2048, "")
	command.Flags().String("gpu", "host", "")
	command.Flags().Bool("keep-audio", false, "")
	command.Flags().Bool("keep-camera", false, "")
	return command
}

func slimTestAVD(t *testing.T, config string, manifest *avd.Manifest) avd.AVD {
	t.Helper()

	base := t.TempDir()
	t.Setenv("ANDROID_AVD_HOME", base)
	dir := filepath.Join(base, "Test.avd")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.ini"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	a := avd.AVD{Name: "Test", Dir: dir}
	if manifest != nil {
		if err := avd.SaveManifest(a, manifest); err != nil {
			t.Fatal(err)
		}
	}
	return a
}

func tunedManifest() *avd.Manifest {
	return &avd.Manifest{
		Version: 1,
		AVD:     "Test",
		Config: map[string]avd.ConfigChange{
			"hw.ramSize": {Original: "4096", Tuned: "1536", Existed: true},
		},
	}
}

func TestSlimTuneStateKeepsExistingTune(t *testing.T) {
	a := slimTestAVD(t, "hw.ramSize=1536\nhw.gpu.mode=host\n", tunedManifest())

	tune, ram, gpu := slimTuneState(slimTestCommand(), a, 2048, "auto")
	if tune {
		t.Fatal("tune = true, want the existing tune preserved")
	}
	if ram != 1536 || gpu != "host" {
		t.Fatalf("ram/gpu = %d/%q, want the tuned config values", ram, gpu)
	}
}

func TestSlimTuneStateExplicitRAM(t *testing.T) {
	a := slimTestAVD(t, "hw.ramSize=1536\nhw.gpu.mode=host\n", tunedManifest())

	command := slimTestCommand()
	if err := command.Flags().Set("ram", "2048"); err != nil {
		t.Fatal(err)
	}
	tune, ram, gpu := slimTuneState(command, a, 2048, "auto")
	if !tune || ram != 2048 {
		t.Fatalf("tune/ram = %v/%d, want a retune at 2048", tune, ram)
	}
	if gpu != "host" {
		t.Fatalf("gpu = %q, want the untouched config value", gpu)
	}
}

func TestSlimTuneStateKeepAudioKeepsRAM(t *testing.T) {
	a := slimTestAVD(t, "hw.ramSize=1536\nhw.gpu.mode=host\n", tunedManifest())

	command := slimTestCommand()
	if err := command.Flags().Set("keep-audio", "true"); err != nil {
		t.Fatal(err)
	}
	tune, ram, gpu := slimTuneState(command, a, 2048, "auto")
	if !tune {
		t.Fatal("tune = false, want a retune for --keep-audio")
	}
	if ram != 1536 || gpu != "host" {
		t.Fatalf("ram/gpu = %d/%q, want the tuned config values", ram, gpu)
	}
}

func TestSlimTuneStateFreshAVD(t *testing.T) {
	a := slimTestAVD(t, "hw.ramSize=4096\nhw.gpu.mode=auto\n", nil)

	tune, ram, gpu := slimTuneState(slimTestCommand(), a, 2048, "host")
	if !tune || ram != 2048 || gpu != "host" {
		t.Fatalf("tune/ram/gpu = %v/%d/%q, want the command defaults", tune, ram, gpu)
	}
}

func TestValidateSlimSkip(t *testing.T) {
	if err := validateSlimSkip([]string{"media", "telephony"}); err != nil {
		t.Fatalf("validateSlimSkip(valid) = %v", err)
	}
	if err := validateSlimSkip([]string{"everything"}); err == nil {
		t.Fatal("validateSlimSkip accepted an unknown group")
	}
}
