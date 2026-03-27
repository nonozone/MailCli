package cmd

import (
	"github.com/spf13/cobra"

	mailindex "github.com/nonozone/MailCli/internal/index"
)

func newThreadCmd() *cobra.Command {
	var (
		indexPath string
		account   string
		mailbox   string
		format    string
		limit     int
	)

	cmd := &cobra.Command{
		Use:   "thread [thread_id]",
		Short: "Return full local messages for a selected thread",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store := mailindex.NewFileStore(indexPath)
			results, err := store.ThreadMessages(mailindex.ThreadMessageQuery{
				ThreadID: args[0],
				Account:  account,
				Mailbox:  mailbox,
				Limit:    limit,
			})
			if err != nil {
				return err
			}
			return writeFullSearchResults(cmd.OutOrStdout(), results, format)
		},
	}

	cmd.Flags().StringVar(&indexPath, "index", "", "local index file path")
	cmd.Flags().StringVar(&account, "account", "", "filter local thread messages by account")
	cmd.Flags().StringVar(&mailbox, "mailbox", "", "filter local thread messages by mailbox")
	cmd.Flags().IntVar(&limit, "limit", 50, "maximum number of thread messages")
	cmd.Flags().StringVar(&format, "format", "json", "output format: json")
	return cmd
}
