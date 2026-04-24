package cmd

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/leroy/redash/internal/client"
	"github.com/leroy/redash/internal/output"
	"github.com/spf13/cobra"
)

func newWidgetsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "widgets",
		Aliases: []string{"widget"},
		Short:   "Manage dashboard widgets (visualizations and text)",
	}
	c.AddCommand(newWidgetsAddCmd())
	c.AddCommand(newWidgetsUpdateCmd())
	c.AddCommand(newWidgetsRemoveCmd())
	return c
}

func newWidgetsAddCmd() *cobra.Command {
	var (
		dashboardID     int
		visualizationID int
		text            string
		width           int
		col, row        int
		sizeX, sizeY    int
		options         string
	)
	c := &cobra.Command{
		Use:   "add",
		Short: "Add a widget (visualization or text) to a dashboard",
		Long: `Add a widget to a dashboard.

Exactly one of --visualization or --text must be provided. Position on
the dashboard grid is controlled with --col/--row/--size-x/--size-y, or
supply a full options JSON via --options (inline, file path, or - for
stdin).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dashboardID <= 0 {
				return errors.New("--dashboard is required")
			}
			if (visualizationID == 0) == (text == "") {
				return errors.New("pass exactly one of --visualization or --text")
			}

			in := client.AddWidgetInput{
				DashboardID:     dashboardID,
				VisualizationID: visualizationID,
				Text:            text,
				Width:           width,
			}
			if options != "" {
				raw, err := readJSONArg(options)
				if err != nil {
					return fmt.Errorf("--options: %w", err)
				}
				in.Options = raw
			} else {
				in.Options = client.DefaultWidgetOptions(client.Position{
					Col: col, Row: row, SizeX: sizeX, SizeY: sizeY,
				})
			}

			cli, _, err := resolveClient()
			if err != nil {
				return err
			}
			w, err := cli.AddWidget(cmd.Context(), in)
			if err != nil {
				return err
			}
			f, err := parseFormat()
			if err != nil {
				return err
			}
			logf("added widget %d to dashboard %d", w.ID, dashboardID)
			return output.Object(cmd.OutOrStdout(), w, f)
		},
	}
	c.Flags().IntVar(&dashboardID, "dashboard", 0, "dashboard ID to add to (required)")
	c.Flags().IntVar(&visualizationID, "visualization", 0, "visualization ID to pin (mutually exclusive with --text)")
	c.Flags().StringVar(&text, "text", "", "markdown text for a text widget (mutually exclusive with --visualization)")
	c.Flags().IntVar(&width, "width", 0, "legacy width field (default 1)")
	c.Flags().IntVar(&col, "col", 0, "grid column (0-indexed)")
	c.Flags().IntVar(&row, "row", 0, "grid row (0-indexed)")
	c.Flags().IntVar(&sizeX, "size-x", 0, "grid width in columns (default 3)")
	c.Flags().IntVar(&sizeY, "size-y", 0, "grid height in rows (default 8)")
	c.Flags().StringVar(&options, "options", "", "full options JSON (overrides --col/--row/--size-*; inline, file path, or - for stdin)")
	return c
}

func newWidgetsUpdateCmd() *cobra.Command {
	var (
		text    string
		width   int
		options string
	)
	c := &cobra.Command{
		Use:   "update ID",
		Short: "Update a widget's text, width, or options (position)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid widget ID %q", args[0])
			}
			in := client.UpdateWidgetInput{}
			if cmd.Flags().Changed("text") {
				in.Text = &text
			}
			if cmd.Flags().Changed("width") {
				in.Width = &width
			}
			if options != "" {
				raw, err := readJSONArg(options)
				if err != nil {
					return fmt.Errorf("--options: %w", err)
				}
				in.Options = raw
			}
			if in.Text == nil && in.Width == nil && in.Options == nil {
				return errors.New("nothing to update: pass --text, --width, or --options")
			}
			cli, _, err := resolveClient()
			if err != nil {
				return err
			}
			w, err := cli.UpdateWidget(cmd.Context(), id, in)
			if err != nil {
				return err
			}
			f, err := parseFormat()
			if err != nil {
				return err
			}
			logf("updated widget %d", w.ID)
			return output.Object(cmd.OutOrStdout(), w, f)
		},
	}
	c.Flags().StringVar(&text, "text", "", "new markdown text (text widgets only)")
	c.Flags().IntVar(&width, "width", 0, "new legacy width")
	c.Flags().StringVar(&options, "options", "", "new options JSON (inline, file path, or - for stdin)")
	return c
}

func newWidgetsRemoveCmd() *cobra.Command {
	var yes bool
	c := &cobra.Command{
		Use:     "remove ID",
		Aliases: []string{"rm", "delete"},
		Short:   "Remove a widget from its dashboard",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid widget ID %q", args[0])
			}
			if !yes {
				return errors.New("refusing to remove without --yes")
			}
			cli, _, err := resolveClient()
			if err != nil {
				return err
			}
			if err := cli.RemoveWidget(cmd.Context(), id); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "removed widget %d\n", id)
			return nil
		},
	}
	c.Flags().BoolVar(&yes, "yes", false, "confirm removal (required)")
	return c
}
