package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadSQLFromArg(t *testing.T) {
	got, err := readSQL([]string{"SELECT 1"}, "", bytes.NewBufferString(""))
	if err != nil {
		t.Fatalf("readSQL: %v", err)
	}
	if got != "SELECT 1" {
		t.Errorf("got %q", got)
	}
}

func TestReadSQLFromStdinDash(t *testing.T) {
	got, err := readSQL(nil, "-", bytes.NewBufferString("SELECT 2"))
	if err != nil {
		t.Fatalf("readSQL: %v", err)
	}
	if got != "SELECT 2" {
		t.Errorf("got %q", got)
	}
}

func TestReadSQLArgAndFileMutualExclusive(t *testing.T) {
	_, err := readSQL([]string{"SELECT 1"}, "some.sql", bytes.NewBufferString(""))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestReadSQLFromFile(t *testing.T) {
	f := filepath.Join(t.TempDir(), "q.sql")
	if err := os.WriteFile(f, []byte("SELECT 3"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readSQL(nil, f, bytes.NewBufferString(""))
	if err != nil {
		t.Fatalf("readSQL: %v", err)
	}
	if strings.TrimSpace(got) != "SELECT 3" {
		t.Errorf("got %q", got)
	}
}

func TestParseParamsMerges(t *testing.T) {
	got, err := parseParams(
		[]string{"limit=100", "active=true", "country=US"},
		`{"override":"yes","limit":200}`,
	)
	if err != nil {
		t.Fatalf("parseParams: %v", err)
	}
	// --params JSON takes precedence.
	if got["limit"].(float64) != 200 {
		t.Errorf("limit: %v", got["limit"])
	}
	if got["active"] != true {
		t.Errorf("active: %v", got["active"])
	}
	if got["country"] != "US" {
		t.Errorf("country: %v", got["country"])
	}
	if got["override"] != "yes" {
		t.Errorf("override: %v", got["override"])
	}
}

func TestParseParamsInvalidPair(t *testing.T) {
	_, err := parseParams([]string{"bad-no-equals"}, "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseParamsInvalidJSON(t *testing.T) {
	_, err := parseParams(nil, `not json`)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCoerceParam(t *testing.T) {
	cases := []struct {
		in   string
		want any
	}{
		{"", ""},
		{"true", true},
		{"false", false},
		{"42", int64(42)},
		{"3.14", 3.14},
		{"hello", "hello"},
	}
	for _, tc := range cases {
		if got := coerceParam(tc.in); got != tc.want {
			t.Errorf("coerceParam(%q) = %v (%T), want %v", tc.in, got, got, tc.want)
		}
	}
}
