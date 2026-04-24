package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadJSONArgInline(t *testing.T) {
	got, err := readJSONArg(`  [{"a":1}]  `)
	if err != nil {
		t.Fatalf("readJSONArg: %v", err)
	}
	if string(got) != `[{"a":1}]` {
		t.Errorf("got %q", got)
	}
}

func TestReadJSONArgFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "params.json")
	if err := os.WriteFile(p, []byte(`{"b": 2}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readJSONArg(p)
	if err != nil {
		t.Fatalf("readJSONArg: %v", err)
	}
	if string(got) != `{"b":2}` {
		t.Errorf("got %q", got)
	}
}

func TestReadJSONArgStdin(t *testing.T) {
	// readJSONArg reads from os.Stdin; swap it for this test.
	orig := os.Stdin
	defer func() { os.Stdin = orig }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdin = r
	go func() {
		defer w.Close()
		_, _ = io.Copy(w, bytes.NewBufferString(`[1,2,3]`))
	}()

	got, err := readJSONArg("-")
	if err != nil {
		t.Fatalf("readJSONArg: %v", err)
	}
	if string(got) != `[1,2,3]` {
		t.Errorf("got %q", got)
	}
}

func TestReadJSONArgInvalidJSON(t *testing.T) {
	_, err := readJSONArg(`{not json`)
	if err == nil || !strings.Contains(err.Error(), "invalid JSON") {
		t.Fatalf("expected invalid JSON error, got %v", err)
	}
}

func TestReadJSONArgMissingFile(t *testing.T) {
	_, err := readJSONArg(filepath.Join(t.TempDir(), "nope.json"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestReadJSONArgEmpty(t *testing.T) {
	_, err := readJSONArg("   ")
	if err == nil {
		t.Fatal("expected error for empty value")
	}
}
