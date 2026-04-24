package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// Query is a Redash saved query. Fields are a subset of the API payload; the
// full payload is kept in Raw for "get" commands.
type Query struct {
	ID                int             `json:"id"`
	Name              string          `json:"name"`
	Description       string          `json:"description,omitempty"`
	Query             string          `json:"query"`
	DataSourceID      int             `json:"data_source_id"`
	Schedule          json.RawMessage `json:"schedule,omitempty"`
	Options           json.RawMessage `json:"options,omitempty"`
	Tags              []string        `json:"tags,omitempty"`
	IsArchived        bool            `json:"is_archived"`
	IsDraft           bool            `json:"is_draft"`
	UpdatedAt         string          `json:"updated_at,omitempty"`
	CreatedAt         string          `json:"created_at,omitempty"`
	User              *User           `json:"user,omitempty"`
	LatestQueryDataID *int            `json:"latest_query_data_id,omitempty"`

	Raw json.RawMessage `json:"-"`
}

// QueryList is a paginated list of saved queries.
type QueryList struct {
	Count    int     `json:"count"`
	Page     int     `json:"page"`
	PageSize int     `json:"page_size"`
	Results  []Query `json:"results"`
}

// ListQueriesParams controls pagination and search for ListQueries.
type ListQueriesParams struct {
	Page     int
	PageSize int
	Search   string
	Tags     []string
}

// ListQueries returns a page of saved queries.
func (c *Client) ListQueries(ctx context.Context, p ListQueriesParams) (*QueryList, error) {
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
	for _, t := range p.Tags {
		q.Add("tags", t)
	}
	var out QueryList
	if err := c.Do(ctx, "GET", "/api/queries", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetQuery returns a single saved query by ID.
func (c *Client) GetQuery(ctx context.Context, id int) (*Query, error) {
	var raw json.RawMessage
	if err := c.Do(ctx, "GET", fmt.Sprintf("/api/queries/%d", id), nil, nil, &raw); err != nil {
		return nil, err
	}
	var out Query
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode query: %w", err)
	}
	out.Raw = raw
	return &out, nil
}

// CreateQueryInput is the payload for CreateQuery. Schedule/Options/Tags are
// optional and may be left nil/empty.
type CreateQueryInput struct {
	Name         string          `json:"name"`
	Query        string          `json:"query"`
	DataSourceID int             `json:"data_source_id"`
	Description  string          `json:"description,omitempty"`
	Schedule     json.RawMessage `json:"schedule,omitempty"`
	Options      json.RawMessage `json:"options,omitempty"`
	Tags         []string        `json:"tags,omitempty"`
}

// CreateQuery creates a new draft query.
func (c *Client) CreateQuery(ctx context.Context, in CreateQueryInput) (*Query, error) {
	var raw json.RawMessage
	if err := c.Do(ctx, "POST", "/api/queries", nil, in, &raw); err != nil {
		return nil, err
	}
	var out Query
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode query: %w", err)
	}
	out.Raw = raw
	return &out, nil
}

// UpdateQueryInput is the payload for UpdateQuery. All fields are optional;
// only non-nil / non-empty fields are sent.
type UpdateQueryInput struct {
	Name         *string         `json:"name,omitempty"`
	Query        *string         `json:"query,omitempty"`
	DataSourceID *int            `json:"data_source_id,omitempty"`
	Description  *string         `json:"description,omitempty"`
	Schedule     json.RawMessage `json:"schedule,omitempty"`
	Options      json.RawMessage `json:"options,omitempty"`
	Tags         *[]string       `json:"tags,omitempty"`
	IsDraft      *bool           `json:"is_draft,omitempty"`
}

// UpdateQuery updates a saved query.
func (c *Client) UpdateQuery(ctx context.Context, id int, in UpdateQueryInput) (*Query, error) {
	var raw json.RawMessage
	if err := c.Do(ctx, "POST", fmt.Sprintf("/api/queries/%d", id), nil, in, &raw); err != nil {
		return nil, err
	}
	var out Query
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode query: %w", err)
	}
	out.Raw = raw
	return &out, nil
}

// ArchiveQuery soft-deletes a saved query.
func (c *Client) ArchiveQuery(ctx context.Context, id int) error {
	return c.Do(ctx, "DELETE", fmt.Sprintf("/api/queries/%d", id), nil, nil, nil)
}

// --- Execution ---------------------------------------------------------

// Job represents an async execution job. Status: 1 pending, 2 started,
// 3 success, 4 failure, 5 cancelled.
type Job struct {
	ID            string `json:"id"`
	Status        int    `json:"status"`
	Error         string `json:"error,omitempty"`
	QueryResultID int    `json:"query_result_id,omitempty"`
	UpdatedAt     any    `json:"updated_at,omitempty"`
}

type jobEnvelope struct {
	Job Job `json:"job"`
}

// StatusText returns a human label for j.Status.
func (j Job) StatusText() string {
	switch j.Status {
	case 1:
		return "pending"
	case 2:
		return "started"
	case 3:
		return "success"
	case 4:
		return "failure"
	case 5:
		return "cancelled"
	default:
		return fmt.Sprintf("unknown(%d)", j.Status)
	}
}

// QueryResult is the payload of /api/query_results/{id}.
type QueryResult struct {
	ID           int             `json:"id"`
	QueryHash    string          `json:"query_hash"`
	Query        string          `json:"query"`
	DataSourceID int             `json:"data_source_id"`
	Runtime      float64         `json:"runtime"`
	RetrievedAt  string          `json:"retrieved_at"`
	Data         QueryResultData `json:"data"`
}

