package app

import (
	"reflect"
	"testing"
)

func TestParseTerminalLine(t *testing.T) {
	got, err := parseTerminalLine(`docker logs "api service" --tail 10`)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"docker", "logs", "api service", "--tail", "10"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseTerminalLine() = %#v, want %#v", got, want)
	}
}

func TestParseTerminalLineUnclosedQuote(t *testing.T) {
	if _, err := parseTerminalLine(`docker logs "api`); err == nil {
		t.Fatal("expected error for unclosed quote")
	}
}
