package client

import (
	"context"
	"encoding/json"
	"fmt"
)

// Visualization is a Redash visualization attached to a query. Each query
// gets an implicit TABLE visualization automatically; additional ones
// (charts, counters, pivots, ...) are created explicitly.
type Visualization struct {
	ID          int             `json:"id,omitempty"`
	QueryID     int             `json:"query_id,omitempty"`
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Options     json.RawMessage `json:"options,omitempty"`
	UpdatedAt   string          `json:"updated_at,omitempty"`
	CreatedAt   string          `json:"created_at,omitempty"`

	Raw json.RawMessage `json:"-"`
}

// CreateVisualizationInput is the payload for CreateVisualization.
//
// Common Type values: TABLE, CHART, COUNTER, PIVOT_TABLE, MAP, WORD_CLOUD,
// SUNBURST_SEQUENCE, SANKEY, BOXPLOT, CHOROPLETH, DETAILS. Options are
// type-specific JSON — refer to the Redash UI's "Edit Visualization"
// payload for the shape.
type CreateVisualizationInput struct {
	QueryID     int             `json:"query_id"`
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Options     json.RawMessage `json:"options,omitempty"`
}

// CreateVisualization creates a new visualization on a query.
func (c *Client) CreateVisualization(ctx context.Context, in CreateVisualizationInput) (*Visualization, error) {
	if in.Options == nil {
		in.Options = json.RawMessage(`{}`)
	}
	var raw json.RawMessage
	if err := c.Do(ctx, "POST", "/api/visualizations", nil, in, &raw); err != nil {
		return nil, err
	}
	var out Visualization
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode visualization: %w", err)
	}
	out.Raw = raw
	return &out, nil
}

// UpdateVisualizationInput is a partial update. All fields are optional;
// non-nil / non-empty fields are sent.
type UpdateVisualizationInput struct {
	Type        *string         `json:"type,omitempty"`
	Name        *string         `json:"name,omitempty"`
	Description *string         `json:"description,omitempty"`
	Options     json.RawMessage `json:"options,omitempty"`
}

// UpdateVisualization updates an existing visualization.
func (c *Client) UpdateVisualization(ctx context.Context, id int, in UpdateVisualizationInput) (*Visualization, error) {
	var raw json.RawMessage
	if err := c.Do(ctx, "POST", fmt.Sprintf("/api/visualizations/%d", id), nil, in, &raw); err != nil {
		return nil, err
	}
	var out Visualization
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode visualization: %w", err)
	}
	out.Raw = raw
	return &out, nil
}

// DeleteVisualization removes a visualization. The implicit TABLE
// visualization of a query cannot be deleted.
func (c *Client) DeleteVisualization(ctx context.Context, id int) error {
	return c.Do(ctx, "DELETE", fmt.Sprintf("/api/visualizations/%d", id), nil, nil, nil)
}
