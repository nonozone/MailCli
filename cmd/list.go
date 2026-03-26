package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/yourname/mailcli/internal/config"
	"github.com/yourname/mailcli/pkg/schema"
)

func newListCmd() *cobra.Command {
	var (
		configPath string
		account    string
		format     string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List messages from the selected account",
		RunE: func(cmd *cobra.Command, args []string) error {
			if configPath == "" {
				configPath = config.DefaultPath()
			}

			cfg, err := loadConfigFunc(configPath)
			if err != nil {
				return err
			}

			selected, err := cfg.ResolveAccount(account)
			if err != nil {
				return err
			}

			drv, err := driverFactoryFunc(selected)
			if err != nil {
				return err
			}

			items, err := drv.List(cmd.Context(), schema.SearchQuery{})
			if err != nil {
				return err
			}

			return writeMessageList(cmd.OutOrStdout(), items, format)
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "config file path")
	cmd.Flags().StringVar(&account, "account", "", "account name override")
	cmd.Flags().StringVar(&format, "format", "json", "output format: json, table")
	return cmd
}

func writeMessageList(out io.Writer, items []schema.MessageMetaSummary, format string) error {
	switch format {
	case "", "json":
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(items)
	case "table":
		table := tablewriter.NewWriter(out)
		table.SetHeader([]string{"ID", "Subject"})
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			rows = append(rows, []string{item.ID, item.Subject})
		}
		table.AppendBulk(rows)
		table.Render()
		return nil
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}
