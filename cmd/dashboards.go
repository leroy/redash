package cmd

import (
	"errors"
	"fmt"
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
		Short:   "Manage dashboards",
	}
	c.AddCommand(newDashboardsListCmd())
	c.AddCommand(newDashboardsGetCmd())
	c.AddCommand(newDashboardsCreateCmd())
	c.AddCommand(newDashboardsUpdateCmd())
	c.AddCommand(newDashboardsArchiveCmd())
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
				if _, err := cmd.OutOrStdout().Write(append(indentJSON(d.Raw), '\n')); err != nil {
					return err
				}
				return nil
			}
			return output.Object(cmd.OutOrStdout(), d, f)
		},
	}
}

func newDashboardsCreateCmd() *cobra.Command {
	var name string
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a new (empty, draft) dashboard",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if name == "" {
				return errors.New("--name is required")
			}
			cli, _, err := resolveClient()
			if err != nil {
				return err
			}
			d, err := cli.CreateDashboard(cmd.Context(), client.CreateDashboardInput{Name: name})
			if err != nil {
				return err
			}
			f, err := parseFormat()
			if err != nil {
				return err
			}
			logf("created dashboard %d (%s)", d.ID, d.Slug)
			return output.Object(cmd.OutOrStdout(), d, f)
		},
	}
	c.Flags().StringVar(&name, "name", "", "dashboard name (required)")
	return c
}

func newDashboardsUpdateCmd() *cobra.Command {
	var (
		name            string
		tags            []string
		publish         bool
		unpublish       bool
		filtersEnabled  bool
		filtersDisabled bool
	)
	c := &cobra.Command{
		Use:   "update ID",
		Short: "Update dashboard metadata (name, tags, draft state, filters)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid dashboard ID %q", args[0])
			}
			if publish && unpublish {
				return errors.New("pass --publish or --unpublish, not both")
			}
			if filtersEnabled && filtersDisabled {
				return errors.New("pass --enable-filters or --disable-filters, not both")
			}

			in := client.UpdateDashboardInput{}
			if cmd.Flags().Changed("name") {
				in.Name = &name
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
			if filtersEnabled {
				t := true
				in.DashboardFiltersEnabled = &t
			} else if filtersDisabled {
				fl := false
				in.DashboardFiltersEnabled = &fl
			}
			if in.Name == nil && in.Tags == nil && in.IsDraft == nil && in.DashboardFiltersEnabled == nil {
				return errors.New("nothing to update: pass --name, --tag, --publish, --unpublish, --enable-filters, or --disable-filters")
			}

			cli, _, err := resolveClient()
			if err != nil {
				return err
			}
			d, err := cli.UpdateDashboard(cmd.Context(), id, in)
			if err != nil {
				return err
			}
			f, err := parseFormat()
			if err != nil {
				return err
			}
			logf("updated dashboard %d", d.ID)
			return output.Object(cmd.OutOrStdout(), d, f)
		},
	}
	c.Flags().StringVar(&name, "name", "", "new dashboard name")
	c.Flags().StringArrayVar(&tags, "tag", nil, "replacement tag (repeatable; replaces existing tags)")
	c.Flags().BoolVar(&publish, "publish", false, "mark as published (is_draft=false)")
	c.Flags().BoolVar(&unpublish, "unpublish", false, "mark as draft (is_draft=true)")
	c.Flags().BoolVar(&filtersEnabled, "enable-filters", false, "enable dashboard-level filters")
	c.Flags().BoolVar(&filtersDisabled, "disable-filters", false, "disable dashboard-level filters")
	return c
}

func newDashboardsArchiveCmd() *cobra.Command {
	var yes bool
	c := &cobra.Command{
		Use:   "archive ID",
		Short: "Archive (soft-delete) a dashboard",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid dashboard ID %q", args[0])
			}
			if !yes {
				return errors.New("refusing to archive without --yes")
			}
			cli, _, err := resolveClient()
			if err != nil {
				return err
			}
			if err := cli.ArchiveDashboard(cmd.Context(), id); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "archived dashboard %d\n", id)
			return nil
		},
	}
	c.Flags().BoolVar(&yes, "yes", false, "confirm archival (required)")
	return c
}
