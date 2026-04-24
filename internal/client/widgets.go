package client

import (
	"context"
	"encoding/json"
	"fmt"
)

// Widget is a dashboard widget. A widget is either a visualization pinned
// to a dashboard (VisualizationID set) or a text/markdown widget
// (Text set, VisualizationID zero).
type Widget struct {
	ID              int             `json:"id,omitempty"`
	DashboardID     int             `json:"dashboard_id,omitempty"`
	VisualizationID int             `json:"-"`
	Visualization   *Visualization  `json:"visualization,omitempty"`
	Text            string          `json:"text,omitempty"`
	Width           int             `json:"width,omitempty"`
	Options         json.RawMessage `json:"options,omitempty"`
	UpdatedAt       string          `json:"updated_at,omitempty"`
	CreatedAt       string          `json:"created_at,omitempty"`

	Raw json.RawMessage `json:"-"`
}

// Position is the grid placement inside Widget.Options.position.
type Position struct {
	Col      int `json:"col"`
	Row      int `json:"row"`
	SizeX    int `json:"sizeX"`
	SizeY    int `json:"sizeY"`
	MinSizeX int `json:"minSizeX,omitempty"`
	MaxSizeX int `json:"maxSizeX,omitempty"`
	MinSizeY int `json:"minSizeY,omitempty"`
	MaxSizeY int `json:"maxSizeY,omitempty"`
}

// DefaultWidgetOptions builds an options payload with sensible defaults
// for a widget placed at (col, row) with a (sizeX, sizeY) grid footprint.
// Callers that need finer control should build the JSON themselves.
func DefaultWidgetOptions(p Position) json.RawMessage {
	if p.SizeX == 0 {
		p.SizeX = 3
	}
	if p.SizeY == 0 {
		p.SizeY = 8
	}
	if p.MinSizeX == 0 {
		p.MinSizeX = 1
	}
	if p.MaxSizeX == 0 {
		p.MaxSizeX = 6
	}
	if p.MinSizeY == 0 {
		p.MinSizeY = 1
	}
	if p.MaxSizeY == 0 {
		p.MaxSizeY = 1000
	}
	body := map[string]any{
		"isHidden": false,
		"position": p,
	}
	out, _ := json.Marshal(body)
	return out
}

// AddWidgetInput is the payload for AddWidget. Set VisualizationID to
// pin a visualization, or leave it zero and set Text for a text widget.
// Options is the full options JSON (including the position block);
// DefaultWidgetOptions helps build a reasonable default.
type AddWidgetInput struct {
	DashboardID     int             `json:"dashboard_id"`
	VisualizationID int             `json:"visualization_id,omitempty"`
	Text            string          `json:"text,omitempty"`
	Width           int             `json:"width,omitempty"`
	Options         json.RawMessage `json:"options,omitempty"`
}

// MarshalJSON normalizes zero VisualizationID to null (required for text
// widgets) and applies a default Width of 1 when unset.
func (in AddWidgetInput) MarshalJSON() ([]byte, error) {
	m := map[string]any{
		"dashboard_id": in.DashboardID,
		"width":        1,
	}
	if in.Width > 0 {
		m["width"] = in.Width
	}
	if in.VisualizationID > 0 {
		m["visualization_id"] = in.VisualizationID
	} else {
		m["visualization_id"] = nil
		m["text"] = in.Text
	}
	if len(in.Options) > 0 {
		m["options"] = in.Options
	} else {
		m["options"] = DefaultWidgetOptions(Position{})
	}
	return json.Marshal(m)
}

// AddWidget adds a widget to a dashboard.
func (c *Client) AddWidget(ctx context.Context, in AddWidgetInput) (*Widget, error) {
	if in.VisualizationID == 0 && in.Text == "" {
		return nil, fmt.Errorf("widget must have either a visualization_id or text")
	}
	var raw json.RawMessage
	if err := c.Do(ctx, "POST", "/api/widgets", nil, in, &raw); err != nil {
		return nil, err
	}
	var out Widget
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode widget: %w", err)
	}
	out.Raw = raw
	return &out, nil
}

// UpdateWidgetInput is a partial update.
type UpdateWidgetInput struct {
	Text    *string         `json:"text,omitempty"`
	Width   *int            `json:"width,omitempty"`
	Options json.RawMessage `json:"options,omitempty"`
}

// UpdateWidget updates a widget (text, width, options/position).
func (c *Client) UpdateWidget(ctx context.Context, id int, in UpdateWidgetInput) (*Widget, error) {
	var raw json.RawMessage
	if err := c.Do(ctx, "POST", fmt.Sprintf("/api/widgets/%d", id), nil, in, &raw); err != nil {
		return nil, err
	}
	var out Widget
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode widget: %w", err)
	}
	out.Raw = raw
	return &out, nil
}

// RemoveWidget removes a widget from its dashboard.
func (c *Client) RemoveWidget(ctx context.Context, id int) error {
	return c.Do(ctx, "DELETE", fmt.Sprintf("/api/widgets/%d", id), nil, nil, nil)
}
