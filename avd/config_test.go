package avd

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.ini")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestConfigSetPreservesCommentsAndOrder(t *testing.T) {
	path := writeConfig(t, "# comment\nhw.ramSize=2048\nhw.gpu.mode = auto\n\nhw.keyboard=yes\n")
	config, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}

	old, existed := config.Set("hw.gpu.mode", "host")
	if old != "auto" || !existed {
		t.Fatalf("Set() = %q, %v, want auto, true", old, existed)
	}
	config.Set("hw.audioInput", "no")

	if err := config.Save(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	want := "# comment\nhw.ramSize=2048\nhw.gpu.mode=host\n\nhw.keyboard=yes\nhw.audioInput=no\n"
	if string(got) != want {
		t.Fatalf("config =\n%q\nwant\n%q", got, want)
	}
}

func TestConfigSetDropsDuplicateKeys(t *testing.T) {
	config, err := LoadConfig(writeConfig(t, "a=1\na=2\nb=3\n"))
	if err != nil {
		t.Fatal(err)
	}

	old, existed := config.Set("a", "4")
	if old != "1" || !existed {
		t.Fatalf("Set() = %q, %v, want 1, true", old, existed)
	}
	if got, want := string(config.Bytes()), "a=4\nb=3\n"; got != want {
		t.Fatalf("config = %q, want %q", got, want)
	}
}

func TestConfigDelete(t *testing.T) {
	config, err := LoadConfig(writeConfig(t, "# keep\na=1\nb=2\n"))
	if err != nil {
		t.Fatal(err)
	}

	old, existed := config.Delete("a")
	if old != "1" || !existed {
		t.Fatalf("Delete() = %q, %v, want 1, true", old, existed)
	}
	if got, want := string(config.Bytes()), "# keep\nb=2\n"; got != want {
		t.Fatalf("config = %q, want %q", got, want)
	}
}

func TestConfigPreservesCRLF(t *testing.T) {
	config, err := LoadConfig(writeConfig(t, "a=1\r\nb=2\r\n"))
	if err != nil {
		t.Fatal(err)
	}

	config.Set("b", "3")
	if got, want := string(config.Bytes()), "a=1\r\nb=3\r\n"; got != want {
		t.Fatalf("config = %q, want %q", got, want)
	}
}
