package cmd

import (
	"github.com/spf13/cobra"
)

// Verbose and Quiet are package-level flags accessible to all subcommands
// via the persistent flags on the root command.
var Verbose bool
var Quiet bool

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mailcli",
		Short: "AI-native email normalization toolkit",
		Long: `mailcli — AI-native email normalization toolkit.

Reads and writes email via IMAP or local .eml files, outputting structured
JSON that is ready for LLM consumption or AI pipeline processing.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Global persistent flags — available to every subcommand.
	cmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "print extra diagnostic output to stderr")
	cmd.PersistentFlags().BoolVarP(&Quiet, "quiet", "q", false, "suppress all non-error output")

	cmd.AddCommand(newParseCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newSyncCmd())
	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newThreadsCmd())
	cmd.AddCommand(newThreadCmd())
	cmd.AddCommand(newSendCmd())
	cmd.AddCommand(newReplyCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newMoveCmd())
	cmd.AddCommand(newMarkCmd())
	cmd.AddCommand(newExportCmd())
	cmd.AddCommand(newWatchCmd())
	cmd.AddCommand(newConfigCmd())

	return cmd
}
