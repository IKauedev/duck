package tui

import "testing"

func TestParseDockerRows(t *testing.T) {
	input := "api\tUp 2 hours (healthy)\tnginx:latest\t0.0.0.0:80->80/tcp\trunning\n" +
		"db\tExited (1) 1 day ago\tpostgres:16\t\texited\n"

	rows := parseDockerRows(input)
	if len(rows) != 2 {
		t.Fatalf("parseDockerRows() len = %d, want 2", len(rows))
	}
	if rows[0].Name != "api" || rows[0].Health != "healthy" {
		t.Fatalf("first row = %#v", rows[0])
	}
	if rows[1].State != "exited" {
		t.Fatalf("second row state = %q", rows[1].State)
	}
}

func TestHealthFromDockerStatus(t *testing.T) {
	if got := healthFromDockerStatus("Up 2 hours (healthy)"); got != "healthy" {
		t.Fatalf("health = %q", got)
	}
	if got := healthFromDockerStatus("Up 10 seconds (health: starting)"); got != "health: starting" {
		t.Fatalf("health = %q", got)
	}
	if got := healthFromDockerStatus("Exited (1) 1 day ago"); got != "" {
		t.Fatalf("health = %q", got)
	}
}

func TestFilterDockerRows(t *testing.T) {
	rows := []dockerRow{
		{Name: "api", Status: "Up", Image: "nginx"},
		{Name: "db", Status: "Exited", Image: "postgres"},
	}
	filtered := filterDockerRows(rows, "post")
	if len(filtered) != 1 || filtered[0].Name != "db" {
		t.Fatalf("filterDockerRows() = %#v", filtered)
	}
}

func TestDockerStatusStyleColors(t *testing.T) {
	healthy := dockerStatusStyle("Up 2 hours", "healthy", "running").Render("ok")
	unhealthy := dockerStatusStyle("Up 2 hours", "unhealthy", "running").Render("bad")
	exited := dockerStatusStyle("Exited (1)", "", "exited").Render("off")

	if healthy == unhealthy || healthy == exited || unhealthy == exited {
		t.Fatal("expected distinct styles for healthy, unhealthy and exited containers")
	}
}

func TestKubeStatusStyleColors(t *testing.T) {
	running := kubeStatusStyle("Running", "Running").Render("ok")
	pending := kubeStatusStyle("Pending", "Pending").Render("wait")
	crash := kubeStatusStyle("Running", "CrashLoopBackOff").Render("bad")

	if running == pending || running == crash || pending == crash {
		t.Fatal("expected distinct styles for running, pending and crashloop pods")
	}
}

func TestParseKubeRows(t *testing.T) {
	input := `{
		"items": [{
			"metadata": {"namespace":"default","name":"api-1","creationTimestamp":"2026-06-06T10:00:00Z"},
			"status": {
				"phase":"Running",
				"containerStatuses":[{"ready":true,"restartCount":2,"state":{"running":{}}}]
			}
		}]
	}`

	rows, err := parseKubeRows(kubeResPods, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("len = %d", len(rows))
	}
	if rows[0].Namespace != "default" || rows[0].Name != "api-1" || rows[0].Restarts != 2 {
		t.Fatalf("row = %#v", rows[0])
	}
}

func TestParseKubeRowsPodsAlias(t *testing.T) {
	input := `{"items":[{"metadata":{"namespace":"default","name":"api-1","creationTimestamp":"2026-06-06T10:00:00Z"},"status":{"phase":"Running","containerStatuses":[{"ready":true,"restartCount":0,"state":{"running":{}}}]}}]}`
	rows, err := parseKubeRows(kubeResPods, input)
	if err != nil || len(rows) != 1 {
		t.Fatalf("parseKubeRows() = %#v, %v", rows, err)
	}
}
