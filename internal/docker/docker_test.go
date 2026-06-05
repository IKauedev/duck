package docker

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNonEmptyLines(t *testing.T) {
	got := nonEmptyLines("one\n\n two \r\nthree")
	want := []string{"one", "two", "three"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("nonEmptyLines() = %#v, want %#v", got, want)
	}
}

func TestFindComposeFile(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "app", "api")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}
	compose := filepath.Join(root, "compose.yaml")
	if err := os.WriteFile(compose, []byte("services: {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	got, ok := findComposeFile(nested)
	if !ok {
		t.Fatal("findComposeFile() did not find compose file")
	}
	if got != compose {
		t.Fatalf("findComposeFile() = %q, want %q", got, compose)
	}
}
