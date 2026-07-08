package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/nonozone/MailCli/internal/config"
	opstore "github.com/nonozone/MailCli/internal/operations"
	"github.com/nonozone/MailCli/pkg/composer"
	"github.com/nonozone/MailCli/pkg/driver"
	"github.com/nonozone/MailCli/pkg/parser"
	"github.com/nonozone/MailCli/pkg/schema"
	"github.com/spf13/cobra"
)

var (
	errSendFailure       = errors.New("outbound command failed")
	errIntentAlreadySent = errors.New("intent already sent")
)

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

			var (
				selectedAccount config.AccountConfig
				drv             driver.Driver
			)
			if !dryRun {
				selectedAccount, err = resolveSelectedAccount(configPath, account, draft.Account)
				if err != nil {
					return writeSendFailure(cmd, providerName, accountName, err)
				}
				accountName = selectedAccount.Name
				providerName = selectedAccount.Driver
				applyDefaultFromAddress(selectedAccount, &draft.From)

				drv, err = driverFactoryFunc(selectedAccount)
				if err != nil {
					return writeSendFailure(cmd, providerName, accountName, err)
				}

				if err := validateEnvelopeAddressing(draft.From, draft.To, draft.Cc, draft.Bcc); err != nil {
					return writeSendFailure(cmd, providerName, accountName, err)
				}
			}

			mime, messageID, err := composer.ComposeDraft(draft)
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

			if err := drv.SendRaw(cmd.Context(), mime); err != nil {
				return writeSendFailure(cmd, providerName, accountName, err)
			}

			result := schema.SendResult{
				OK:        true,
				MessageID: messageID,
				Provider:  providerName,
				Account:   accountName,
			}
			return writeJSON(cmd.OutOrStdout(), &result)
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "config file path")
	cmd.Flags().StringVar(&account, "account", "", "account name override")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print MIME instead of sending it")
	cmd.AddCommand(newSendPrepareCmd())
	cmd.AddCommand(newSendConfirmCmd())
	return cmd
}

func newSendPrepareCmd() *cobra.Command {
	var (
		configPath     string
		account        string
		operationsPath string
	)

	cmd := &cobra.Command{
		Use:   "prepare [file|-]",
		Short: "Prepare a send intent without sending mail",
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

			selectedAccount, err := resolveSelectedAccount(configPath, account, draft.Account)
			if err != nil {
				return err
			}
			applyDefaultFromAddress(selectedAccount, &draft.From)
			if err := validateEnvelopeAddressing(draft.From, draft.To, draft.Cc, draft.Bcc); err != nil {
				return err
			}

			_, messageID, err := composer.ComposeDraft(draft)
			if err != nil {
				return err
			}
			if draft.Headers == nil {
				draft.Headers = map[string]string{}
			}
			draft.Headers["Message-ID"] = messageID

			store := opstore.NewStore(operationsPath)
			summary := summarizeDraft(draft)
			intent := opstore.SendIntent{
				ID:        opstore.NewID("intent"),
				Operation: "send",
				Account:   selectedAccount.Name,
				MessageID: messageID,
				CreatedAt: opstore.Now(),
				Summary:   summary,
				Draft:     draft,
			}
			if err := store.SaveSendIntent(intent); err != nil {
				return err
			}

			return writeJSON(cmd.OutOrStdout(), schema.OperationIntentResult{
				Status:         "prepared",
				IntentID:       intent.ID,
				Operation:      intent.Operation,
				Account:        intent.Account,
				MessageID:      intent.MessageID,
				OperationsPath: store.Path(),
				ConfirmCommand: buildSendConfirmCommand(configPath, store.Path(), intent.ID),
				Summary:        summary,
			})
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "config file path")
	cmd.Flags().StringVar(&account, "account", "", "account name override")
	cmd.Flags().StringVar(&operationsPath, "operations", "", "operations log path")
	return cmd
}

