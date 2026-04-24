package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/leroy/redash/internal/manual"
)

// TestManualCoversEverySubcommand fails if a new top-level subcommand is
// added without a corresponding entry in the manual. This is the drift
// guard described in the "For AI agents" section of the README: the
// agent-facing docs must keep pace with the command tree.
func TestManualCoversEverySubcommand(t *testing.T) {
	covered := manual.CommandTopics()

	for _, c := range Root.Commands() {
		name := c.Name()
		// Cobra auto-adds "help" and "completion"; those don't need entries.
		if name == "help" || name == "completion" {
			continue
		}
		if _, ok := covered[name]; !ok {
			t.Errorf("command %q has no manual entry — add one in internal/manual/manual.go", name)
		}
	}
}

// TestManualCommandRunsMarkdown checks the `manual` command without flags
// prints markdown that contains every topic title.
func TestManualCommandRunsMarkdown(t *testing.T) {
	var buf bytes.Buffer
	c := newManualCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{})
	if err := c.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := buf.String()
	if !strings.HasPrefix(out, "# redash CLI") {
		t.Errorf("expected manual to start with top-level heading, got: %q", out[:min(80, len(out))])
	}
	for _, topic := range manual.Topics() {
		if !strings.Contains(out, topic.Title) {
			t.Errorf("manual output missing topic title %q", topic.Title)
		}
	}
}

func TestManualCommandRunsJSON(t *testing.T) {
	var buf bytes.Buffer
	c := newManualCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"--format", "json"})
	if err := c.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	var got manual.JSONCatalog
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if got.CLI != "redash" {
		t.Errorf("cli: %q", got.CLI)
	}
	if len(got.Topics) != len(manual.Topics()) {
		t.Errorf("topic count: got %d, want %d", len(got.Topics), len(manual.Topics()))
	}
}

func TestManualCommandSingleTopic(t *testing.T) {
	var buf bytes.Buffer
	c := newManualCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"--topic", "query"})
	if err := c.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "redash query") {
		t.Errorf("expected query section, got: %q", out[:min(120, len(out))])
	}
	// And MUST NOT contain unrelated sections.
	if strings.Contains(out, "## Error handling") {
		t.Errorf("single-topic output should not include other sections")
	}
}

func TestManualCommandListTopics(t *testing.T) {
	var buf bytes.Buffer
	c := newManualCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"--list-topics"})
	if err := c.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := buf.String()
	// Each topic name should appear as the first token on a line.
	for _, topic := range manual.Topics() {
		if !strings.Contains(out, topic.Name) {
			t.Errorf("--list-topics missing %q", topic.Name)
		}
	}
}

func TestManualUnknownTopic(t *testing.T) {
	var buf bytes.Buffer
	c := newManualCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SilenceErrors = true
	c.SilenceUsage = true
	c.SetArgs([]string{"--topic", "nonsense"})
	err := c.Execute()
	if err == nil {
		t.Fatal("expected error for unknown topic")
	}
	if !strings.Contains(err.Error(), "available:") {
		t.Errorf("error should list available topics, got: %v", err)
	}
}
