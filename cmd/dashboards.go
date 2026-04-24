package cmd

import (
	"strconv"
	"strings"

	"github.com/leroy/redash/internal/client"
	"github.com/leroy/redash/internal/output"
	"github.com/spf13/cobra"
)

func newDashboardsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "dashboards",
		Aliases: []string{"dashboard"},
		Short:   "Inspect dashboards",
	}
	c.AddCommand(newDashboardsListCmd())
	c.AddCommand(newDashboardsGetCmd())
	return c
}

func newDashboardsListCmd() *cobra.Command {
	var (
		page     int
		pageSize int
		search   string
	)
	c := &cobra.Command{
		Use:   "list",
		Short: "List dashboards",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, _, err := resolveClient()
			if err != nil {
				return err
			}
			list, err := cli.ListDashboards(cmd.Context(), client.ListDashboardsParams{
				Page: page, PageSize: pageSize, Search: search,
			})
			if err != nil {
				return err
			}
			f, err := parseFormat()
			if err != nil {
				return err
			}
			rows := make([][]string, len(list.Results))
			for i, d := range list.Results {
				user := ""
				if d.User != nil {
					user = d.User.Email
				}
				rows[i] = []string{
					strconv.Itoa(d.ID),
					d.Slug,
					d.Name,
					strings.Join(d.Tags, ","),
					user,
					d.UpdatedAt,
				}
			}
			logf("showing %d / %d dashboards", len(list.Results), list.Count)
			return output.Records{
				Columns: []string{"id", "slug", "name", "tags", "user", "updated_at"},
				Rows:    rows,
			}.Render(cmd.OutOrStdout(), f)
		},
	}
	c.Flags().IntVar(&page, "page", 1, "page number")
	c.Flags().IntVar(&pageSize, "page-size", 25, "page size")
	c.Flags().StringVarP(&search, "search", "s", "", "search term")
	return c
}

func newDashboardsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get SLUG_OR_ID",
		Short: "Get a dashboard by slug (or numeric ID on newer Redash)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, _, err := resolveClient()
			if err != nil {
				return err
			}
			d, err := cli.GetDashboard(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			f, err := parseFormat()
			if err != nil {
				return err
			}
			if f == output.FormatJSON && len(d.Raw) > 0 {
				cmd.OutOrStdout().Write(append(indentJSON(d.Raw), '\n'))
				return nil
			}
			return output.Object(cmd.OutOrStdout(), d, f)
		},
	}
}
