package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/leroy/redash/internal/client"
	"github.com/leroy/redash/internal/output"
	"github.com/spf13/cobra"
)

func newQueriesCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "queries",
		Short: "Manage saved queries",
	}
	c.AddCommand(newQueriesListCmd())
	c.AddCommand(newQueriesGetCmd())
	c.AddCommand(newQueriesRunCmd())
	c.AddCommand(newQueriesCreateCmd())
	c.AddCommand(newQueriesUpdateCmd())
	c.AddCommand(newQueriesArchiveCmd())
	return c
}

func newQueriesListCmd() *cobra.Command {
	var (
		page     int
		pageSize int
		search   string
		tags     []string
	)
	c := &cobra.Command{
		Use:   "list",
		Short: "List saved queries",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, _, err := resolveClient()
			if err != nil {
				return err
			}
			list, err := cli.ListQueries(cmd.Context(), client.ListQueriesParams{
				Page: page, PageSize: pageSize, Search: search, Tags: tags,
			})
			if err != nil {
				return err
			}
			f, err := parseFormat()
			if err != nil {
				return err
			}
			rows := make([][]string, len(list.Results))
			for i, q := range list.Results {
				user := ""
				if q.User != nil {
					user = q.User.Email
				}
				rows[i] = []string{
					strconv.Itoa(q.ID),
					q.Name,
					strconv.Itoa(q.DataSourceID),
					strings.Join(q.Tags, ","),
					user,
					q.UpdatedAt,
				}
			}
			logf("showing %d / %d queries", len(list.Results), list.Count)
			return output.Records{
				Columns: []string{"id", "name", "data_source", "tags", "user", "updated_at"},
				Rows:    rows,
			}.Render(cmd.OutOrStdout(), f)
		},
	}
	c.Flags().IntVar(&page, "page", 1, "page number")
	c.Flags().IntVar(&pageSize, "page-size", 25, "page size")
	c.Flags().StringVarP(&search, "search", "s", "", "search term")
	c.Flags().StringArrayVar(&tags, "tag", nil, "filter by tag (repeatable)")
	return c
}

func newQueriesGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get ID",
		Short: "Get a saved query by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid query ID %q", args[0])
			}
			cli, _, err := resolveClient()
			if err != nil {
				return err
			}
			q, err := cli.GetQuery(cmd.Context(), id)
			if err != nil {
				return err
			}
			f, err := parseFormat()
			if err != nil {
				return err
			}
			if f == output.FormatJSON && len(q.Raw) > 0 {
				if _, err := cmd.OutOrStdout().Write(append(indentJSON(q.Raw), '\n')); err != nil {
					return err
				}
				return nil
			}
			return output.Object(cmd.OutOrStdout(), q, f)
		},
	}
}

func newQueriesRunCmd() *cobra.Command {
	var (
		maxAge       int
		paramPairs   []string
		paramsJSON   string
		pollInterval time.Duration
	)
	c := &cobra.Command{
		Use:   "run ID",
		Short: "Execute a saved query by ID and print the results",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid query ID %q", args[0])
			}
			params, err := parseParams(paramPairs, paramsJSON)
			if err != nil {
				return err
			}
			f, err := parseFormat()
			if err != nil {
				return err
			}
			cli, profileName, err := resolveClient()
			if err != nil {
				return err
			}
			logf("executing saved query %d against %s (profile: %s)", id, cli.BaseURL(), profileName)
			qr, err := cli.ExecuteSavedQueryAndWait(cmd.Context(), id, params, maxAge, pollInterval)
			if err != nil {
				return err
			}
			logf("fetched %d rows in %.2fs", len(qr.Data.Rows), qr.Runtime)
			return output.QueryResult(cmd.OutOrStdout(), qr, f)
		},
	}
	c.Flags().IntVar(&maxAge, "max-age", 0, "accept a cached result up to this many seconds old (0 = always run)")
	c.Flags().StringArrayVar(&paramPairs, "param", nil, "query parameter key=value (repeatable)")
	c.Flags().StringVar(&paramsJSON, "params", "", "query parameters as a JSON object")
	c.Flags().DurationVar(&pollInterval, "poll", 500*time.Millisecond, "job polling interval")
	return c
}

func newQueriesCreateCmd() *cobra.Command {
	var (
		name         string
		description  string
		dataSourceID int
		sqlFile      string
		tags         []string
	)
	c := &cobra.Command{
		Use:   "create [SQL]",
		Short: "Create a new (draft) saved query",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return errors.New("--name is required")
			}
			if dataSourceID <= 0 {
				return errors.New("--datasource is required")
			}
			sql, err := readSQL(args, sqlFile, os.Stdin)
			if err != nil {
				return err
			}
			if strings.TrimSpace(sql) == "" {
				return errors.New("no SQL provided: pass as argument, --file, or stdin")
			}
			cli, _, err := resolveClient()
			if err != nil {
				return err
			}
			q, err := cli.CreateQuery(cmd.Context(), client.CreateQueryInput{
				Name:         name,
				Description:  description,
				Query:        sql,
				DataSourceID: dataSourceID,
				Tags:         tags,
			})
			if err != nil {
				return err
			}
			f, err := parseFormat()
			if err != nil {
				return err
			}
			logf("created query %d (draft)", q.ID)
			return output.Object(cmd.OutOrStdout(), q, f)
		},
	}
	c.Flags().StringVar(&name, "name", "", "query name (required)")
	c.Flags().StringVar(&description, "description", "", "query description")
	c.Flags().IntVarP(&dataSourceID, "datasource", "d", 0, "data source ID (required)")
	c.Flags().StringVarP(&sqlFile, "file", "f", "", "read SQL from file (use - for stdin)")
	c.Flags().StringArrayVar(&tags, "tag", nil, "tag (repeatable)")
	return c
}

