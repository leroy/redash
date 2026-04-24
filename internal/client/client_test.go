package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c, err := New(srv.URL, "test-key", 5*time.Second, WithUserAgent("redash-cli/test"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c, srv
}

func TestDoSendsAuthHeaderAndJSON(t *testing.T) {
	var gotAuth, gotUA, gotCT, gotAccept, gotBody string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotUA = r.Header.Get("User-Agent")
		gotCT = r.Header.Get("Content-Type")
		gotAccept = r.Header.Get("Accept")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	var out map[string]any
	if err := c.Do(context.Background(), "POST", "/api/hello", nil, map[string]any{"x": 1}, &out); err != nil {
		t.Fatalf("Do: %v", err)
	}

	if gotAuth != "Key test-key" {
		t.Errorf("Authorization: %q", gotAuth)
	}
	if gotUA != "redash-cli/test" {
		t.Errorf("User-Agent: %q", gotUA)
	}
	if gotCT != "application/json" {
		t.Errorf("Content-Type: %q", gotCT)
	}
	if gotAccept != "application/json" {
		t.Errorf("Accept: %q", gotAccept)
	}
	if !strings.Contains(gotBody, `"x":1`) {
		t.Errorf("body: %q", gotBody)
	}
	if out["ok"] != true {
		t.Errorf("out: %v", out)
	}
}

func TestDoPropagatesAPIError(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"query not found"}`))
	})
	err := c.Do(context.Background(), "GET", "/api/queries/999", nil, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.Status != 404 {
		t.Errorf("status: %d", apiErr.Status)
	}
	if apiErr.Message != "query not found" {
		t.Errorf("message: %q", apiErr.Message)
	}
}

func TestListQueriesBuildsQueryString(t *testing.T) {
	var gotPath, gotQuery string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"count":1,"page":1,"page_size":10,"results":[{"id":1,"name":"q"}]}`))
	})
	list, err := c.ListQueries(context.Background(), ListQueriesParams{
		Page: 2, PageSize: 50, Search: "active users", Tags: []string{"a", "b"},
	})
	if err != nil {
		t.Fatalf("ListQueries: %v", err)
	}
	if gotPath != "/api/queries" {
		t.Errorf("path: %q", gotPath)
	}
	if !strings.Contains(gotQuery, "page=2") || !strings.Contains(gotQuery, "page_size=50") {
		t.Errorf("query: %q", gotQuery)
	}
	if !strings.Contains(gotQuery, "q=active+users") && !strings.Contains(gotQuery, "q=active%20users") {
		t.Errorf("missing q: %q", gotQuery)
	}
	if !strings.Contains(gotQuery, "tags=a") || !strings.Contains(gotQuery, "tags=b") {
		t.Errorf("missing tags: %q", gotQuery)
	}
	if len(list.Results) != 1 || list.Results[0].Name != "q" {
		t.Errorf("results: %+v", list)
	}
}

func TestRunAdhocQueryAndWaitPollsJob(t *testing.T) {
	step := 0
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/api/query_results":
			_, _ = w.Write([]byte(`{"job":{"id":"job-1","status":1}}`))
		case r.URL.Path == "/api/jobs/job-1":
			step++
			if step < 2 {
				_, _ = w.Write([]byte(`{"job":{"id":"job-1","status":2}}`))
				return
			}
			_, _ = w.Write([]byte(`{"job":{"id":"job-1","status":3,"query_result_id":42}}`))
		case r.URL.Path == "/api/query_results/42":
			_, _ = w.Write([]byte(`{"query_result":{"id":42,"query":"SELECT 1","runtime":0.01,"data":{"columns":[{"name":"c","type":"int"}],"rows":[{"c":1}]}}}`))
		default:
			t.Errorf("unexpected call: %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected", 500)
		}
	})

	qr, err := c.RunAdhocQueryAndWait(context.Background(), 3, "SELECT 1", nil, 0, 1*time.Millisecond)
	if err != nil {
		t.Fatalf("RunAdhocQueryAndWait: %v", err)
	}
	if qr.ID != 42 {
		t.Errorf("id: %d", qr.ID)
	}
	if len(qr.Data.Rows) != 1 {
		t.Fatalf("rows: %+v", qr.Data.Rows)
	}
	if qr.Data.Rows[0]["c"] != float64(1) {
		t.Errorf("row: %+v", qr.Data.Rows[0])
	}
	if step < 2 {
		t.Errorf("expected job polled at least twice, got %d", step)
	}
}

func TestWaitForJobFailureReturnsError(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"job":{"id":"job-1","status":4,"error":"syntax error"}}`))
	})
	_, err := c.WaitForJob(context.Background(), "job-1", 1*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "syntax error") {
		t.Fatalf("expected failure error, got %v", err)
	}
}

func TestListDataSources(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/data_sources" {
			t.Errorf("path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"id":1,"name":"warehouse","type":"pg","syntax":"sql"}]`))
	})
	ds, err := c.ListDataSources(context.Background())
	if err != nil {
		t.Fatalf("ListDataSources: %v", err)
	}
	if len(ds) != 1 || ds[0].Name != "warehouse" || ds[0].Type != "pg" {
		t.Errorf("ds: %+v", ds)
	}
}

func TestCreateQueryMarshalsBody(t *testing.T) {
	var body map[string]any
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		_, _ = w.Write([]byte(`{"id":7,"name":"hello","query":"SELECT 1","data_source_id":3}`))
	})
	q, err := c.CreateQuery(context.Background(), CreateQueryInput{
		Name: "hello", Query: "SELECT 1", DataSourceID: 3, Tags: []string{"a"},
	})
	if err != nil {
		t.Fatalf("CreateQuery: %v", err)
	}
	if q.ID != 7 {
		t.Errorf("id: %d", q.ID)
	}
	if body["name"] != "hello" || body["query"] != "SELECT 1" {
		t.Errorf("body: %+v", body)
	}
	if body["data_source_id"].(float64) != 3 {
		t.Errorf("data_source_id: %v", body["data_source_id"])
	}
}
