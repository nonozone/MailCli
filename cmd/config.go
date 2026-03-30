package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nonozone/MailCli/internal/config"
	"github.com/nonozone/MailCli/pkg/schema"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage mailcli configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newConfigShowCmd())
	cmd.AddCommand(newConfigTestCmd())
	return cmd
}

// config show ─────────────────────────────────────────────────────────────────

func newConfigShowCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Print the current configuration (passwords redacted)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if configPath == "" {
				configPath = config.DefaultPath()
			}

			cfg, err := loadConfigFunc(configPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "config file:     %s\n", configPath)
			fmt.Fprintf(out, "current account: %s\n", cfg.CurrentAccount)
			fmt.Fprintf(out, "accounts:\n")
			for _, acc := range cfg.Accounts {
				fmt.Fprintf(out, "\n  name:     %s\n", acc.Name)
				fmt.Fprintf(out, "  driver:   %s\n", acc.Driver)
				switch strings.ToLower(acc.Driver) {
				case "imap":
					fmt.Fprintf(out, "  imap:\n")
					fmt.Fprintf(out, "    host:     %s:%d\n", acc.Host, acc.Port)
					fmt.Fprintf(out, "    username: %s\n", acc.Username)
					fmt.Fprintf(out, "    tls:      %v\n", acc.TLS)
					if acc.Mailbox != "" {
						fmt.Fprintf(out, "    mailbox:  %s\n", acc.Mailbox)
					}
					if acc.SMTPHost != "" {
						fmt.Fprintf(out, "  smtp:\n")
						fmt.Fprintf(out, "    host:     %s:%d\n", acc.SMTPHost, acc.SMTPPort)
						smtpUser := acc.SMTPUsername
						if smtpUser == "" {
							smtpUser = acc.Username + " (inherited)"
						}
						fmt.Fprintf(out, "    username: %s\n", smtpUser)
						fmt.Fprintf(out, "    tls:      %v\n", acc.SMTPTLS)
					}
				case "dir":
					fmt.Fprintf(out, "  path:     %s\n", acc.Path)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "config file path")
	return cmd
}

// config test ─────────────────────────────────────────────────────────────────

func newConfigTestCmd() *cobra.Command {
	var (
		configPath string
		account    string
	)

	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test the connection for the selected account",
		RunE: func(cmd *cobra.Command, args []string) error {
			selectedAccount, err := resolveSelectedAccount(configPath, account, "")
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "testing account: %s (driver: %s)\n",
				selectedAccount.Name, selectedAccount.Driver)

			drv, err := driverFactoryFunc(selectedAccount)
			if err != nil {
				return fmt.Errorf("driver init failed: %w", err)
			}

			// Attempt a List with limit=1 as a lightweight connectivity probe.
			_, err = drv.List(cmd.Context(), schema.SearchQuery{Limit: 1})
			if err != nil {
				return fmt.Errorf("connection test failed: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "ok: connection successful")
			return nil
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "config file path")
	cmd.Flags().StringVar(&account, "account", "", "account name to test")
	return cmd
}
