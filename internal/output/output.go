// Package output renders query results and generic record lists as
// pretty tables, JSON, or CSV.
package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/leroy/redash/internal/client"
	"github.com/olekukonko/tablewriter"
)

// Format is the output format selected by the user.
type Format string

const (
	// FormatTable renders a pretty terminal table (default).
	FormatTable Format = "table"
	// FormatJSON renders compact JSON on a single line per row, or the
	// raw response when rendering a result object.
	FormatJSON Format = "json"
	// FormatCSV renders RFC 4180 CSV.
	FormatCSV Format = "csv"
)

// ParseFormat returns the matching Format for a string, or an error for
// unknown values.
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(s) {
	case "", "table":
		return FormatTable, nil
	case "json":
		return FormatJSON, nil
	case "csv":
		return FormatCSV, nil
	default:
		return "", fmt.Errorf("unknown format %q (want: table|json|csv)", s)
	}
}

// Records is a generic rendering helper for lists of records that don't
// come from the query_result endpoint (e.g. list-queries, list-datasources).
type Records struct {
	Columns []string
	Rows    [][]string
}

// Render writes the records in the requested format to w.
func (r Records) Render(w io.Writer, f Format) error {
	switch f {
	case FormatTable:
		return renderTable(w, r.Columns, r.Rows)
	case FormatCSV:
		return renderCSV(w, r.Columns, r.Rows)
	case FormatJSON:
		return renderRecordsJSON(w, r.Columns, r.Rows)
	default:
		return fmt.Errorf("unsupported format %q", f)
	}
}

// QueryResult renders a Redash QueryResult to w.
func QueryResult(w io.Writer, qr *client.QueryResult, f Format) error {
	if qr == nil {
		return fmt.Errorf("nil query result")
	}
	cols := make([]string, len(qr.Data.Columns))
	for i, c := range qr.Data.Columns {
		cols[i] = c.Name
	}
	rows := make([][]string, len(qr.Data.Rows))
	for i, r := range qr.Data.Rows {
		row := make([]string, len(cols))
		for j, c := range cols {
			row[j] = stringify(r[c])
		}
		rows[i] = row
	}

	switch f {
	case FormatTable:
		return renderTable(w, cols, rows)
	case FormatCSV:
		return renderCSV(w, cols, rows)
	case FormatJSON:
		// For JSON we preserve original types rather than stringifying.
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		return enc.Encode(qr.Data.Rows)
	default:
		return fmt.Errorf("unsupported format %q", f)
	}
}

// Object pretty-prints a single object (used for `get` commands in JSON
// mode). For table/CSV it's rendered as two columns: field, value.
func Object(w io.Writer, obj any, f Format) error {
	switch f {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		return enc.Encode(obj)
	case FormatTable, FormatCSV:
		data, err := json.Marshal(obj)
		if err != nil {
			return err
		}
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			// Fall back to JSON if the object isn't a map.
			enc := json.NewEncoder(w)
			enc.SetIndent("", "  ")
			return enc.Encode(obj)
		}
		rows := make([][]string, 0, len(m))
		for k, v := range m {
			rows = append(rows, []string{k, stringify(v)})
		}
		// Stable-ish ordering.
		sortRowsByFirst(rows)
		cols := []string{"field", "value"}
		if f == FormatCSV {
			return renderCSV(w, cols, rows)
		}
		return renderTable(w, cols, rows)
	default:
		return fmt.Errorf("unsupported format %q", f)
	}
}

// --- internals --------------------------------------------------------

func renderTable(w io.Writer, cols []string, rows [][]string) error {
	tw := tablewriter.NewWriter(w)
	tw.Header(cols)
	for _, r := range rows {
		if err := tw.Append(r); err != nil {
			return err
		}
	}
	return tw.Render()
}

func renderCSV(w io.Writer, cols []string, rows [][]string) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(cols); err != nil {
		return err
	}
	for _, r := range rows {
		if err := cw.Write(r); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func renderRecordsJSON(w io.Writer, cols []string, rows [][]string) error {
	out := make([]map[string]string, len(rows))
	for i, r := range rows {
		m := make(map[string]string, len(cols))
		for j, c := range cols {
			if j < len(r) {
				m[c] = r[j]
			}
		}
		out[i] = m
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(out)
}

func stringify(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	case float64:
		// Drop trailing .0 on integers.
		if x == float64(int64(x)) {
			return fmt.Sprintf("%d", int64(x))
		}
		return fmt.Sprintf("%v", x)
	default:
		// Maps, slices: render as JSON.
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	}
}

func sortRowsByFirst(rows [][]string) {
	// Small lists; insertion sort keeps it trivial without importing sort.
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0 && rows[j][0] < rows[j-1][0]; j-- {
			rows[j], rows[j-1] = rows[j-1], rows[j]
		}
	}
}
