package cmd

import "github.com/spf13/cobra"

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mailcli",
		Short: "AI-native email normalization toolkit",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newParseCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newSendCmd())
	cmd.AddCommand(newReplyCmd())

	return cmd
}
