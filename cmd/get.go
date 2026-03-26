package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yourname/mailcli/pkg/parser"
)

func newGetCmd() *cobra.Command {
	var (
		configPath string
		account    string
		format     string
	)

	cmd := &cobra.Command{
		Use:   "get [id]",
		Short: "Fetch and parse a message by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			selectedAccount, err := resolveSelectedAccount(configPath, account, "")
			if err != nil {
				return err
			}

			drv, err := driverFactoryFunc(selectedAccount)
			if err != nil {
				return err
			}

			raw, err := drv.FetchRaw(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			msg, err := parser.Parse(raw)
			if err != nil {
				return err
			}

			return writeMessage(cmd.OutOrStdout(), msg, format)
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "config file path")
	cmd.Flags().StringVar(&account, "account", "", "account name override")
	cmd.Flags().StringVar(&format, "format", "json", "output format: json, yaml, table")
	return cmd
}
