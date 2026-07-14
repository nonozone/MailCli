package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	mailindex "github.com/nonozone/MailCli/internal/index"
	mailtriage "github.com/nonozone/MailCli/internal/triage"
	"github.com/nonozone/MailCli/pkg/parser"
	"github.com/nonozone/MailCli/pkg/schema"
)

func newTriageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "triage",
		Short: "Build deterministic triage evidence and validate optional enrichment",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newTriageMessageCmd())
	cmd.AddCommand(newTriageThreadCmd())
	return cmd
}

func newTriageMessageCmd() *cobra.Command {
	var enrichmentPath string
	cmd := &cobra.Command{
		Use:   "message [file|-]",
		Short: "Build deterministic triage evidence for one message",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] == "-" && enrichmentPath == "-" {
				return fmt.Errorf("message input and enrichment cannot both use stdin")
			}
			raw, err := readInput(cmd, args[0])
			if err != nil {
				return err
			}
			message, err := parser.Parse(raw)
			if err != nil {
				return err
			}

			record := mailtriage.FromMessage(*message)
			if err := mergeTriageEnrichment(cmd, &record, enrichmentPath); err != nil {
				return err
			}
			return writeJSON(cmd.OutOrStdout(), record)
		},
	}
	cmd.Flags().StringVar(&enrichmentPath, "enrichment", "", "optional external enrichment JSON file or - for stdin")
	return cmd
}

func newTriageThreadCmd() *cobra.Command {
	var (
		indexPath      string
		account        string
		mailbox        string
		enrichmentPath string
	)
	cmd := &cobra.Command{
		Use:   "thread [thread_id]",
		Short: "Build deterministic triage evidence for a complete local thread",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store := mailindex.NewFileStore(indexPath)
			messages, err := store.ThreadMessages(mailindex.ThreadMessageQuery{
				ThreadID: args[0],
				Account:  account,
				Mailbox:  mailbox,
				Limit:    0,
			})
			if err != nil {
				return err
			}
			if len(messages) == 0 {
				return fmt.Errorf("thread %q was not found", args[0])
			}

			record, err := mailtriage.FromThread(args[0], messages)
			if err != nil {
				return err
			}
			if err := mergeTriageEnrichment(cmd, &record, enrichmentPath); err != nil {
				return err
			}
			return writeJSON(cmd.OutOrStdout(), record)
		},
	}
	cmd.Flags().StringVar(&indexPath, "index", "", "local index file path")
	cmd.Flags().StringVar(&account, "account", "", "filter local thread messages by account")
	cmd.Flags().StringVar(&mailbox, "mailbox", "", "filter local thread messages by mailbox")
	cmd.Flags().StringVar(&enrichmentPath, "enrichment", "", "optional external enrichment JSON file or - for stdin")
	return cmd
}

func mergeTriageEnrichment(cmd *cobra.Command, record *schema.TriageRecord, path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	raw, err := readInput(cmd, path)
	if err != nil {
		return err
	}
	enrichment, err := decodeTriageEnrichment(raw)
	if err != nil {
		return err
	}
	return mailtriage.ApplyEnrichment(record, enrichment)
}

func decodeTriageEnrichment(raw []byte) (schema.TriageEnrichment, error) {
	var enrichment schema.TriageEnrichment
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&enrichment); err != nil {
		return schema.TriageEnrichment{}, fmt.Errorf("invalid triage enrichment JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return schema.TriageEnrichment{}, fmt.Errorf("invalid triage enrichment JSON: expected one object")
		}
		return schema.TriageEnrichment{}, fmt.Errorf("invalid triage enrichment JSON: %w", err)
	}
	return enrichment, nil
}
