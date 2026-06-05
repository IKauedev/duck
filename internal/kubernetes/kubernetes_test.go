package kubernetes

import (
	"reflect"
	"testing"
)

func TestNamespaceArgs(t *testing.T) {
	namespace, rest, err := namespaceArgs([]string{"api", "-n", "apps", "--tail", "100"})
	if err != nil {
		t.Fatal(err)
	}
	if namespace != "apps" {
		t.Fatalf("namespace = %q, want apps", namespace)
	}
	if !reflect.DeepEqual(rest, []string{"api", "--tail", "100"}) {
		t.Fatalf("rest = %#v", rest)
	}
}

func TestStripForce(t *testing.T) {
	force, rest := stripForce([]string{"deployment", "api", "--yes"})
	if !force {
		t.Fatal("force = false, want true")
	}
	if !reflect.DeepEqual(rest, []string{"deployment", "api"}) {
		t.Fatalf("rest = %#v", rest)
	}
}

func TestNormalizeCurlURL(t *testing.T) {
	got, err := normalizeCurlURL("example.com/health", "8080")
	if err != nil {
		t.Fatal(err)
	}
	want := "http://example.com:8080/health"
	if got != want {
		t.Fatalf("normalizeCurlURL() = %q, want %q", got, want)
	}
}

func TestNormalizeCurlURLKeepsExistingPort(t *testing.T) {
	got, err := normalizeCurlURL("https://example.com:8443/health", "443")
	if err != nil {
		t.Fatal(err)
	}
	want := "https://example.com:8443/health"
	if got != want {
		t.Fatalf("normalizeCurlURL() = %q, want %q", got, want)
	}
}
