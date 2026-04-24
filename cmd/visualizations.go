package cmd

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/leroy/redash/internal/client"
	"github.com/leroy/redash/internal/output"
	"github.com/spf13/cobra"
)

func newVisualizationsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "visualizations",
		Aliases: []string{"viz", "visualization"},
		Short:   "Manage query visualizations (charts, counters, pivots, ...)",
	}
	c.AddCommand(newVisualizationsCreateCmd())
	c.AddCommand(newVisualizationsUpdateCmd())
	c.AddCommand(newVisualizationsDeleteCmd())
	return c
}

func newVisualizationsCreateCmd() *cobra.Command {
	var (
		queryID     int
		vizType     string
		name        string
		description string
		options     string
	)
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a new visualization on a query",
		Long: `Create a new visualization attached to a query.

--options accepts either inline JSON, a file path, or - for stdin. The
shape is type-specific; inspect an existing visualization of the same
type with ` + "`redash queries get <id>` --format json" + ` to see a valid example.

Common types: TABLE, CHART, COUNTER, PIVOT_TABLE, MAP, WORD_CLOUD,
SUNBURST_SEQUENCE, SANKEY, BOXPLOT, CHOROPLETH, DETAILS.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if queryID <= 0 {
				return errors.New("--query is required")
			}
			if vizType == "" {
				return errors.New("--type is required")
			}
			if name == "" {
				return errors.New("--name is required")
			}
			in := client.CreateVisualizationInput{
				QueryID:     queryID,
				Type:        vizType,
				Name:        name,
				Description: description,
			}
			if options != "" {
				raw, err := readJSONArg(options)
				if err != nil {
					return fmt.Errorf("--options: %w", err)
				}
				in.Options = raw
			}
			cli, _, err := resolveClient()
			if err != nil {
				return err
			}
			v, err := cli.CreateVisualization(cmd.Context(), in)
			if err != nil {
				return err
			}
			f, err := parseFormat()
			if err != nil {
				return err
			}
			logf("created visualization %d (%s) on query %d", v.ID, v.Type, queryID)
			return output.Object(cmd.OutOrStdout(), v, f)
		},
	}
	c.Flags().IntVar(&queryID, "query", 0, "query ID to attach to (required)")
	c.Flags().StringVar(&vizType, "type", "", "visualization type (e.g. CHART, COUNTER, PIVOT_TABLE; required)")
	c.Flags().StringVar(&name, "name", "", "visualization name (required)")
	c.Flags().StringVar(&description, "description", "", "visualization description")
	c.Flags().StringVar(&options, "options", "", "type-specific options JSON (inline, file path, or - for stdin)")
	return c
}

func newVisualizationsUpdateCmd() *cobra.Command {
	var (
		vizType     string
		name        string
		description string
		options     string
	)
	c := &cobra.Command{
		Use:   "update ID",
		Short: "Update a visualization's name, type, description, or options",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid visualization ID %q", args[0])
			}
			in := client.UpdateVisualizationInput{}
			if cmd.Flags().Changed("type") {
				in.Type = &vizType
			}
			if cmd.Flags().Changed("name") {
				in.Name = &name
			}
			if cmd.Flags().Changed("description") {
				in.Description = &description
			}
			if options != "" {
				raw, err := readJSONArg(options)
				if err != nil {
					return fmt.Errorf("--options: %w", err)
				}
				in.Options = raw
			}
			if in.Type == nil && in.Name == nil && in.Description == nil && in.Options == nil {
				return errors.New("nothing to update: pass --type, --name, --description, or --options")
			}
			cli, _, err := resolveClient()
			if err != nil {
				return err
			}
			v, err := cli.UpdateVisualization(cmd.Context(), id, in)
			if err != nil {
				return err
			}
			f, err := parseFormat()
			if err != nil {
				return err
			}
			logf("updated visualization %d", v.ID)
			return output.Object(cmd.OutOrStdout(), v, f)
		},
	}
	c.Flags().StringVar(&vizType, "type", "", "new visualization type")
	c.Flags().StringVar(&name, "name", "", "new visualization name")
	c.Flags().StringVar(&description, "description", "", "new visualization description")
	c.Flags().StringVar(&options, "options", "", "new options JSON (inline, file path, or - for stdin)")
	return c
}

func newVisualizationsDeleteCmd() *cobra.Command {
	var yes bool
	c := &cobra.Command{
		Use:   "delete ID",
		Short: "Delete a visualization (the implicit TABLE viz on a query cannot be deleted)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid visualization ID %q", args[0])
			}
			if !yes {
				return errors.New("refusing to delete without --yes")
			}
			cli, _, err := resolveClient()
			if err != nil {
				return err
			}
			if err := cli.DeleteVisualization(cmd.Context(), id); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "deleted visualization %d\n", id)
			return nil
		},
	}
	c.Flags().BoolVar(&yes, "yes", false, "confirm deletion (required)")
	return c
}
