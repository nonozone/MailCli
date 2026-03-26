package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"github.com/yourname/mailcli/internal/config"
	"github.com/yourname/mailcli/pkg/composer"
	"github.com/yourname/mailcli/pkg/schema"
)

func newSendCmd() *cobra.Command {
	var (
		configPath string
		account    string
		dryRun     bool
	)

	cmd := &cobra.Command{
		Use:   "send [file|-]",
		Short: "Send a draft message or print its MIME in dry-run mode",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, err := readInput(cmd, args[0])
			if err != nil {
				return err
			}

			var draft schema.DraftMessage
			if err := json.Unmarshal(raw, &draft); err != nil {
				return err
			}

			mime, err := composer.ComposeDraft(draft)
			if err != nil {
				return err
			}

			if dryRun {
				_, err := cmd.OutOrStdout().Write(mime)
				return err
			}

			selectedAccount, err := resolveSelectedAccount(configPath, account, draft.Account)
			if err != nil {
				return err
			}

			drv, err := driverFactoryFunc(selectedAccount)
			if err != nil {
				return err
			}

			if err := drv.SendRaw(cmd.Context(), mime); err != nil {
				return err
			}

			result := schema.SendResult{
				OK:       true,
				Provider: selectedAccount.Driver,
				Account:  selectedAccount.Name,
			}
			return writeJSON(cmd.OutOrStdout(), &result)
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "config file path")
	cmd.Flags().StringVar(&account, "account", "", "account name override")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print MIME instead of sending it")
	return cmd
}

func newReplyCmd() *cobra.Command {
	var (
		configPath string
		account    string
		dryRun     bool
	)

	cmd := &cobra.Command{
		Use:   "reply [file|-]",
		Short: "Compile a reply draft into MIME or print it in dry-run mode",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, err := readInput(cmd, args[0])
			if err != nil {
				return err
			}

			var draft schema.ReplyDraft
			if err := json.Unmarshal(raw, &draft); err != nil {
				return err
			}

			mime, err := composer.ComposeReply(draft)
			if err != nil {
				return err
			}

			if dryRun {
				_, err = cmd.OutOrStdout().Write(mime)
				return err
			}

			selectedAccount, err := resolveSelectedAccount(configPath, account, draft.Account)
			if err != nil {
				return err
			}

			drv, err := driverFactoryFunc(selectedAccount)
			if err != nil {
				return err
			}

			if err := drv.SendRaw(cmd.Context(), mime); err != nil {
				return err
			}

			result := schema.SendResult{
				OK:       true,
				Provider: selectedAccount.Driver,
				Account:  selectedAccount.Name,
			}
			return writeJSON(cmd.OutOrStdout(), &result)
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "config file path")
	cmd.Flags().StringVar(&account, "account", "", "account name override")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print MIME instead of sending it")
	return cmd
}

func resolveSelectedAccount(configPath, cliAccount, draftAccount string) (config.AccountConfig, error) {
	if configPath == "" {
		configPath = config.DefaultPath()
	}

	cfg, err := loadConfigFunc(configPath)
	if err != nil {
		return config.AccountConfig{}, err
	}

	target := cliAccount
	if target == "" {
		target = draftAccount
	}

	return cfg.ResolveAccount(target)
}
