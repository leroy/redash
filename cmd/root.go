// Package cmd implements the redash CLI command tree.
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/leroy/redash/internal/client"
	"github.com/leroy/redash/internal/config"
	"github.com/leroy/redash/internal/output"
	"github.com/spf13/cobra"
)

// Build metadata, set at link-time by goreleaser.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Persistent flags.
var (
	flagConfigPath string
	flagProfile    string
	flagFormat     string
	flagTimeout    time.Duration
	flagInsecure   bool
	flagQuiet      bool
)

// Root is the top-level command.
var Root = &cobra.Command{
	Use:           "redash",
	Short:         "Command-line client for the Redash API",
	Long:          "redash is a CLI for the Redash REST API: run queries, manage saved queries, inspect data source schemas, list dashboards, and more.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	Root.PersistentFlags().StringVar(&flagConfigPath, "config", "", "path to config file (default: "+config.DefaultPath()+")")
	Root.PersistentFlags().StringVarP(&flagProfile, "profile", "p", "", "profile to use (default: file's default_profile or REDASH_PROFILE)")
	Root.PersistentFlags().StringVarP(&flagFormat, "format", "o", "table", "output format: table, json, csv")
	Root.PersistentFlags().DurationVar(&flagTimeout, "timeout", 0, "request timeout (default: 30s or profile value)")
	Root.PersistentFlags().BoolVar(&flagInsecure, "insecure", false, "skip TLS verification")
	Root.PersistentFlags().BoolVarP(&flagQuiet, "quiet", "q", false, "suppress informational output to stderr")

	Root.AddCommand(newQueryCmd())
	Root.AddCommand(newQueriesCmd())
	Root.AddCommand(newDataSourcesCmd())
	Root.AddCommand(newDashboardsCmd())
	Root.AddCommand(newUsersCmd())
	Root.AddCommand(newConfigCmd())
	Root.AddCommand(newVersionCmd())
}

// Execute runs the CLI. It installs a signal handler that cancels the
// root context on SIGINT/SIGTERM so in-flight requests are aborted.
func Execute() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := Root.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "redash:", err.Error())
		os.Exit(1)
	}
}

// resolveClient loads config, resolves the active profile, and returns
// a ready-to-use API client. It's called by every subcommand that needs
// to talk to Redash.
func resolveClient() (*client.Client, string, error) {
	path := flagConfigPath
	if path == "" {
		path = config.DefaultPath()
	}
	f, err := config.Load(path)
	if err != nil {
		return nil, "", err
	}
	name, profile, err := config.Resolve(f, flagProfile)
	if err != nil {
		return nil, "", err
	}
	if flagTimeout > 0 {
		profile.Timeout = flagTimeout
	}
	opts := []client.Option{client.WithUserAgent("redash-cli/" + version)}
	if flagInsecure || profile.Insecure {
		opts = append(opts, client.WithInsecure())
	}
	c, err := client.New(profile.URL, profile.APIKey, profile.Timeout, opts...)
	if err != nil {
		return nil, "", err
	}
	return c, name, nil
}

// parseFormat wraps output.ParseFormat with the user's flag value.
func parseFormat() (output.Format, error) {
	return output.ParseFormat(flagFormat)
}

func logf(format string, args ...any) {
	if flagQuiet {
		return
	}
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}
