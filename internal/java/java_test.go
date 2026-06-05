package java

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateJavaHome(t *testing.T) {
	home := t.TempDir()
	bin := filepath.Join(home, "bin")
	if err := os.MkdirAll(bin, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, executable("java")), []byte{}, 0755); err != nil {
		t.Fatal(err)
	}

	if err := validateJavaHome(home); err != nil {
		t.Fatalf("validateJavaHome() error = %v", err)
	}
}

func TestShellQuote(t *testing.T) {
	got := shellQuote("/opt/java's/jdk")
	want := "'/opt/java'\"'\"'s/jdk'"
	if got != want {
		t.Fatalf("shellQuote() = %q, want %q", got, want)
	}
}