func newQueriesUpdateCmd() *cobra.Command {
	var (
		name         string
		description  string
		dataSourceID int
		sqlFile      string
		tags         []string
		publish      bool
		unpublish    bool
		parameters   string
	)
	c := &cobra.Command{
		Use:   "update ID [SQL]",
		Short: "Update an existing saved query",
		Long: `Update an existing saved query.

Any combination of flags may be used; only fields explicitly set are sent.
SQL may be passed as a positional argument, from --file, or from stdin.

Query parameters (the ` + "`options.parameters`" + ` array) can be replaced with
--parameters, which accepts either inline JSON, a file path, or - for
stdin. The existing options are preserved; only the parameters array is
overwritten. Example parameter:

  [{"name":"country","title":"Country","type":"text","value":"US","global":false}]`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid query ID %q", args[0])
			}
			if publish && unpublish {
				return errors.New("pass --publish or --unpublish, not both")
			}

			cli, _, err := resolveClient()
			if err != nil {
				return err
			}

			// --parameters is a merge (fetch options, splice, write back)
			// so it has to be handled as its own call.
			var q *client.Query
			if parameters != "" {
				params, err := readJSONArg(parameters)
				if err != nil {
					return fmt.Errorf("--parameters: %w", err)
				}
				q, err = cli.UpdateQueryParameters(cmd.Context(), id, params)
				if err != nil {
					return err
				}
				logf("updated parameters on query %d", q.ID)
			}

			in := client.UpdateQueryInput{}
			if cmd.Flags().Changed("name") {
				in.Name = &name
			}
			if cmd.Flags().Changed("description") {
				in.Description = &description
			}
			if cmd.Flags().Changed("datasource") {
				in.DataSourceID = &dataSourceID
			}
			if cmd.Flags().Changed("tag") {
				tagsCopy := append([]string(nil), tags...)
				in.Tags = &tagsCopy
			}
			if publish {
				f := false
				in.IsDraft = &f
			} else if unpublish {
				t := true
				in.IsDraft = &t
			}

			sqlArgs := args[1:]
			if len(sqlArgs) > 0 || sqlFile != "" {
				sql, err := readSQL(sqlArgs, sqlFile, os.Stdin)
				if err != nil {
					return err
				}
				if strings.TrimSpace(sql) != "" {
					in.Query = &sql
				}
			}

			// Skip the second call when the user only passed --parameters.
			if hasAny(in) {
				q, err = cli.UpdateQuery(cmd.Context(), id, in)
				if err != nil {
					return err
				}
				logf("updated query %d", q.ID)
			} else if q == nil {
				return errors.New("nothing to update: pass --name, --query, --parameters, --publish, etc.")
			}

			f, err := parseFormat()
			if err != nil {
				return err
			}
			return output.Object(cmd.OutOrStdout(), q, f)
		},
	}
	c.Flags().StringVar(&name, "name", "", "new query name")
	c.Flags().StringVar(&description, "description", "", "new query description")
	c.Flags().IntVarP(&dataSourceID, "datasource", "d", 0, "new data source ID")
	c.Flags().StringVarP(&sqlFile, "file", "f", "", "read SQL from file (use - for stdin)")
	c.Flags().StringArrayVar(&tags, "tag", nil, "replacement tag (repeatable; replaces existing tags)")
	c.Flags().BoolVar(&publish, "publish", false, "mark the query as published (is_draft=false)")
	c.Flags().BoolVar(&unpublish, "unpublish", false, "mark the query as draft (is_draft=true)")
	c.Flags().StringVar(&parameters, "parameters", "", "replacement parameters JSON (inline, file path, or - for stdin)")
	return c
}

// hasAny reports whether an UpdateQueryInput has any field set.
func hasAny(in client.UpdateQueryInput) bool {
	return in.Name != nil || in.Query != nil || in.DataSourceID != nil ||
		in.Description != nil || in.Tags != nil || in.IsDraft != nil ||
		len(in.Options) > 0 || len(in.Schedule) > 0
}

func newQueriesArchiveCmd() *cobra.Command {
	var yes bool
	c := &cobra.Command{
		Use:   "archive ID",
		Short: "Archive (soft-delete) a saved query",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid query ID %q", args[0])
			}
			if !yes {
				return errors.New("refusing to archive without --yes")
			}
			cli, _, err := resolveClient()
			if err != nil {
				return err
			}
			if err := cli.ArchiveQuery(cmd.Context(), id); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "archived query %d\n", id)
			return nil
		},
	}
	c.Flags().BoolVar(&yes, "yes", false, "confirm archival (required)")
	return c
}

func indentJSON(raw []byte) []byte {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return raw
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return raw
	}
	return b
}

// unused in this file but keeps io import stable when editing.
var _ io.Reader = os.Stdin
