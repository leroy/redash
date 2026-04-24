package client

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// TestUpdateQueryParametersPreservesOtherOptions verifies the fetch-merge-
// patch flow: existing options.anything that isn't "parameters" must
// survive the update, and the new parameters must replace the old ones.
func TestUpdateQueryParametersPreservesOtherOptions(t *testing.T) {
	var updateBody map[string]any
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/queries/42":
			_, _ = w.Write([]byte(`{
				"id": 42,
				"name": "Q",
				"query": "SELECT 1",
				"data_source_id": 3,
				"options": {
					"parameters": [{"name":"old","type":"text","value":"x"}],
					"runAtLoad": true,
					"apply_auto_limit": false
				}
			}`))
		case r.Method == "POST" && r.URL.Path == "/api/queries/42":
			_ = json.NewDecoder(r.Body).Decode(&updateBody)
			_, _ = w.Write([]byte(`{"id":42,"name":"Q","query":"SELECT 1"}`))
		default:
			t.Errorf("unexpected call: %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected", 500)
		}
	})

	newParams := json.RawMessage(`[{"name":"country","title":"Country","type":"text","value":"US","global":false}]`)
	if _, err := c.UpdateQueryParameters(context.Background(), 42, newParams); err != nil {
		t.Fatalf("UpdateQueryParameters: %v", err)
	}

	opts, ok := updateBody["options"].(map[string]any)
	if !ok {
		t.Fatalf("options not in update body: %+v", updateBody)
	}

	// New parameters replaced the old.
	params, ok := opts["parameters"].([]any)
	if !ok || len(params) != 1 {
		t.Fatalf("parameters not replaced: %v", opts["parameters"])
	}
	p0 := params[0].(map[string]any)
	if p0["name"] != "country" || p0["value"] != "US" {
		t.Errorf("new parameter not set: %+v", p0)
	}

	// Other options fields preserved.
	if opts["runAtLoad"] != true {
		t.Errorf("runAtLoad not preserved: %v", opts["runAtLoad"])
	}
	if opts["apply_auto_limit"] != false {
		t.Errorf("apply_auto_limit not preserved: %v", opts["apply_auto_limit"])
	}

	// Only options should be touched (not name/query/etc).
	if _, ok := updateBody["name"]; ok {
		t.Errorf("name should not be sent on parameter-only update: %v", updateBody)
	}
}

// TestUpdateQueryParametersRejectsNonArray ensures a shape error is
// returned, not silently accepted.
func TestUpdateQueryParametersRejectsNonArray(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			_, _ = w.Write([]byte(`{"id":1,"options":{}}`))
			return
		}
		t.Error("should not reach POST")
	})
	_, err := c.UpdateQueryParameters(context.Background(), 1, json.RawMessage(`{"not":"an array"}`))
	if err == nil || !strings.Contains(err.Error(), "array") {
		t.Fatalf("expected array-shape error, got %v", err)
	}
}

// TestUpdateQueryParametersNoExistingOptions works when options is
// absent or empty on the existing query.
func TestUpdateQueryParametersNoExistingOptions(t *testing.T) {
	var updateBody map[string]any
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			_, _ = w.Write([]byte(`{"id":1}`))
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&updateBody)
		_, _ = w.Write([]byte(`{"id":1}`))
	})
	_, err := c.UpdateQueryParameters(context.Background(), 1, json.RawMessage(`[]`))
	if err != nil {
		t.Fatalf("UpdateQueryParameters: %v", err)
	}
	opts, ok := updateBody["options"].(map[string]any)
	if !ok {
		t.Fatalf("options missing: %+v", updateBody)
	}
	if params, ok := opts["parameters"].([]any); !ok || len(params) != 0 {
		t.Errorf("parameters: %v", opts["parameters"])
	}
}