func newSendConfirmCmd() *cobra.Command {
	var (
		configPath     string
		account        string
		operationsPath string
	)

	cmd := &cobra.Command{
		Use:   "confirm [intent-id]",
		Short: "Send a previously prepared send intent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store := opstore.NewStore(operationsPath)
			intent, err := store.LoadSendIntent(args[0])
			if err != nil {
				return err
			}
			if intent.Operation != "send" {
				return fmt.Errorf("unsupported intent operation: %s", intent.Operation)
			}

			selectedAccount, err := resolveSelectedAccount(configPath, account, intent.Account)
			if err != nil {
				return recordAndWriteSendFailure(cmd, store, intent, "", intent.Account, err)
			}
			if strings.TrimSpace(account) != "" && selectedAccount.Name != intent.Account {
				return fmt.Errorf("intent %s belongs to account %s, not %s", intent.ID, intent.Account, selectedAccount.Name)
			}
			if sentEntry, err := store.SentEntryForIntent(intent.ID); err == nil {
				return recordAndWriteSendFailure(cmd, store, intent, selectedAccount.Driver, selectedAccount.Name, fmt.Errorf("%w: %s was sent as %s", errIntentAlreadySent, intent.ID, sentEntry.ID))
			} else if !errors.Is(err, opstore.ErrNotFound) {
				return err
			}

			if err := validateEnvelopeAddressing(intent.Draft.From, intent.Draft.To, intent.Draft.Cc, intent.Draft.Bcc); err != nil {
				return recordAndWriteSendFailure(cmd, store, intent, selectedAccount.Driver, selectedAccount.Name, err)
			}
			mime, _, err := composer.ComposeDraft(intent.Draft)
			if err != nil {
				return recordAndWriteSendFailure(cmd, store, intent, selectedAccount.Driver, selectedAccount.Name, err)
			}

			drv, err := driverFactoryFunc(selectedAccount)
			if err != nil {
				return recordAndWriteSendFailure(cmd, store, intent, selectedAccount.Driver, selectedAccount.Name, err)
			}
			if err := drv.SendRaw(cmd.Context(), mime); err != nil {
				return recordAndWriteSendFailure(cmd, store, intent, selectedAccount.Driver, selectedAccount.Name, err)
			}

			operationID := opstore.NewID("op")
			summary := intent.Summary
			if err := store.Append(schema.OperationLogEntry{
				ID:        operationID,
				IntentID:  intent.ID,
				Operation: "send",
				Status:    "sent",
				Account:   selectedAccount.Name,
				MessageID: intent.MessageID,
				CreatedAt: opstore.Now(),
				Summary:   &summary,
			}); err != nil {
				return err
			}

			return writeJSON(cmd.OutOrStdout(), &schema.SendResult{
				OK:          true,
				MessageID:   intent.MessageID,
				Provider:    selectedAccount.Driver,
				Account:     selectedAccount.Name,
				IntentID:    intent.ID,
				OperationID: operationID,
			})
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "config file path")
	cmd.Flags().StringVar(&account, "account", "", "account name override")
	cmd.Flags().StringVar(&operationsPath, "operations", "", "operations log path")
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
				applyDefaultFromAddress(selectedAccount, &draft.From)

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

			if !dryRun {
				if err := validateEnvelopeAddressing(draft.From, draft.To, draft.Cc, draft.Bcc); err != nil {
					return writeSendFailure(cmd, providerName, accountName, err)
				}
			}

			mime, messageID, err := composer.ComposeReply(draft)
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
				OK:        true,
				MessageID: messageID,
				Provider:  providerName,
				Account:   accountName,
			}
			return writeJSON(cmd.OutOrStdout(), &result)
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "config file path")
	cmd.Flags().StringVar(&account, "account", "", "account name override")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print MIME instead of sending it")
	cmd.AddCommand(newReplyPrepareCmd())
	cmd.AddCommand(newReplyConfirmCmd())
	return cmd
}

