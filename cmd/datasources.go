package cmd

import (
	"fmt"
	"strconv"

	"github.com/leroy/redash/internal/output"
	"github.com/spf13/cobra"
)

func newDataSourcesCmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "datasources",
		Aliases: []string{"ds", "datasource"},
		Short:   "Inspect data sources",
	}
	c.AddCommand(newDataSourcesListCmd())
	c.AddCommand(newDataSourcesGetCmd())
	c.AddCommand(newDataSourcesSchemaCmd())
	return c
}

func newDataSourcesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List data sources",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, _, err := resolveClient()
			if err != nil {
				return err
			}
			dss, err := cli.ListDataSources(cmd.Context())
			if err != nil {
				return err
			}
			f, err := parseFormat()
			if err != nil {
				return err
			}
			rows := make([][]string, len(dss))
			for i, d := range dss {
				rows[i] = []string{
					strconv.Itoa(d.ID),
					d.Name,
					d.Type,
					d.Syntax,
					strconv.FormatBool(d.ViewOnly),
				}
			}
			return output.Records{
				Columns: []string{"id", "name", "type", "syntax", "view_only"},
				Rows:    rows,
			}.Render(cmd.OutOrStdout(), f)
		},
	}
}

func newDataSourcesGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get ID",
		Short: "Get a single data source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid data source ID %q", args[0])
			}
			cli, _, err := resolveClient()
			if err != nil {
				return err
			}
			ds, err := cli.GetDataSource(cmd.Context(), id)
			if err != nil {
				return err
			}
			f, err := parseFormat()
			if err != nil {
				return err
			}
			return output.Object(cmd.OutOrStdout(), ds, f)
		},
	}
}

func newDataSourcesSchemaCmd() *cobra.Command {
	var refresh bool
	c := &cobra.Command{
		Use:   "schema ID",
		Short: "Fetch the table/column schema for a data source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid data source ID %q", args[0])
			}
			cli, _, err := resolveClient()
			if err != nil {
				return err
			}
			schema, err := cli.GetSchema(cmd.Context(), id, refresh)
			if err != nil {
				return err
			}
			f, err := parseFormat()
			if err != nil {
				return err
			}
			// Flatten to table/column rows for table/csv; raw for json.
			if f == output.FormatJSON {
				return output.Object(cmd.OutOrStdout(), schema, f)
			}
			rows := make([][]string, 0, 64)
			for _, t := range schema.Tables {
				if len(t.Columns) == 0 {
					rows = append(rows, []string{t.Name, "", ""})
					continue
				}
				for _, col := range t.Columns {
					rows = append(rows, []string{t.Name, col.Name, col.Type})
				}
			}
			return output.Records{
				Columns: []string{"table", "column", "type"},
				Rows:    rows,
			}.Render(cmd.OutOrStdout(), f)
		},
	}
	c.Flags().BoolVar(&refresh, "refresh", false, "request a schema refresh on the server")
	return c
}
