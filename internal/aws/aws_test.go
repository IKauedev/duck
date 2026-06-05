package aws

import (
	"reflect"
	"testing"
)

func TestSplitLogsArgs(t *testing.T) {
	flags, rest, err := splitLogsArgs([]string{"/aws/lambda/api", "--follow", "--since", "10m", "--profile", "dev"})
	if err != nil {
		t.Fatal(err)
	}

	wantFlags := []string{"--follow", "--since", "10m", "--profile", "dev"}
	wantRest := []string{"/aws/lambda/api"}
	if !reflect.DeepEqual(flags, wantFlags) {
		t.Fatalf("flags = %#v, want %#v", flags, wantFlags)
	}
	if !reflect.DeepEqual(rest, wantRest) {
		t.Fatalf("rest = %#v, want %#v", rest, wantRest)
	}
}

func TestParseCostArgs(t *testing.T) {
	days, flags, err := parseCostArgs([]string{"--days", "7", "--region", "us-east-1"})
	if err != nil {
		t.Fatal(err)
	}
	if days != 7 {
		t.Fatalf("days = %d, want 7", days)
	}
	if !reflect.DeepEqual(flags, []string{"--region", "us-east-1"}) {
		t.Fatalf("flags = %#v", flags)
	}
}

func TestSplitProfileRegionAlias(t *testing.T) {
	flags, rest, err := splitProfileRegionAlias([]string{"cluster", "--alias", "dev", "--profile", "admin"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(flags, []string{"--alias", "dev", "--profile", "admin"}) {
		t.Fatalf("flags = %#v", flags)
	}
	if !reflect.DeepEqual(rest, []string{"cluster"}) {
		t.Fatalf("rest = %#v", rest)
	}
}