func newReplyPrepareCmd() *cobra.Command {
	var (
		configPath     string
		account        string
		operationsPath string
	)

	cmd := &cobra.Command{
		Use:   "prepare [file|-]",
		Short: "Prepare a reply intent without sending mail",
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

			selectedAccount, err := resolveSelectedAccount(configPath, account, draft.Account)
			if err != nil {
				return err
			}
			applyDefaultFromAddress(selectedAccount, &draft.From)

			if strings.TrimSpace(draft.ReplyToID) != "" {
				drv, err := driverFactoryFunc(selectedAccount)
				if err != nil {
					return err
				}
				if err := enrichReplyDraft(cmd.Context(), drv, &draft); err != nil {
					return err
				}
			}
			if err := validateEnvelopeAddressing(draft.From, draft.To, draft.Cc, draft.Bcc); err != nil {
				return err
			}

			_, messageID, err := composer.ComposeReply(draft)
			if err != nil {
				return err
			}
			if draft.Headers == nil {
				draft.Headers = map[string]string{}
			}
			draft.Headers["Message-ID"] = messageID

			store := opstore.NewStore(operationsPath)
			summary := summarizeReplyDraft(draft)
			intent := opstore.ReplyIntent{
				ID:        opstore.NewID("intent"),
				Operation: "reply",
				Account:   selectedAccount.Name,
				MessageID: messageID,
				CreatedAt: opstore.Now(),
				Summary:   summary,
				Draft:     draft,
			}
			if err := store.SaveReplyIntent(intent); err != nil {
				return err
			}

			return writeJSON(cmd.OutOrStdout(), schema.OperationIntentResult{
				Status:         "prepared",
				IntentID:       intent.ID,
				Operation:      intent.Operation,
				Account:        intent.Account,
				MessageID:      intent.MessageID,
				OperationsPath: store.Path(),
				ConfirmCommand: buildReplyConfirmCommand(configPath, store.Path(), intent.ID),
				Summary:        summary,
			})
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "config file path")
	cmd.Flags().StringVar(&account, "account", "", "account name override")
	cmd.Flags().StringVar(&operationsPath, "operations", "", "operations log path")
	return cmd
}

func newReplyConfirmCmd() *cobra.Command {
	var (
		configPath     string
		account        string
		operationsPath string
	)

	cmd := &cobra.Command{
		Use:   "confirm [intent-id]",
		Short: "Send a previously prepared reply intent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store := opstore.NewStore(operationsPath)
			intent, err := store.LoadReplyIntent(args[0])
			if err != nil {
				return err
			}
			if intent.Operation != "reply" {
				return fmt.Errorf("unsupported intent operation: %s", intent.Operation)
			}

			selectedAccount, err := resolveSelectedAccount(configPath, account, intent.Account)
			if err != nil {
				return recordAndWriteReplyFailure(cmd, store, intent, "", intent.Account, err)
			}
			if strings.TrimSpace(account) != "" && selectedAccount.Name != intent.Account {
				return fmt.Errorf("intent %s belongs to account %s, not %s", intent.ID, intent.Account, selectedAccount.Name)
			}
			if sentEntry, err := store.SentEntryForOperation(intent.ID, "reply"); err == nil {
				return recordAndWriteReplyFailure(cmd, store, intent, selectedAccount.Driver, selectedAccount.Name, fmt.Errorf("%w: %s was sent as %s", errIntentAlreadySent, intent.ID, sentEntry.ID))
			} else if !errors.Is(err, opstore.ErrNotFound) {
				return err
			}

			if err := validateEnvelopeAddressing(intent.Draft.From, intent.Draft.To, intent.Draft.Cc, intent.Draft.Bcc); err != nil {
				return recordAndWriteReplyFailure(cmd, store, intent, selectedAccount.Driver, selectedAccount.Name, err)
			}
			mime, _, err := composer.ComposeReply(intent.Draft)
			if err != nil {
				return recordAndWriteReplyFailure(cmd, store, intent, selectedAccount.Driver, selectedAccount.Name, err)
			}

			drv, err := driverFactoryFunc(selectedAccount)
			if err != nil {
				return recordAndWriteReplyFailure(cmd, store, intent, selectedAccount.Driver, selectedAccount.Name, err)
			}
			if err := drv.SendRaw(cmd.Context(), mime); err != nil {
				return recordAndWriteReplyFailure(cmd, store, intent, selectedAccount.Driver, selectedAccount.Name, err)
			}

			operationID := opstore.NewID("op")
			summary := intent.Summary
			if err := store.Append(schema.OperationLogEntry{
				ID:        operationID,
				IntentID:  intent.ID,
				Operation: "reply",
				Status:    "sent",
				Account:   selectedAccount.Name,
				MessageID: intent.MessageID,
				CreatedAt: opstore.Now(),
				Summary:   &summary,
			}); err != nil {
				return err
			}

			return writeJSON(cmd.OutOrStdout(), &schema.SendResult{
				OK:          true,
				MessageID:   intent.MessageID,
				Provider:    selectedAccount.Driver,
				Account:     selectedAccount.Name,
				IntentID:    intent.ID,
				OperationID: operationID,
			})
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "config file path")
	cmd.Flags().StringVar(&account, "account", "", "account name override")
	cmd.Flags().StringVar(&operationsPath, "operations", "", "operations log path")
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

	if len(draft.To) == 0 {
		draft.To = deriveReplyRecipients(msg, draft.From)
	}

	return nil
}

