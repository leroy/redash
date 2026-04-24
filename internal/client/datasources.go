package client

import (
	"context"
	"fmt"
)

// DataSource is a Redash data source.
type DataSource struct {
	ID       int            `json:"id"`
	Name     string         `json:"name"`
	Type     string         `json:"type"`
	Syntax   string         `json:"syntax,omitempty"`
	Paused   int            `json:"paused,omitempty"`
	ViewOnly bool           `json:"view_only,omitempty"`
	Options  map[string]any `json:"options,omitempty"`
}

// ListDataSources returns every data source visible to the current API key.
func (c *Client) ListDataSources(ctx context.Context) ([]DataSource, error) {
	var out []DataSource
	if err := c.Do(ctx, "GET", "/api/data_sources", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetDataSource returns a single data source by ID.
func (c *Client) GetDataSource(ctx context.Context, id int) (*DataSource, error) {
	var out DataSource
	if err := c.Do(ctx, "GET", fmt.Sprintf("/api/data_sources/%d", id), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SchemaColumn describes a single column on a table in a data source schema.
type SchemaColumn struct {
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
}

// SchemaTable describes a single table in a data source schema.
type SchemaTable struct {
	Name    string         `json:"name"`
	Columns []SchemaColumn `json:"columns"`
}

// Schema is returned by GetSchema. The Redash response is either:
//
//	{"schema": [ {"name": "...", "columns": [...]}]}
//
// or (older clusters, or when refresh=true) it may return a job envelope.
// We handle the simple case; the CLI surfaces the job case as an error
// asking the user to retry.
type Schema struct {
	Tables []SchemaTable `json:"schema"`
}

// GetSchema returns the cached schema for a data source. If refresh is
// true, a refresh is requested; on some Redash versions this returns a
// job rather than a schema, in which case the caller should retry after
// a short delay.
func (c *Client) GetSchema(ctx context.Context, dataSourceID int, refresh bool) (*Schema, error) {
	path := fmt.Sprintf("/api/data_sources/%d/schema", dataSourceID)
	if refresh {
		path += "?refresh=true"
	}
	var out Schema
	if err := c.Do(ctx, "GET", path, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
