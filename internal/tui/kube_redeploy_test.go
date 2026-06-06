package tui

import (
	"strings"
	"testing"
)

func TestDeploymentLabelSelectorTrim(t *testing.T) {
	raw := "app=api,version=v1,"
	got := strings.TrimSuffix(strings.TrimSpace(raw), ",")
	want := "app=api,version=v1"
	if got != want {
		t.Fatalf("trim = %q, want %q", got, want)
	}
}

func TestKubeSupportsYAML(t *testing.T) {
	m := model{kubeResource: kubeResIngress}
	if !m.kubeSupportsYAML() {
		t.Fatal("ingress should support yaml")
	}
	m.kubeResource = kubeResEvents
	if m.kubeSupportsYAML() {
		t.Fatal("events should not support yaml")
	}
}

func TestIsYAMLDetail(t *testing.T) {
	if !isYAMLDetail("YAML: default/api") {
		t.Fatal("expected yaml detail")
	}
}
