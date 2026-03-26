package parser

import (
	"testing"
)

func TestParseRenovateBody_Table(t *testing.T) {
	body := `## This PR contains the following updates

| Package | Current | New |
|---|---|---|
| cert-manager | ` + "`1.14.5`" + ` | ` + "`1.14.7`" + ` |
| cluster-autoscaler | ` + "`1.29.0`" + ` | ` + "`1.30.1`" + ` |

### Release Notes
https://github.com/cert-manager/cert-manager/releases
`

	results := ParseRenovateBody(body)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].PackageName != "cert-manager" {
		t.Errorf("expected package 'cert-manager', got %q", results[0].PackageName)
	}
	if results[0].FromVersion != "1.14.5" {
		t.Errorf("expected from '1.14.5', got %q", results[0].FromVersion)
	}
	if results[0].ToVersion != "1.14.7" {
		t.Errorf("expected to '1.14.7', got %q", results[0].ToVersion)
	}

	if results[1].PackageName != "cluster-autoscaler" {
		t.Errorf("expected package 'cluster-autoscaler', got %q", results[1].PackageName)
	}
	if results[1].FromVersion != "1.29.0" {
		t.Errorf("expected from '1.29.0', got %q", results[1].FromVersion)
	}
	if results[1].ToVersion != "1.30.1" {
		t.Errorf("expected to '1.30.1', got %q", results[1].ToVersion)
	}
}

func TestParseRenovateBody_UpdateLine(t *testing.T) {
	body := "Update `cert-manager` from v1.14.5 to v1.14.7\n"

	results := ParseRenovateBody(body)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].PackageName != "cert-manager" {
		t.Errorf("expected 'cert-manager', got %q", results[0].PackageName)
	}
	if results[0].FromVersion != "1.14.5" {
		t.Errorf("expected '1.14.5', got %q", results[0].FromVersion)
	}
	if results[0].ToVersion != "1.14.7" {
		t.Errorf("expected '1.14.7', got %q", results[0].ToVersion)
	}
}

func TestParseRenovateBody_Empty(t *testing.T) {
	results := ParseRenovateBody("")
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty body, got %d", len(results))
	}
}

func TestCleanVersion(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"v1.2.3", "1.2.3"},
		{"1.2.3", "1.2.3"},
		{" v1.2.3 ", "1.2.3"},
	}
	for _, tt := range tests {
		got := cleanVersion(tt.input)
		if got != tt.want {
			t.Errorf("cleanVersion(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
