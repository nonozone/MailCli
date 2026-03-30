package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nonozone/MailCli/pkg/driver"
	"github.com/nonozone/MailCli/pkg/schema"
)

func newMoveCmd() *cobra.Command {
	var (
		configPath string
		account    string
		mailbox    string
	)

	cmd := &cobra.Command{
		Use:   "move [id] [dest-mailbox]",
		Short: "Move a message to another mailbox",
		Long: `Move a message by copying it to dest-mailbox then expunging the original.

The message ID can be any format supported by the driver:
  - Message-ID header:  <abc@example.com>  or  abc@example.com
  - IMAP UID:           uid:42  or  imap:uid:42
  - IMAP sequence num:  42

Common dest-mailbox values: Archive, Trash, Spam, Sent`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			dest := args[1]

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
				return fmt.Errorf("driver %q does not support move", selectedAccount.Driver)
			}

			if err := w.Move(cmd.Context(), id, dest); err != nil {
				return err
			}

			return writeJSON(cmd.OutOrStdout(), &schema.OperationResult{
				OK:      true,
				ID:      id,
				Account: selectedAccount.Name,
				Extra:   map[string]string{"dest_mailbox": dest},
			})
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "config file path")
	cmd.Flags().StringVar(&account, "account", "", "account name override")
	cmd.Flags().StringVar(&mailbox, "mailbox", "", "source mailbox (default: account mailbox)")
	return cmd
}
