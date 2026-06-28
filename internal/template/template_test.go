package template

import "testing"

func TestSanitizeName(t *testing.T) {
	tests := map[string]string{
		"My API":    "my-api",
		"my_api":    "my-api",
		"  hello  ": "hello",
		"...":       "",
	}
	for input, want := range tests {
		got := sanitizeName(input)
		if got != want {
			t.Fatalf("sanitizeName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestFindTemplateAliases(t *testing.T) {
	cases := []string{"docker", "compose", "terraform", "jenkins", "helm", "kubernetes", "go", "tf", "k8s", "ci"}
	for _, id := range cases {
		if _, ok := findTemplate(id); !ok {
			t.Fatalf("findTemplate(%q) not found", id)
		}
	}
}

func TestRender(t *testing.T) {
	got := render("hello {{ProjectName}} / {{ProjectNameLower}}", "My-App")
	want := "hello My-App / my-app"
	if got != want {
		t.Fatalf("render() = %q, want %q", got, want)
	}
}
