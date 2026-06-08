package version

import "testing"

func TestNormalizeTag(t *testing.T) {
	tests := map[string]string{
		"v0.1.2":         "0.1.2",
		"0.1.2":          "0.1.2",
		"refs/tags/v1.0": "1.0",
		"":               "",
	}
	for input, want := range tests {
		if got := normalizeTag(input); got != want {
			t.Fatalf("normalizeTag(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestMatchesTag(t *testing.T) {
	original := Version
	defer func() { Version = original }()

	Version = "0.1.2"
	if !MatchesTag("v0.1.2") {
		t.Fatal("expected v0.1.2 to match 0.1.2")
	}
	if MatchesTag("v0.1.3") {
		t.Fatal("expected v0.1.3 not to match 0.1.2")
	}

	Version = "dev"
	if MatchesTag("v0.1.2") {
		t.Fatal("dev should not match release tag")
	}
}

func TestLabel(t *testing.T) {
	original := Version
	defer func() { Version = original }()

	Version = "1.2.3"
	if got := Label(); got != "1.2.3" {
		t.Fatalf("Label() = %q, want 1.2.3", got)
	}

	Version = "dev"
	if got := Label(); got != "dev" {
		t.Fatalf("Label() = %q, want dev", got)
	}
}
