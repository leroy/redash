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

func newUsersCmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "users",
		Aliases: []string{"user"},
		Short:   "Manage users",
	}
	c.AddCommand(newUsersListCmd())
	c.AddCommand(newUsersGetCmd())
	c.AddCommand(newUsersCreateCmd())
	c.AddCommand(newUsersDisableCmd())
	c.AddCommand(newUsersEnableCmd())
	return c
}

func newUsersListCmd() *cobra.Command {
	var (
		page     int
		pageSize int
		search   string
		disabled bool
	)
	c := &cobra.Command{
		Use:   "list",
		Short: "List users",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, _, err := resolveClient()
			if err != nil {
				return err
			}
			list, err := cli.ListUsers(cmd.Context(), client.ListUsersParams{
				Page: page, PageSize: pageSize, Search: search, Disabled: disabled,
			})
			if err != nil {
				return err
			}
			f, err := parseFormat()
			if err != nil {
				return err
			}
			rows := make([][]string, len(list.Results))
			for i, u := range list.Results {
				groups := make([]string, len(u.Groups))
				for j, g := range u.Groups {
					groups[j] = strconv.Itoa(g)
				}
				rows[i] = []string{
					strconv.Itoa(u.ID),
					u.Name,
					u.Email,
					strings.Join(groups, ","),
					strconv.FormatBool(u.IsDisabled),
				}
			}
			logf("showing %d / %d users", len(list.Results), list.Count)
			return output.Records{
				Columns: []string{"id", "name", "email", "groups", "disabled"},
				Rows:    rows,
			}.Render(cmd.OutOrStdout(), f)
		},
	}
	c.Flags().IntVar(&page, "page", 1, "page number")
	c.Flags().IntVar(&pageSize, "page-size", 25, "page size")
	c.Flags().StringVarP(&search, "search", "s", "", "search term")
	c.Flags().BoolVar(&disabled, "disabled", false, "include disabled users")
	return c
}

func newUsersGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get ID",
		Short: "Get a user by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid user ID %q", args[0])
			}
			cli, _, err := resolveClient()
			if err != nil {
				return err
			}
			u, err := cli.GetUser(cmd.Context(), id)
			if err != nil {
				return err
			}
			f, err := parseFormat()
			if err != nil {
				return err
			}
			if f == output.FormatJSON && len(u.Raw) > 0 {
				cmd.OutOrStdout().Write(append(indentJSON(u.Raw), '\n'))
				return nil
			}
			return output.Object(cmd.OutOrStdout(), u, f)
		},
	}
}

func newUsersCreateCmd() *cobra.Command {
	var (
		name   string
		email  string
		groups []int
	)
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a new user (sends invitation email)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if name == "" || email == "" {
				return errors.New("--name and --email are required")
			}
			cli, _, err := resolveClient()
			if err != nil {
				return err
			}
			u, err := cli.CreateUser(cmd.Context(), client.CreateUserInput{
				Name: name, Email: email, Groups: groups,
			})
			if err != nil {
				return err
			}
			f, err := parseFormat()
			if err != nil {
				return err
			}
			logf("created user %d <%s>", u.ID, u.Email)
			return output.Object(cmd.OutOrStdout(), u, f)
		},
	}
	c.Flags().StringVar(&name, "name", "", "user name (required)")
	c.Flags().StringVar(&email, "email", "", "user email (required)")
	c.Flags().IntSliceVar(&groups, "group", nil, "group ID (repeatable)")
	return c
}

func newUsersDisableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable ID",
		Short: "Disable a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid user ID %q", args[0])
			}
			cli, _, err := resolveClient()
			if err != nil {
				return err
			}
			if err := cli.DisableUser(cmd.Context(), id); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "disabled user %d\n", id)
			return nil
		},
	}
}

func newUsersEnableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "enable ID",
		Short: "Re-enable a previously disabled user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid user ID %q", args[0])
			}
			cli, _, err := resolveClient()
			if err != nil {
				return err
			}
			if err := cli.EnableUser(cmd.Context(), id); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "enabled user %d\n", id)
			return nil
		},
	}
}