func applyDefaultFromAddress(account config.AccountConfig, current **schema.Address) {
	if current != nil && *current != nil && strings.TrimSpace((*current).Address) != "" {
		return
	}

	address := firstNonEmpty(account.SMTPUsername, account.Username)
	if address == "" {
		return
	}

	if current != nil && *current != nil {
		(*current).Address = address
		return
	}

	if current != nil {
		*current = &schema.Address{Address: address}
	}
}

func deriveReplyRecipients(msg *schema.StandardMessage, sender *schema.Address) []schema.Address {
	if msg == nil {
		return nil
	}

	senderAddress := ""
	if sender != nil {
		senderAddress = strings.TrimSpace(sender.Address)
	}

	if msg.Meta.From != nil && !sameAddress(msg.Meta.From.Address, senderAddress) {
		return []schema.Address{*msg.Meta.From}
	}

	recipients := make([]schema.Address, 0, len(msg.Meta.To))
	for _, addr := range msg.Meta.To {
		if sameAddress(addr.Address, senderAddress) {
			continue
		}
		if strings.TrimSpace(addr.Address) == "" && strings.TrimSpace(addr.Name) == "" {
			continue
		}
		recipients = append(recipients, addr)
	}
	if len(recipients) > 0 {
		return recipients
	}

	if msg.Meta.From != nil && sameAddress(msg.Meta.From.Address, senderAddress) {
		return nil
	}

	if msg.Meta.From != nil && (strings.TrimSpace(msg.Meta.From.Address) != "" || strings.TrimSpace(msg.Meta.From.Name) != "") {
		return []schema.Address{*msg.Meta.From}
	}

	return nil
}

func sameAddress(left, right string) bool {
	left = strings.ToLower(strings.TrimSpace(left))
	right = strings.ToLower(strings.TrimSpace(right))
	if left == "" || right == "" {
		return false
	}
	return left == right
}

func validateEnvelopeAddressing(from *schema.Address, to, cc, bcc []schema.Address) error {
	if from == nil || strings.TrimSpace(from.Address) == "" {
		return errors.New("missing from header")
	}
	if countAddressRecipients(to)+countAddressRecipients(cc)+countAddressRecipients(bcc) == 0 {
		return errors.New("missing recipients")
	}
	return nil
}

func countAddressRecipients(values []schema.Address) int {
	count := 0
	for _, addr := range values {
		if strings.TrimSpace(addr.Address) != "" {
			count++
		}
	}
	return count
}

func summarizeDraft(draft schema.DraftMessage) schema.OperationSummary {
	return schema.OperationSummary{
		Subject:         strings.TrimSpace(draft.Subject),
		From:            draft.From,
		To:              append([]schema.Address{}, draft.To...),
		Cc:              append([]schema.Address{}, draft.Cc...),
		BccCount:        countAddressRecipients(draft.Bcc),
		AttachmentCount: len(draft.Attachments),
	}
}

func summarizeReplyDraft(draft schema.ReplyDraft) schema.OperationSummary {
	return schema.OperationSummary{
		Subject:         strings.TrimSpace(draft.Subject),
		From:            draft.From,
		To:              append([]schema.Address{}, draft.To...),
		Cc:              append([]schema.Address{}, draft.Cc...),
		BccCount:        countAddressRecipients(draft.Bcc),
		AttachmentCount: len(draft.Attachments),
	}
}

func buildSendConfirmCommand(configPath, operationsPath, intentID string) string {
	return buildOperationConfirmCommand("send", configPath, operationsPath, intentID)
}

