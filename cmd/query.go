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

	"github.com/leroy/redash/internal/output"
	"github.com/spf13/cobra"
)

func newQueryCmd() *cobra.Command {
	var (
		sqlFile      string
		dataSourceID int
		maxAge       int
		paramPairs   []string
		paramsJSON   string
		pollInterval time.Duration
	)

	c := &cobra.Command{
		Use:   "query [SQL]",
		Short: "Run an ad-hoc SQL query against a data source",
		Long: `Run an ad-hoc SQL query against a data source and print the results.

The SQL may be passed as a positional argument, from --file, or from stdin.
Parameters may be passed as repeated --param key=value flags, or as a single
--params JSON object.

Example:

  redash query --datasource 3 "SELECT count(*) FROM users"
  cat report.sql | redash query --datasource 3 -o csv > out.csv
  redash query -d 3 -f report.sql --param country=US --param limit=100`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if dataSourceID <= 0 {
				return errors.New("--datasource (-d) is required")
			}
			sql, err := readSQL(args, sqlFile, os.Stdin)
			if err != nil {
				return err
			}
			if strings.TrimSpace(sql) == "" {
				return errors.New("no SQL provided: pass as argument, --file, or stdin")
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
			logf("running query against %s (profile: %s, datasource: %d)", cli.BaseURL(), profileName, dataSourceID)

			qr, err := cli.RunAdhocQueryAndWait(cmd.Context(), dataSourceID, sql, params, maxAge, pollInterval)
			if err != nil {
				return err
			}
			logf("fetched %d rows in %.2fs", len(qr.Data.Rows), qr.Runtime)
			return output.QueryResult(cmd.OutOrStdout(), qr, f)
		},
	}
	c.Flags().StringVarP(&sqlFile, "file", "f", "", "read SQL from file (use - for stdin)")
	c.Flags().IntVarP(&dataSourceID, "datasource", "d", 0, "data source ID to run the query against (required)")
	c.Flags().IntVar(&maxAge, "max-age", 0, "accept a cached result up to this many seconds old (0 = always run)")
	c.Flags().StringArrayVar(&paramPairs, "param", nil, "query parameter key=value (repeatable)")
	c.Flags().StringVar(&paramsJSON, "params", "", "query parameters as a JSON object")
	c.Flags().DurationVar(&pollInterval, "poll", 500*time.Millisecond, "job polling interval")
	_ = c.MarkFlagRequired("datasource")
	return c
}

// readSQL chooses the SQL source: positional arg, --file (or stdin via
// --file -), or stdin (only if stdin is piped and no arg/file given).
func readSQL(args []string, file string, stdin io.Reader) (string, error) {
	switch {
	case len(args) == 1 && file != "":
		return "", errors.New("pass SQL as an argument or --file, not both")
	case len(args) == 1:
		return args[0], nil
	case file == "-":
		b, err := io.ReadAll(stdin)
		if err != nil {
			return "", fmt.Errorf("read stdin: %w", err)
		}
		return string(b), nil
	case file != "":
		b, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", file, err)
		}
		return string(b), nil
	default:
		// Try stdin if it's piped.
		fi, err := os.Stdin.Stat()
		if err == nil && (fi.Mode()&os.ModeCharDevice) == 0 {
			b, err := io.ReadAll(stdin)
			if err != nil {
				return "", fmt.Errorf("read stdin: %w", err)
			}
			return string(b), nil
		}
		return "", nil
	}
}

// parseParams merges --param key=value pairs and --params JSON into a
// single map. Values from --param are parsed as JSON when possible
// (so `--param limit=100` becomes int, `--param active=true` becomes bool),
// otherwise kept as string. --params JSON takes precedence for conflicts.
func parseParams(pairs []string, jsonStr string) (map[string]any, error) {
	out := map[string]any{}
	for _, p := range pairs {
		k, v, ok := strings.Cut(p, "=")
		if !ok || k == "" {
			return nil, fmt.Errorf("--param must be key=value, got %q", p)
		}
		out[k] = coerceParam(v)
	}
	if jsonStr != "" {
		var m map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
			return nil, fmt.Errorf("--params must be a JSON object: %w", err)
		}
		for k, v := range m {
			out[k] = v
		}
	}
	return out, nil
}

func coerceParam(v string) any {
	if v == "" {
		return ""
	}
	if v == "true" {
		return true
	}
	if v == "false" {
		return false
	}
	if i, err := strconv.ParseInt(v, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		return f
	}
	return v
}
