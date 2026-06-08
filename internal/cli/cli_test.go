package cli

import "testing"

func TestRunVersionFlag(t *testing.T) {
	if code := Run("duck", nil, []string{"--version"}); code != 0 {
		t.Fatalf("Run(--version) = %d, want 0", code)
	}
	if code := Run("duck", nil, []string{"-V"}); code != 0 {
		t.Fatalf("Run(-V) = %d, want 0", code)
	}
}

func TestRunVersionFlagWithGlobalOptions(t *testing.T) {
	if code := Run("duck", nil, []string{"--quiet", "--version"}); code != 0 {
		t.Fatalf("Run(--quiet --version) = %d, want 0", code)
	}
}
