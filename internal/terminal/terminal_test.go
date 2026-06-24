package terminal

import (
	"reflect"
	"testing"

	"github.com/IKauedev/duck/internal/cli"
)

func TestParseLine(t *testing.T) {
	got, err := ParseLine(`docker logs "api service" --tail 10`)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"docker", "logs", "api service", "--tail", "10"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseLine() = %#v, want %#v", got, want)
	}
}

func TestParseLineUnclosedQuote(t *testing.T) {
	if _, err := ParseLine(`docker logs "api`); err == nil {
		t.Fatal("expected error for unclosed quote")
	}
}

func TestIsDuckCommand(t *testing.T) {
	commands := []cli.Command{
		{Name: "status"},
		{Name: "docker", Children: []cli.Command{
			{Name: "ps"},
			{Name: "logs"},
		}},
		{Name: "help"},
	}

	tests := []struct {
		args []string
		want bool
	}{
		{[]string{"status"}, true},
		{[]string{"docker", "ps"}, true},
		{[]string{"docker"}, true},
		{[]string{"help"}, true},
		{[]string{"dir"}, false},
		{[]string{"ls", "-la"}, false},
		{[]string{"echo", "hello"}, false},
	}

	for _, tc := range tests {
		got := IsDuckCommand(tc.args, commands)
		if got != tc.want {
			t.Fatalf("IsDuckCommand(%v) = %v, want %v", tc.args, got, tc.want)
		}
	}
}
