package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/leroy/redash/internal/config"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
	}
	c.AddCommand(newConfigPathCmd())
	c.AddCommand(newConfigShowCmd())
	c.AddCommand(newConfigInitCmd())
	c.AddCommand(newConfigSetCmd())
	c.AddCommand(newConfigRemoveCmd())
	c.AddCommand(newConfigUseCmd())
	return c
}

func newConfigPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the path to the config file",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			p := flagConfigPath
			if p == "" {
				p = config.DefaultPath()
			}
			fmt.Fprintln(cmd.OutOrStdout(), p)
		},
	}
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print the resolved active profile (API key redacted)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path := flagConfigPath
			if path == "" {
				path = config.DefaultPath()
			}
			f, err := config.Load(path)
			if err != nil {
				return err
			}
			name, p, err := config.Resolve(f, flagProfile)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "profile:  %s\n", name)
			fmt.Fprintf(cmd.OutOrStdout(), "url:      %s\n", p.URL)
			fmt.Fprintf(cmd.OutOrStdout(), "api_key:  %s\n", redact(p.APIKey))
			fmt.Fprintf(cmd.OutOrStdout(), "timeout:  %s\n", p.Timeout)
			fmt.Fprintf(cmd.OutOrStdout(), "insecure: %t\n", p.Insecure)
			return nil
		},
	}
}

func newConfigInitCmd() *cobra.Command {
	var (
		name    string
		url     string
		apiKey  string
		timeout time.Duration
		set     bool
	)
	c := &cobra.Command{
		Use:   "init",
		Short: "Create or update a named profile (interactive if flags omitted)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path := flagConfigPath
			if path == "" {
				path = config.DefaultPath()
			}
			f, err := config.Load(path)
			if err != nil {
				return err
			}
			reader := bufio.NewReader(os.Stdin)
			if name == "" {
				name, err = prompt(reader, "profile name", "default")
				if err != nil {
					return err
				}
			}
			if url == "" {
				url, err = prompt(reader, "redash url (e.g. https://redash.example.com)", "")
				if err != nil {
					return err
				}
			}
			if apiKey == "" {
				apiKey, err = prompt(reader, "api key", "")
				if err != nil {
					return err
				}
			}
			if url == "" || apiKey == "" {
				return errors.New("url and api key are required")
			}
			p := f.Profiles[name]
			p.URL = strings.TrimSpace(url)
			p.APIKey = strings.TrimSpace(apiKey)
			if timeout > 0 {
				p.Timeout = timeout
			}
			f.Profiles[name] = p
			if set || f.DefaultProfile == "" {
				f.DefaultProfile = name
			}
			if err := config.Save(path, f); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "wrote profile %q to %s\n", name, path)
			return nil
		},
	}
	c.Flags().StringVar(&name, "name", "", "profile name")
	c.Flags().StringVar(&url, "url", "", "redash URL")
	c.Flags().StringVar(&apiKey, "api-key", "", "redash API key")
	c.Flags().DurationVar(&timeout, "timeout", 0, "request timeout")
	c.Flags().BoolVar(&set, "default", false, "set this profile as the default")
	return c
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set PROFILE KEY VALUE",
		Short: "Set a single field on a profile (url|api_key|timeout|insecure)",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := flagConfigPath
			if path == "" {
				path = config.DefaultPath()
			}
			f, err := config.Load(path)
			if err != nil {
				return err
			}
			p := f.Profiles[args[0]]
			switch strings.ToLower(args[1]) {
			case "url":
				p.URL = args[2]
			case "api_key", "api-key":
				p.APIKey = args[2]
			case "timeout":
				d, err := time.ParseDuration(args[2])
				if err != nil {
					return fmt.Errorf("timeout: %w", err)
				}
				p.Timeout = d
			case "insecure":
				switch strings.ToLower(args[2]) {
				case "true", "yes", "1":
					p.Insecure = true
				case "false", "no", "0":
					p.Insecure = false
				default:
					return fmt.Errorf("insecure must be true or false")
				}
			default:
				return fmt.Errorf("unknown key %q (want: url, api_key, timeout, insecure)", args[1])
			}
			f.Profiles[args[0]] = p
			if f.DefaultProfile == "" {
				f.DefaultProfile = args[0]
			}
			if err := config.Save(path, f); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "updated %s.%s\n", args[0], args[1])
			return nil
		},
	}
}

func newConfigRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "remove PROFILE",
		Aliases: []string{"rm"},
		Short:   "Remove a profile",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := flagConfigPath
			if path == "" {
				path = config.DefaultPath()
			}
			f, err := config.Load(path)
			if err != nil {
				return err
			}
			if _, ok := f.Profiles[args[0]]; !ok {
				return fmt.Errorf("profile %q not found", args[0])
			}
			delete(f.Profiles, args[0])
			if f.DefaultProfile == args[0] {
				f.DefaultProfile = ""
				for k := range f.Profiles {
					f.DefaultProfile = k
					break
				}
			}
			if err := config.Save(path, f); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "removed profile %q\n", args[0])
			return nil
		},
	}
}

func newConfigUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use PROFILE",
		Short: "Set the default profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := flagConfigPath
			if path == "" {
				path = config.DefaultPath()
			}
			f, err := config.Load(path)
			if err != nil {
				return err
			}
			if _, ok := f.Profiles[args[0]]; !ok {
				return fmt.Errorf("profile %q not found (run `redash config init`)", args[0])
			}
			f.DefaultProfile = args[0]
			if err := config.Save(path, f); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "default profile set to %q\n", args[0])
			return nil
		},
	}
}

func prompt(r *bufio.Reader, label, def string) (string, error) {
	if def != "" {
		fmt.Fprintf(os.Stderr, "%s [%s]: ", label, def)
	} else {
		fmt.Fprintf(os.Stderr, "%s: ", label)
	}
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return def, nil
	}
	return line, nil
}

func redact(s string) string {
	if len(s) <= 6 {
		return strings.Repeat("*", len(s))
	}
	return s[:3] + strings.Repeat("*", len(s)-6) + s[len(s)-3:]
}
