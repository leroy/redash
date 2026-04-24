package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestAddWidgetVisualization(t *testing.T) {
	var gotBody map[string]any
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/widgets" {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"id":1,"dashboard_id":7}`))
	})
	opts := DefaultWidgetOptions(Position{Col: 2, Row: 3, SizeX: 4, SizeY: 5})
	w, err := c.AddWidget(context.Background(), AddWidgetInput{
		DashboardID: 7, VisualizationID: 99, Options: opts,
	})
	if err != nil {
		t.Fatalf("AddWidget: %v", err)
	}
	if w.ID != 1 {
		t.Errorf("id: %d", w.ID)
	}
	if gotBody["dashboard_id"].(float64) != 7 {
		t.Errorf("dashboard_id: %v", gotBody["dashboard_id"])
	}
	if gotBody["visualization_id"].(float64) != 99 {
		t.Errorf("visualization_id: %v", gotBody["visualization_id"])
	}
	if _, ok := gotBody["text"]; ok {
		t.Errorf("text should not be present for viz widget: %v", gotBody)
	}
	// width should default to 1 when unset.
	if gotBody["width"].(float64) != 1 {
		t.Errorf("width default: %v", gotBody["width"])
	}
	pos, _ := gotBody["options"].(map[string]any)["position"].(map[string]any)
	if pos["col"].(float64) != 2 || pos["sizeX"].(float64) != 4 {
		t.Errorf("position not propagated: %v", pos)
	}
}

func TestAddWidgetText(t *testing.T) {
	var gotBody map[string]any
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"id":2,"dashboard_id":7,"text":"hi"}`))
	})
	_, err := c.AddWidget(context.Background(), AddWidgetInput{
		DashboardID: 7, Text: "hi",
	})
	if err != nil {
		t.Fatalf("AddWidget: %v", err)
	}
	if gotBody["text"] != "hi" {
		t.Errorf("text: %v", gotBody["text"])
	}
	// Text widgets must send visualization_id as null.
	if gotBody["visualization_id"] != nil {
		t.Errorf("visualization_id should be null for text widget, got %v", gotBody["visualization_id"])
	}
}

func TestAddWidgetRejectsMissingBody(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not hit the server")
	})
	_, err := c.AddWidget(context.Background(), AddWidgetInput{DashboardID: 7})
	if err == nil {
		t.Fatal("expected error (no viz, no text)")
	}
}

func TestUpdateWidget(t *testing.T) {
	var gotBody map[string]any
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/widgets/1" {
			t.Errorf("path: %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"id":1,"text":"new"}`))
	})
	text := "new"
	_, err := c.UpdateWidget(context.Background(), 1, UpdateWidgetInput{Text: &text})
	if err != nil {
		t.Fatalf("UpdateWidget: %v", err)
	}
	if gotBody["text"] != "new" {
		t.Errorf("text: %v", gotBody["text"])
	}
	if _, ok := gotBody["width"]; ok {
		t.Errorf("width should be omitted: %v", gotBody)
	}
}

func TestRemoveWidget(t *testing.T) {
	var gotMethod, gotPath string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
	})
	if err := c.RemoveWidget(context.Background(), 99); err != nil {
		t.Fatalf("RemoveWidget: %v", err)
	}
	if gotMethod != "DELETE" || gotPath != "/api/widgets/99" {
		t.Errorf("got %s %s", gotMethod, gotPath)
	}
}

func TestDefaultWidgetOptionsSetsReasonableDefaults(t *testing.T) {
	raw := DefaultWidgetOptions(Position{})
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	pos, ok := decoded["position"].(map[string]any)
	if !ok {
		t.Fatalf("no position: %v", decoded)
	}
	if pos["sizeX"].(float64) != 3 {
		t.Errorf("sizeX default: %v", pos["sizeX"])
	}
	if pos["sizeY"].(float64) != 8 {
		t.Errorf("sizeY default: %v", pos["sizeY"])
	}
}
