package netcheck

import "testing"

func TestNormalizeURL(t *testing.T) {
	got, err := NormalizeURL("example.com/health", "8080")
	if err != nil {
		t.Fatal(err)
	}
	want := "http://example.com:8080/health"
	if got != want {
		t.Fatalf("NormalizeURL() = %q, want %q", got, want)
	}
}

func TestNormalizeURLKeepsExistingPort(t *testing.T) {
	got, err := NormalizeURL("https://example.com:8443/health", "443")
	if err != nil {
		t.Fatal(err)
	}
	want := "https://example.com:8443/health"
	if got != want {
		t.Fatalf("NormalizeURL() = %q, want %q", got, want)
	}
}
