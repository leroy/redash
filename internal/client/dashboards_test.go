package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestCreateDashboard(t *testing.T) {
	var gotBody map[string]any
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/dashboards" {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"id":7,"slug":"revenue","name":"Revenue"}`))
	})
	d, err := c.CreateDashboard(context.Background(), CreateDashboardInput{Name: "Revenue"})
	if err != nil {
		t.Fatalf("CreateDashboard: %v", err)
	}
	if d.ID != 7 || d.Slug != "revenue" {
		t.Errorf("got: %+v", d)
	}
	if gotBody["name"] != "Revenue" {
		t.Errorf("name: %v", gotBody["name"])
	}
}

func TestUpdateDashboardPartial(t *testing.T) {
	var gotBody map[string]any
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/dashboards/7" {
			t.Errorf("path: %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"id":7,"name":"Renamed","is_draft":false}`))
	})
	name := "Renamed"
	isDraft := false
	_, err := c.UpdateDashboard(context.Background(), 7, UpdateDashboardInput{
		Name: &name, IsDraft: &isDraft,
	})
	if err != nil {
		t.Fatalf("UpdateDashboard: %v", err)
	}
	if gotBody["name"] != "Renamed" {
		t.Errorf("name: %v", gotBody["name"])
	}
	if gotBody["is_draft"] != false {
		t.Errorf("is_draft: %v", gotBody["is_draft"])
	}
	// Unset fields must NOT be present.
	if _, ok := gotBody["tags"]; ok {
		t.Errorf("tags should be omitted: %v", gotBody)
	}
	if _, ok := gotBody["dashboard_filters_enabled"]; ok {
		t.Errorf("dashboard_filters_enabled should be omitted: %v", gotBody)
	}
}

func TestArchiveDashboard(t *testing.T) {
	var gotMethod, gotPath string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
	})
	if err := c.ArchiveDashboard(context.Background(), 7); err != nil {
		t.Fatalf("ArchiveDashboard: %v", err)
	}
	if gotMethod != "DELETE" || gotPath != "/api/dashboards/7" {
		t.Errorf("got %s %s", gotMethod, gotPath)
	}
}
