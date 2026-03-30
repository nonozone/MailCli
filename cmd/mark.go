package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nonozone/MailCli/pkg/driver"
	"github.com/nonozone/MailCli/pkg/schema"
)

func newMarkCmd() *cobra.Command {
	var (
		configPath string
		account    string
		mailbox    string
		unread     bool
	)

	cmd := &cobra.Command{
		Use:   "mark [id]",
		Short: "Mark a message as read or unread",
		Long: `Set or clear the \\Seen flag on a message.

By default marks as read. Pass --unread to clear the flag.

The message ID can be any format supported by the driver:
  - Message-ID header:  <abc@example.com>  or  abc@example.com
  - IMAP UID:           uid:42  or  imap:uid:42
  - IMAP sequence num:  42`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			selectedAccount, err := resolveSelectedAccount(configPath, account, "")
			if err != nil {
				return err
			}

			if mailbox != "" {
				selectedAccount.Mailbox = mailbox
			}

			drv, err := driverFactoryFunc(selectedAccount)
			if err != nil {
				return err
			}

			w, ok := drv.(driver.Writer)
			if !ok {
				return fmt.Errorf("driver %q does not support mark", selectedAccount.Driver)
			}

			read := !unread
			if err := w.MarkRead(cmd.Context(), args[0], read); err != nil {
				return err
			}

			state := "read"
			if !read {
				state = "unread"
			}

			return writeJSON(cmd.OutOrStdout(), &schema.OperationResult{
				OK:      true,
				ID:      args[0],
				Account: selectedAccount.Name,
				Extra:   map[string]string{"state": state},
			})
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "config file path")
	cmd.Flags().StringVar(&account, "account", "", "account name override")
	cmd.Flags().StringVar(&mailbox, "mailbox", "", "mailbox override (default: account mailbox)")
	cmd.Flags().BoolVar(&unread, "unread", false, "mark as unread instead of read")
	return cmd
}