// QueryResultData is the tabular payload inside a QueryResult.
type QueryResultData struct {
	Columns []QueryResultColumn `json:"columns"`
	Rows    []map[string]any    `json:"rows"`
}

// QueryResultColumn describes a single column.
type QueryResultColumn struct {
	Name         string `json:"name"`
	FriendlyName string `json:"friendly_name"`
	Type         string `json:"type"`
}

type queryResultEnvelope struct {
	QueryResult QueryResult `json:"query_result"`
	Job         *Job        `json:"job,omitempty"`
}

// RunAdhocQuery submits an ad-hoc SQL query against a data source. It
// returns either a completed QueryResult (if Redash had a cached result)
// or a Job to poll. Callers usually prefer RunAdhocQueryAndWait.
func (c *Client) RunAdhocQuery(ctx context.Context, dataSourceID int, query string, params map[string]any, maxAge int) (*queryResultEnvelope, error) {
	body := map[string]any{
		"data_source_id": dataSourceID,
		"query":          query,
		"max_age":        maxAge,
	}
	if len(params) > 0 {
		body["parameters"] = params
	}
	var env queryResultEnvelope
	if err := c.Do(ctx, "POST", "/api/query_results", nil, body, &env); err != nil {
		return nil, err
	}
	return &env, nil
}

// ExecuteSavedQuery triggers execution of a saved query. It returns a Job
// to poll. Callers usually prefer ExecuteSavedQueryAndWait.
func (c *Client) ExecuteSavedQuery(ctx context.Context, queryID int, params map[string]any, maxAge int) (*Job, error) {
	body := map[string]any{
		"max_age": maxAge,
	}
	if len(params) > 0 {
		body["parameters"] = params
	}
	var env jobEnvelope
	if err := c.Do(ctx, "POST", fmt.Sprintf("/api/queries/%d/results", queryID), nil, body, &env); err != nil {
		return nil, err
	}
	return &env.Job, nil
}

// GetJob returns the current status of a job.
func (c *Client) GetJob(ctx context.Context, id string) (*Job, error) {
	var env jobEnvelope
	if err := c.Do(ctx, "GET", "/api/jobs/"+id, nil, nil, &env); err != nil {
		return nil, err
	}
	return &env.Job, nil
}

// GetQueryResult fetches a completed result by its ID.
func (c *Client) GetQueryResult(ctx context.Context, id int) (*QueryResult, error) {
	var env queryResultEnvelope
	if err := c.Do(ctx, "GET", fmt.Sprintf("/api/query_results/%d", id), nil, nil, &env); err != nil {
		return nil, err
	}
	return &env.QueryResult, nil
}

// WaitForJob polls the job endpoint until the job is done (success,
// failure, or cancelled) or ctx is cancelled.
func (c *Client) WaitForJob(ctx context.Context, id string, poll time.Duration) (*Job, error) {
	if poll <= 0 {
		poll = 500 * time.Millisecond
	}
	t := time.NewTicker(poll)
	defer t.Stop()
	for {
		j, err := c.GetJob(ctx, id)
		if err != nil {
			return nil, err
		}
		switch j.Status {
		case 3: // success
			return j, nil
		case 4: // failure
			return j, fmt.Errorf("redash job failed: %s", j.Error)
		case 5: // cancelled
			return j, fmt.Errorf("redash job cancelled")
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-t.C:
		}
	}
}

// RunAdhocQueryAndWait submits an ad-hoc query and waits for the result.
func (c *Client) RunAdhocQueryAndWait(ctx context.Context, dataSourceID int, query string, params map[string]any, maxAge int, poll time.Duration) (*QueryResult, error) {
	env, err := c.RunAdhocQuery(ctx, dataSourceID, query, params, maxAge)
	if err != nil {
		return nil, err
	}
	// Redash returns either a cached result or a job.
	if env.QueryResult.ID != 0 {
		return &env.QueryResult, nil
	}
	if env.Job == nil || env.Job.ID == "" {
		return nil, fmt.Errorf("redash returned neither a result nor a job")
	}
	j, err := c.WaitForJob(ctx, env.Job.ID, poll)
	if err != nil {
		return nil, err
	}
	if j.QueryResultID == 0 {
		return nil, fmt.Errorf("job succeeded but no query_result_id")
	}
	return c.GetQueryResult(ctx, j.QueryResultID)
}

// ExecuteSavedQueryAndWait executes a saved query and waits for the result.
func (c *Client) ExecuteSavedQueryAndWait(ctx context.Context, queryID int, params map[string]any, maxAge int, poll time.Duration) (*QueryResult, error) {
	job, err := c.ExecuteSavedQuery(ctx, queryID, params, maxAge)
	if err != nil {
		return nil, err
	}
	// Some Redash versions return a completed result directly (status 3).
	if job.Status == 3 && job.QueryResultID != 0 {
		return c.GetQueryResult(ctx, job.QueryResultID)
	}
	if job.ID == "" {
		return nil, fmt.Errorf("redash returned an empty job")
	}
	j, err := c.WaitForJob(ctx, job.ID, poll)
	if err != nil {
		return nil, err
	}
	if j.QueryResultID == 0 {
		return nil, fmt.Errorf("job succeeded but no query_result_id")
	}
	return c.GetQueryResult(ctx, j.QueryResultID)
}
