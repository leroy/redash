package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// Dashboard is a Redash dashboard (summary fields).
type Dashboard struct {
	ID         int      `json:"id"`
	Slug       string   `json:"slug"`
	Name       string   `json:"name"`
	Tags       []string `json:"tags,omitempty"`
	IsArchived bool     `json:"is_archived"`
	IsDraft    bool     `json:"is_draft"`
	UpdatedAt  string   `json:"updated_at,omitempty"`
	CreatedAt  string   `json:"created_at,omitempty"`
	User       *User    `json:"user,omitempty"`

	Raw json.RawMessage `json:"-"`
}

// DashboardList is a paginated list of dashboards.
type DashboardList struct {
	Count    int         `json:"count"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
	Results  []Dashboard `json:"results"`
}

// ListDashboardsParams controls pagination/search for ListDashboards.
type ListDashboardsParams struct {
	Page     int
	PageSize int
	Search   string
}

// ListDashboards returns a page of dashboards.
func (c *Client) ListDashboards(ctx context.Context, p ListDashboardsParams) (*DashboardList, error) {
	q := url.Values{}
	if p.Page > 0 {
		q.Set("page", strconv.Itoa(p.Page))
	}
	if p.PageSize > 0 {
		q.Set("page_size", strconv.Itoa(p.PageSize))
	}
	if p.Search != "" {
		q.Set("q", p.Search)
	}
	var out DashboardList
	if err := c.Do(ctx, "GET", "/api/dashboards", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetDashboard returns a single dashboard by slug. Older Redash releases
// use slug-based lookups; newer ones also accept numeric IDs at the same
// endpoint.
func (c *Client) GetDashboard(ctx context.Context, slug string) (*Dashboard, error) {
	var raw json.RawMessage
	if err := c.Do(ctx, "GET", fmt.Sprintf("/api/dashboards/%s", url.PathEscape(slug)), nil, nil, &raw); err != nil {
		return nil, err
	}
	var out Dashboard
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode dashboard: %w", err)
	}
	out.Raw = raw
	return &out, nil
}

// CreateDashboardInput is the payload for CreateDashboard.
type CreateDashboardInput struct {
	Name string `json:"name"`
}

// CreateDashboard creates a new (empty, draft) dashboard.
func (c *Client) CreateDashboard(ctx context.Context, in CreateDashboardInput) (*Dashboard, error) {
	var raw json.RawMessage
	if err := c.Do(ctx, "POST", "/api/dashboards", nil, in, &raw); err != nil {
		return nil, err
	}
	var out Dashboard
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode dashboard: %w", err)
	}
	out.Raw = raw
	return &out, nil
}

// UpdateDashboardInput is a partial update. Only non-nil / non-empty
// fields are sent.
type UpdateDashboardInput struct {
	Name                    *string   `json:"name,omitempty"`
	Tags                    *[]string `json:"tags,omitempty"`
	IsDraft                 *bool     `json:"is_draft,omitempty"`
	DashboardFiltersEnabled *bool     `json:"dashboard_filters_enabled,omitempty"`
}

// UpdateDashboard updates a dashboard's metadata (name, tags, draft state,
// filters). Widgets are managed via the widgets endpoints.
func (c *Client) UpdateDashboard(ctx context.Context, id int, in UpdateDashboardInput) (*Dashboard, error) {
	var raw json.RawMessage
	if err := c.Do(ctx, "POST", fmt.Sprintf("/api/dashboards/%d", id), nil, in, &raw); err != nil {
		return nil, err
	}
	var out Dashboard
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode dashboard: %w", err)
	}
	out.Raw = raw
	return &out, nil
}

// ArchiveDashboard soft-deletes a dashboard.
func (c *Client) ArchiveDashboard(ctx context.Context, id int) error {
	return c.Do(ctx, "DELETE", fmt.Sprintf("/api/dashboards/%d", id), nil, nil, nil)
}
