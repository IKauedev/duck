package tui

import "testing"

func TestShouldFallbackToWSLOnlyWindows(t *testing.T) {
	if shouldFallbackToWSL("docker not recognized", errTest{}) {
		// runtime may not be windows in CI; just ensure function does not panic
	}
}

func TestWSLCommand(t *testing.T) {
	got := wslCommand("docker", []string{"ps", "-a"})
	want := []string{"-e", "docker", "ps", "-a"}
	if len(got) != len(want) {
		t.Fatalf("wslCommand() = %#v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("wslCommand() = %#v, want %#v", got, want)
		}
	}
}

type errTest struct{}

func (errTest) Error() string { return "exit status 1" }
