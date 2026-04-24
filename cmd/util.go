package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// readJSONArg accepts either inline JSON (starts with `{` or `[`),
// a file path whose contents are JSON, or `-` to read from stdin.
// Returns canonicalized (compact) JSON bytes.
//
// Agents: pass inline JSON directly. Humans: pass a file path.
func readJSONArg(val string) (json.RawMessage, error) {
	trim := strings.TrimSpace(val)
	if trim == "" {
		return nil, fmt.Errorf("empty value")
	}

	var data []byte
	switch {
	case trim == "-":
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		data = b
	case strings.HasPrefix(trim, "{") || strings.HasPrefix(trim, "["):
		data = []byte(val)
	default:
		b, err := os.ReadFile(val)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", val, err)
		}
		data = b
	}

	// Validate that it is JSON, and compact it.
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	out, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return out, nil
}
