package tui

import (
	"testing"
	"time"
)

func TestParseRefreshDuration(t *testing.T) {
	got, err := parseRefreshDuration("2s")
	if err != nil {
		t.Fatal(err)
	}
	if got != 2*time.Second {
		t.Fatalf("parseRefreshDuration() = %s", got)
	}
}

func TestOptionsNeedsConfirm(t *testing.T) {
	opts := Options{Confirm: confirmNever}
	if opts.needsConfirm("delete") {
		t.Fatal("never should skip confirm")
	}
	opts.Confirm = confirmAlways
	if !opts.needsConfirm("start") {
		t.Fatal("always should confirm start")
	}
	opts.Confirm = confirmDestructive
	if opts.needsConfirm("start") {
		t.Fatal("destructive should not confirm start")
	}
	if !opts.needsConfirm("delete") {
		t.Fatal("destructive should confirm delete")
	}
}

func TestParseOptionsCompact(t *testing.T) {
	opts, err := ParseOptions([]string{"--compact", "--readonly"})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.Compact || !opts.Readonly {
		t.Fatalf("opts = %#v", opts)
	}
}
