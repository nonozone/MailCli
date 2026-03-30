package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nonozone/MailCli/pkg/driver"
	"github.com/nonozone/MailCli/pkg/schema"
)

func newDeleteCmd() *cobra.Command {
	var (
		configPath string
		account    string
		mailbox    string
	)

	cmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Permanently delete a message from the remote mailbox",
		Long: `Permanently delete a message by setting \Deleted and expunging it.

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

			// Override mailbox on the account so the driver targets the right folder.
			if mailbox != "" {
				selectedAccount.Mailbox = mailbox
			}

			drv, err := driverFactoryFunc(selectedAccount)
			if err != nil {
				return err
			}

			w, ok := drv.(driver.Writer)
			if !ok {
				return fmt.Errorf("driver %q does not support delete", selectedAccount.Driver)
			}

			if err := w.Delete(cmd.Context(), args[0]); err != nil {
				return err
			}

			return writeJSON(cmd.OutOrStdout(), &schema.OperationResult{
				OK:      true,
				ID:      args[0],
				Account: selectedAccount.Name,
			})
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "config file path")
	cmd.Flags().StringVar(&account, "account", "", "account name override")
	cmd.Flags().StringVar(&mailbox, "mailbox", "", "source mailbox (default: account mailbox)")
	return cmd
}
