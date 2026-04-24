package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestCreateVisualizationSendsOptions(t *testing.T) {
	var gotBody map[string]any
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/visualizations" {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"id":101,"query_id":42,"type":"CHART","name":"Chart"}`))
	})
	v, err := c.CreateVisualization(context.Background(), CreateVisualizationInput{
		QueryID: 42, Type: "CHART", Name: "Chart",
		Options: json.RawMessage(`{"series":{"stacking":"normal"}}`),
	})
	if err != nil {
		t.Fatalf("CreateVisualization: %v", err)
	}
	if v.ID != 101 || v.Type != "CHART" {
		t.Errorf("unexpected viz: %+v", v)
	}
	if gotBody["query_id"].(float64) != 42 {
		t.Errorf("query_id: %v", gotBody["query_id"])
	}
	if gotBody["type"] != "CHART" {
		t.Errorf("type: %v", gotBody["type"])
	}
	opts, ok := gotBody["options"].(map[string]any)
	if !ok {
		t.Fatalf("options not a map: %v", gotBody["options"])
	}
	if opts["series"] == nil {
		t.Errorf("options.series missing: %+v", opts)
	}
}

func TestCreateVisualizationDefaultsEmptyOptions(t *testing.T) {
	var gotBody map[string]any
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"id":1,"type":"TABLE","name":"T"}`))
	})
	_, err := c.CreateVisualization(context.Background(), CreateVisualizationInput{
		QueryID: 1, Type: "TABLE", Name: "T",
	})
	if err != nil {
		t.Fatalf("CreateVisualization: %v", err)
	}
	if got, ok := gotBody["options"].(map[string]any); !ok || len(got) != 0 {
		t.Errorf("expected empty options object, got %v", gotBody["options"])
	}
}

func TestUpdateVisualizationPartial(t *testing.T) {
	var gotBody map[string]any
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/visualizations/5" {
			t.Errorf("path: %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"id":5,"type":"CHART","name":"Renamed"}`))
	})
	name := "Renamed"
	_, err := c.UpdateVisualization(context.Background(), 5, UpdateVisualizationInput{Name: &name})
	if err != nil {
		t.Fatalf("UpdateVisualization: %v", err)
	}
	if gotBody["name"] != "Renamed" {
		t.Errorf("name: %v", gotBody["name"])
	}
	// Unset fields must NOT be sent.
	if _, ok := gotBody["type"]; ok {
		t.Errorf("type should be omitted: %v", gotBody)
	}
	if _, ok := gotBody["description"]; ok {
		t.Errorf("description should be omitted: %v", gotBody)
	}
}

func TestDeleteVisualization(t *testing.T) {
	var gotMethod, gotPath string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	})
	if err := c.DeleteVisualization(context.Background(), 42); err != nil {
		t.Fatalf("DeleteVisualization: %v", err)
	}
	if gotMethod != "DELETE" || gotPath != "/api/visualizations/42" {
		t.Errorf("got %s %s", gotMethod, gotPath)
	}
}