func buildReplyConfirmCommand(configPath, operationsPath, intentID string) string {
	return buildOperationConfirmCommand("reply", configPath, operationsPath, intentID)
}

func buildOperationConfirmCommand(command, configPath, operationsPath, intentID string) string {
	args := []string{"mailcli", command, "confirm"}
	if strings.TrimSpace(configPath) != "" {
		args = append(args, "--config", configPath)
	}
	if strings.TrimSpace(operationsPath) != "" {
		args = append(args, "--operations", operationsPath)
	}
	args = append(args, intentID)

	parts := make([]string, 0, len(args))
	for _, arg := range args {
		parts = append(parts, shellQuoteArg(arg))
	}
	return strings.Join(parts, " ")
}

func shellQuoteArg(arg string) string {
	if arg == "" {
		return "''"
	}
	if isShellSafeArg(arg) {
		return arg
	}
	return "'" + strings.ReplaceAll(arg, "'", "'\"'\"'") + "'"
}

func isShellSafeArg(arg string) bool {
	for _, r := range arg {
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		if strings.ContainsRune("@%_+=:,./-", r) {
			continue
		}
		return false
	}
	return true
}

func recordSendFailure(store *opstore.Store, intent opstore.SendIntent, account string, sendErr *schema.SendError) string {
	summary := intent.Summary
	operationID := opstore.NewID("op")
	_ = store.Append(schema.OperationLogEntry{
		ID:        operationID,
		IntentID:  intent.ID,
		Operation: "send",
		Status:    "failed",
		Account:   account,
		MessageID: intent.MessageID,
		CreatedAt: opstore.Now(),
		Summary:   &summary,
		Error:     sendErr,
	})
	return operationID
}

func recordAndWriteSendFailure(cmd *cobra.Command, store *opstore.Store, intent opstore.SendIntent, provider, account string, err error) error {
	operationID := recordSendFailure(store, intent, account, mapSendError(err))
	return writeSendFailureForIntent(cmd, provider, account, intent.ID, operationID, err)
}

func recordReplyFailure(store *opstore.Store, intent opstore.ReplyIntent, account string, sendErr *schema.SendError) string {
	summary := intent.Summary
	operationID := opstore.NewID("op")
	_ = store.Append(schema.OperationLogEntry{
		ID:        operationID,
		IntentID:  intent.ID,
		Operation: "reply",
		Status:    "failed",
		Account:   account,
		MessageID: intent.MessageID,
		CreatedAt: opstore.Now(),
		Summary:   &summary,
		Error:     sendErr,
	})
	return operationID
}

func recordAndWriteReplyFailure(cmd *cobra.Command, store *opstore.Store, intent opstore.ReplyIntent, provider, account string, err error) error {
	operationID := recordReplyFailure(store, intent, account, mapSendError(err))
	return writeSendFailureForIntent(cmd, provider, account, intent.ID, operationID, err)
}

func writeSendFailure(cmd *cobra.Command, provider, account string, err error) error {
	return writeSendFailureForIntent(cmd, provider, account, "", "", err)
}

func writeSendFailureForIntent(cmd *cobra.Command, provider, account, intentID, operationID string, err error) error {
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	if root := cmd.Root(); root != nil {
		root.SilenceUsage = true
		root.SilenceErrors = true
	}
	result := schema.SendResult{
		OK:          false,
		Provider:    provider,
		Account:     account,
		IntentID:    intentID,
		OperationID: operationID,
		Error:       mapSendError(err),
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
	case errors.Is(err, errIntentAlreadySent):
		code = "intent_already_sent"
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
	case errors.Is(err, driver.ErrAuthFailed) ||
		strings.Contains(lower, "auth") ||
		strings.Contains(lower, "authentication") ||
		strings.Contains(lower, "credentials invalid") ||
		strings.Contains(lower, "535"):
		code = "auth_failed"
	case strings.Contains(lower, "550") || strings.Contains(lower, "553") || strings.Contains(lower, "user unknown") || strings.Contains(lower, "no such user"):
		code = "recipient_rejected"
	case strings.Contains(lower, "421") || strings.Contains(lower, "service not available"):
		code = "service_unavailable"
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
