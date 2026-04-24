package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/leroy/redash/internal/client"
)

func TestParseFormat(t *testing.T) {
	cases := []struct {
		in   string
		want Format
		err  bool
	}{
		{"", FormatTable, false},
		{"TABLE", FormatTable, false},
		{"json", FormatJSON, false},
		{"CSV", FormatCSV, false},
		{"xml", "", true},
	}
	for _, tc := range cases {
		got, err := ParseFormat(tc.in)
		if tc.err {
			if err == nil {
				t.Errorf("%q: expected error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("%q: %v", tc.in, err)
		}
		if got != tc.want {
			t.Errorf("%q: got %q, want %q", tc.in, got, tc.want)
		}
	}
}

func sampleResult() *client.QueryResult {
	return &client.QueryResult{
		ID: 1, Query: "SELECT 1", Runtime: 0.1,
		Data: client.QueryResultData{
			Columns: []client.QueryResultColumn{
				{Name: "id", Type: "int"},
				{Name: "name", Type: "string"},
				{Name: "active", Type: "bool"},
			},
			Rows: []map[string]any{
				{"id": float64(1), "name": "alice", "active": true},
				{"id": float64(2), "name": "bob,junior", "active": false},
			},
		},
	}
}

func TestQueryResultCSV(t *testing.T) {
	var buf bytes.Buffer
	if err := QueryResult(&buf, sampleResult(), FormatCSV); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("lines: %d\n%s", len(lines), out)
	}
	if lines[0] != "id,name,active" {
		t.Errorf("header: %q", lines[0])
	}
	if lines[1] != "1,alice,true" {
		t.Errorf("row 1: %q", lines[1])
	}
	// Comma in value must be quoted.
	if !strings.Contains(lines[2], `"bob,junior"`) {
		t.Errorf("row 2 not quoted: %q", lines[2])
	}
}

func TestQueryResultJSONPreservesTypes(t *testing.T) {
	var buf bytes.Buffer
	if err := QueryResult(&buf, sampleResult(), FormatJSON); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := buf.String()
	// Numbers without quotes, booleans without quotes.
	if !strings.Contains(out, `"id":1`) && !strings.Contains(out, `"id": 1`) {
		t.Errorf("missing numeric id: %s", out)
	}
	if !strings.Contains(out, `"active":true`) && !strings.Contains(out, `"active": true`) {
		t.Errorf("missing bool: %s", out)
	}
}

func TestQueryResultTableRenders(t *testing.T) {
	var buf bytes.Buffer
	if err := QueryResult(&buf, sampleResult(), FormatTable); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "alice") || !strings.Contains(out, "bob,junior") {
		t.Errorf("missing row content: %s", out)
	}
	// Header should appear.
	if !strings.Contains(strings.ToLower(out), "id") || !strings.Contains(strings.ToLower(out), "name") {
		t.Errorf("missing header: %s", out)
	}
}

func TestRecordsCSV(t *testing.T) {
	r := Records{
		Columns: []string{"a", "b"},
		Rows:    [][]string{{"1", "x"}, {"2", "y"}},
	}
	var buf bytes.Buffer
	if err := r.Render(&buf, FormatCSV); err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := buf.String(); got != "a,b\n1,x\n2,y\n" {
		t.Errorf("csv: %q", got)
	}
}

func TestRecordsJSON(t *testing.T) {
	r := Records{
		Columns: []string{"a", "b"},
		Rows:    [][]string{{"1", "x"}},
	}
	var buf bytes.Buffer
	if err := r.Render(&buf, FormatJSON); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := strings.TrimSpace(buf.String())
	if out != `[{"a":"1","b":"x"}]` {
		t.Errorf("json: %q", out)
	}
}
