package app

import (
	"testing"

	"github.com/IKauedev/duck/internal/terminal"
)

func TestParseTerminalLine(t *testing.T) {
	got, err := terminal.ParseLine(`docker logs "api service" --tail 10`)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"docker", "logs", "api service", "--tail", "10"}
	if len(got) != len(want) {
		t.Fatalf("ParseLine() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ParseLine() = %#v, want %#v", got, want)
		}
	}
}

func TestParseTerminalLineUnclosedQuote(t *testing.T) {
	if _, err := terminal.ParseLine(`docker logs "api`); err == nil {
		t.Fatal("expected error for unclosed quote")
	}
}
