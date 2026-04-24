package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// User is a Redash user.
type User struct {
	ID               int      `json:"id"`
	Name             string   `json:"name"`
	Email            string   `json:"email"`
	Groups           []int    `json:"groups,omitempty"`
	IsDisabled       bool     `json:"is_disabled,omitempty"`
	IsInvitationPending bool  `json:"is_invitation_pending,omitempty"`
	ProfileImageURL  string   `json:"profile_image_url,omitempty"`
	CreatedAt        string   `json:"created_at,omitempty"`
	UpdatedAt        string   `json:"updated_at,omitempty"`
	ActiveAt         string   `json:"active_at,omitempty"`

	Raw json.RawMessage `json:"-"`
}

// UserList is a paginated list of users.
type UserList struct {
	Count    int    `json:"count"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Results  []User `json:"results"`
}

// ListUsersParams controls pagination/search for ListUsers.
type ListUsersParams struct {
	Page     int
	PageSize int
	Search   string
	Disabled bool
}

// ListUsers returns a page of users.
func (c *Client) ListUsers(ctx context.Context, p ListUsersParams) (*UserList, error) {
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
	if p.Disabled {
		q.Set("disabled", "true")
	}
	var out UserList
	if err := c.Do(ctx, "GET", "/api/users", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetUser returns a single user.
func (c *Client) GetUser(ctx context.Context, id int) (*User, error) {
	var raw json.RawMessage
	if err := c.Do(ctx, "GET", fmt.Sprintf("/api/users/%d", id), nil, nil, &raw); err != nil {
		return nil, err
	}
	var out User
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode user: %w", err)
	}
	out.Raw = raw
	return &out, nil
}

// CreateUserInput is the payload for CreateUser.
type CreateUserInput struct {
	Name   string `json:"name"`
	Email  string `json:"email"`
	Groups []int  `json:"groups,omitempty"`
}

// CreateUser creates a new user (sends an invitation email).
func (c *Client) CreateUser(ctx context.Context, in CreateUserInput) (*User, error) {
	var raw json.RawMessage
	if err := c.Do(ctx, "POST", "/api/users", nil, in, &raw); err != nil {
		return nil, err
	}
	var out User
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode user: %w", err)
	}
	out.Raw = raw
	return &out, nil
}

// DisableUser disables a user account.
func (c *Client) DisableUser(ctx context.Context, id int) error {
	return c.Do(ctx, "POST", fmt.Sprintf("/api/users/%d/disable", id), nil, nil, nil)
}

// EnableUser re-enables a previously disabled user account.
func (c *Client) EnableUser(ctx context.Context, id int) error {
	return c.Do(ctx, "DELETE", fmt.Sprintf("/api/users/%d/disable", id), nil, nil, nil)
}
