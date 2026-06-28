package release

import "testing"

func TestAssetFileName(t *testing.T) {
	if got := AssetFileName("windows", "amd64"); got != "duck_windows_amd64.zip" {
		t.Fatalf("AssetFileName() = %q", got)
	}
	if got := AssetFileName("linux", "arm64"); got != "duck_linux_arm64.tar.gz" {
		t.Fatalf("AssetFileName() = %q", got)
	}
}

func TestDirectAssetURL(t *testing.T) {
	got := DirectAssetURL("", "windows", "amd64")
	want := "https://github.com/IKauedev/duck/releases/latest/download/duck_windows_amd64.zip"
	if got != want {
		t.Fatalf("DirectAssetURL() = %q, want %q", got, want)
	}

	got = DirectAssetURL("v1.2.3", "linux", "amd64")
	want = "https://github.com/IKauedev/duck/releases/download/v1.2.3/duck_linux_amd64.tar.gz"
	if got != want {
		t.Fatalf("DirectAssetURL() = %q, want %q", got, want)
	}
}
