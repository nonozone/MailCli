package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/spf13/cobra"
	"github.com/nonozone/MailCli/internal/config"
	"github.com/nonozone/MailCli/pkg/composer"
	"github.com/nonozone/MailCli/pkg/driver"
	"github.com/nonozone/MailCli/pkg/parser"
	"github.com/nonozone/MailCli/pkg/schema"
)

var errSendFailure = errors.New("outbound command failed")

func ErrSendFailureForExit() error {
	return errSendFailure
}

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
			accountName := account
			providerName := ""

			raw, err := readInput(cmd, args[0])
			if err != nil {
				if dryRun {
					return err
				}
				return writeSendFailure(cmd, providerName, accountName, err)
			}

			var draft schema.DraftMessage
			if err := json.Unmarshal(raw, &draft); err != nil {
				if dryRun {
					return err
				}
				return writeSendFailure(cmd, providerName, accountName, err)
			}
			accountName = firstNonEmpty(accountName, draft.Account)

			mime, err := composer.ComposeDraft(draft)
			if err != nil {
				if dryRun {
					return err
				}
				return writeSendFailure(cmd, providerName, accountName, err)
			}

			if dryRun {
				_, err := cmd.OutOrStdout().Write(mime)
				return err
			}

			selectedAccount, err := resolveSelectedAccount(configPath, account, draft.Account)
			if err != nil {
				return writeSendFailure(cmd, providerName, accountName, err)
			}
			accountName = selectedAccount.Name
			providerName = selectedAccount.Driver

			drv, err := driverFactoryFunc(selectedAccount)
			if err != nil {
				return writeSendFailure(cmd, providerName, accountName, err)
			}

			if err := drv.SendRaw(cmd.Context(), mime); err != nil {
				return writeSendFailure(cmd, providerName, accountName, err)
			}

			result := schema.SendResult{
				OK:       true,
				Provider: providerName,
				Account:  accountName,
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
			accountName := account
			providerName := ""

			raw, err := readInput(cmd, args[0])
			if err != nil {
				if dryRun {
					return err
				}
				return writeSendFailure(cmd, providerName, accountName, err)
			}

			var draft schema.ReplyDraft
			if err := json.Unmarshal(raw, &draft); err != nil {
				if dryRun {
					return err
				}
				return writeSendFailure(cmd, providerName, accountName, err)
			}
			accountName = firstNonEmpty(accountName, draft.Account)

			var (
				selectedAccount config.AccountConfig
				drv             driver.Driver
			)
			if strings.TrimSpace(draft.ReplyToID) != "" || !dryRun {
				selectedAccount, err = resolveSelectedAccount(configPath, account, draft.Account)
				if err != nil {
					return writeSendFailure(cmd, providerName, accountName, err)
				}
				accountName = selectedAccount.Name
				providerName = selectedAccount.Driver

				drv, err = driverFactoryFunc(selectedAccount)
				if err != nil {
					return writeSendFailure(cmd, providerName, accountName, err)
				}
			}

			if err := enrichReplyDraft(cmd.Context(), drv, &draft); err != nil {
				if dryRun {
					return err
				}
				return writeSendFailure(cmd, providerName, accountName, err)
			}

			mime, err := composer.ComposeReply(draft)
			if err != nil {
				if dryRun {
					return err
				}
				return writeSendFailure(cmd, providerName, accountName, err)
			}

			if dryRun {
				_, err = cmd.OutOrStdout().Write(mime)
				return err
			}

			if err := drv.SendRaw(cmd.Context(), mime); err != nil {
				return writeSendFailure(cmd, providerName, accountName, err)
			}

			result := schema.SendResult{
				OK:       true,
				Provider: providerName,
				Account:  accountName,
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

func enrichReplyDraft(ctx context.Context, drv driver.Driver, draft *schema.ReplyDraft) error {
	if draft == nil || strings.TrimSpace(draft.ReplyToID) == "" {
		return nil
	}
	if strings.TrimSpace(draft.ReplyToMessageID) != "" && len(draft.References) > 0 && strings.TrimSpace(draft.Subject) != "" {
		return nil
	}

	raw, err := drv.FetchRaw(ctx, draft.ReplyToID)
	if err != nil {
		return err
	}

	msg, err := parser.Parse(raw)
	if err != nil {
		return err
	}

	if strings.TrimSpace(draft.ReplyToMessageID) == "" {
		draft.ReplyToMessageID = msg.Meta.MessageID
	}

	if len(draft.References) == 0 {
		draft.References = append(append([]string{}, msg.Meta.References...), msg.Meta.MessageID)
	}

	if strings.TrimSpace(draft.Subject) == "" {
		draft.Subject = msg.Meta.Subject
	}

	return nil
}

func writeSendFailure(cmd *cobra.Command, provider, account string, err error) error {
	result := schema.SendResult{
		OK:       false,
		Provider: provider,
		Account:  account,
		Error:    mapSendError(err),
	}
	if writeErr := writeJSON(cmd.OutOrStdout(), &result); writeErr != nil {
		return writeErr
	}
	return errSendFailure
}

func mapSendError(err error) *schema.SendError {
	if err == nil {
		return nil
	}

	message := err.Error()
	lower := strings.ToLower(message)

	code := "transport_failed"
	switch {
	case errors.Is(err, config.ErrAccountNotFound) || strings.Contains(lower, "account not found"):
		code = "account_not_found"
	case errors.Is(err, config.ErrNoAccountSelected) || strings.Contains(lower, "no account selected"):
		code = "account_not_selected"
	case errors.Is(err, driver.ErrMessageNotFound) || strings.Contains(lower, "message not found"):
		code = "message_not_found"
	case errors.Is(err, driver.ErrTransportNotConfigured) || strings.Contains(lower, "smtp settings not configured"):
		code = "transport_not_configured"
	case errors.Is(err, driver.ErrDriverConfigInvalid):
		code = "transport_failed"
	case strings.Contains(lower, "auth") || strings.Contains(lower, "authentication") || strings.Contains(lower, "credentials invalid") || strings.Contains(lower, "535"):
		code = "auth_failed"
	case strings.Contains(lower, "missing from header") ||
		strings.Contains(lower, "missing recipients") ||
		strings.Contains(lower, "invalid character") ||
		strings.Contains(lower, "unexpected end of json input"):
		code = "invalid_draft"
	}

	return &schema.SendError{
		Code:    code,
		Message: message,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
