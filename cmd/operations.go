package cmd

import (
	"fmt"

	opstore "github.com/nonozone/MailCli/internal/operations"
	"github.com/nonozone/MailCli/pkg/schema"
	"github.com/spf13/cobra"
)

func newOperationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "operations",
		Short: "Inspect prepared and executed mailbox operations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newOperationsListCmd())
	cmd.AddCommand(newOperationsShowCmd())
	return cmd
}

func newOperationsListCmd() *cobra.Command {
	var operationsPath string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List operation log entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := opstore.NewStore(operationsPath).List()
			if err != nil {
				return err
			}
			return writeJSON(cmd.OutOrStdout(), schema.OperationListResult{Operations: entries})
		},
	}

	cmd.Flags().StringVar(&operationsPath, "operations", "", "operations log path")
	return cmd
}

func newOperationsShowCmd() *cobra.Command {
	var operationsPath string

	cmd := &cobra.Command{
		Use:   "show [operation-id|intent-id]",
		Short: "Show one operation log entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			entry, err := opstore.NewStore(operationsPath).Find(args[0])
			if err != nil {
				return fmt.Errorf("operation lookup failed: %w", err)
			}
			return writeJSON(cmd.OutOrStdout(), entry)
		},
	}

	cmd.Flags().StringVar(&operationsPath, "operations", "", "operations log path")
	return cmd
}
