package tui

import "testing"

func TestParseKubeDeployments(t *testing.T) {
	input := `{
		"items": [{
			"metadata": {"namespace":"default","name":"api","creationTimestamp":"2026-06-06T10:00:00Z"},
			"spec": {"replicas": 3},
			"status": {"replicas": 3, "readyReplicas": 2, "updatedReplicas": 3, "availableReplicas": 2}
		}]
	}`
	rows, err := parseKubeDeployments(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ColA != "2/3" {
		t.Fatalf("rows = %#v", rows)
	}
}

func TestParseKubeIngress(t *testing.T) {
	input := `{
		"items": [{
			"metadata": {"namespace":"default","name":"web","creationTimestamp":"2026-06-06T10:00:00Z"},
			"spec": {"ingressClassName":"nginx","rules":[{"host":"app.local"}]},
			"status": {"loadBalancer":{"ingress":[{"ip":"10.0.0.5"}]}}
		}]
	}`
	rows, err := parseKubeIngress(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ColB != "app.local" {
		t.Fatalf("rows = %#v", rows)
	}
}

func TestKubeResourceCycle(t *testing.T) {
	if kubeResPods.next() != kubeResDeployments {
		t.Fatal("next resource mismatch")
	}
	if kubeResContexts.next() != kubeResPods {
		t.Fatal("contexts should wrap to pods")
	}
}
