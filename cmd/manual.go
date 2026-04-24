package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/leroy/redash/internal/manual"
	"github.com/spf13/cobra"
)

func newManualCmd() *cobra.Command {
	var (
		topic      string
		format     string
		listTopics bool
	)
	c := &cobra.Command{
		Use:   "manual",
		Short: "Print the agent-oriented usage manual (agents: start here)",
		Long: `Print the agent-oriented usage manual for this build of the CLI.

The manual is compiled into the binary and always matches the installed
version — unlike README files, which can drift. This command is the
recommended entry point for AI agents driving the CLI.

Examples:
  redash manual                      # full markdown to stdout
  redash manual --topic query        # one section only
  redash manual --list-topics        # enumerate available topics
  redash manual --format json        # structured JSON catalog`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			w := cmd.OutOrStdout()

			if listTopics {
				for _, t := range manual.Topics() {
					if _, err := fmt.Fprintf(w, "%-14s  %s\n", t.Name, t.Title); err != nil {
						return err
					}
				}
				return nil
			}

			switch format {
			case "", "markdown", "md":
				md, err := manual.Markdown(topic)
				if err != nil {
					return err
				}
				_, err = fmt.Fprint(w, md)
				return err
			case "json":
				if topic != "" {
					t, err := manual.TopicByName(topic)
					if err != nil {
						return err
					}
					enc := json.NewEncoder(w)
					enc.SetIndent("", "  ")
					enc.SetEscapeHTML(false)
					return enc.Encode(t)
				}
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				enc.SetEscapeHTML(false)
				return enc.Encode(manual.Catalog(version))
			default:
				return fmt.Errorf("unknown --format %q (want: markdown, json)", format)
			}
		},
	}
	c.Flags().StringVar(&topic, "topic", "", "print a single section (see --list-topics)")
	c.Flags().StringVar(&format, "format", "markdown", "output format: markdown, json")
	c.Flags().BoolVar(&listTopics, "list-topics", false, "list available topic names and exit")
	return c
}
