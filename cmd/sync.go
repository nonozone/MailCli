package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	mailindex "github.com/nonozone/MailCli/internal/index"
	"github.com/nonozone/MailCli/pkg/parser"
	"github.com/nonozone/MailCli/pkg/schema"
)

type syncResult struct {
	Account      string `json:"account,omitempty"`
	Mailbox      string `json:"mailbox,omitempty"`
	IndexedCount int    `json:"indexed_count"`
	SkippedCount int    `json:"skipped_count"`
	IndexPath    string `json:"index_path,omitempty"`
}

func newSyncCmd() *cobra.Command {
	var (
		configPath string
		account    string
		mailbox    string
		indexPath  string
		format     string
		limit      int
		refresh    bool
	)

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Fetch recent messages and store them in the local index",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			selectedAccount, err := resolveSelectedAccount(configPath, account, "")
			if err != nil {
				return err
			}

			drv, err := driverFactoryFunc(selectedAccount)
			if err != nil {
				return err
			}

			queryMailbox := strings.TrimSpace(mailbox)
			if queryMailbox == "" {
				queryMailbox = selectedAccount.Mailbox
			}
			if strings.TrimSpace(queryMailbox) == "" {
				queryMailbox = "INBOX"
			}

			items, err := drv.List(cmd.Context(), schema.SearchQuery{
				Mailbox: queryMailbox,
				Limit:   limit,
			})
			if err != nil {
				return err
			}

			store := mailindex.NewFileStore(indexPath)
			indexedCount := 0
			skippedCount := 0
			for _, item := range items {
				if !refresh {
					has, err := store.Has(selectedAccount.Name, item.ID)
					if err != nil {
						return err
					}
					if has {
						skippedCount++
						continue
					}
				}

				raw, err := drv.FetchRaw(cmd.Context(), item.ID)
				if err != nil {
					return err
				}

				msg, err := parser.Parse(raw)
				if err != nil {
					return err
				}

				if err := store.Upsert(mailindex.IndexedMessage{
					Account: selectedAccount.Name,
					Mailbox: queryMailbox,
					ID:      item.ID,
					Message: *msg,
				}); err != nil {
					return err
				}
				indexedCount++
			}

			return writeSyncResult(cmd.OutOrStdout(), syncResult{
				Account:      selectedAccount.Name,
				Mailbox:      queryMailbox,
				IndexedCount: indexedCount,
				SkippedCount: skippedCount,
				IndexPath:    store.Path(),
			}, format)
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "config file path")
	cmd.Flags().StringVar(&account, "account", "", "account name override")
	cmd.Flags().StringVar(&mailbox, "mailbox", "", "mailbox override")
	cmd.Flags().StringVar(&indexPath, "index", "", "local index file path")
	cmd.Flags().IntVar(&limit, "limit", 10, "maximum number of messages to sync")
	cmd.Flags().BoolVar(&refresh, "refresh", false, "re-fetch and reindex messages even if they already exist locally")
	cmd.Flags().StringVar(&format, "format", "json", "output format: json, table")
	return cmd
}

func newSearchCmd() *cobra.Command {
	var (
		indexPath string
		account   string
		mailbox   string
		threadID  string
		format    string
		limit     int
		full      bool
	)

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search the local message index",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store := mailindex.NewFileStore(indexPath)
			query := mailindex.SearchQuery{
				Query:    args[0],
				Account:  account,
				Mailbox:  mailbox,
				ThreadID: threadID,
				Limit:    limit,
			}

			if full {
				items, err := store.SearchMessages(query)
				if err != nil {
					return err
				}
				return writeFullSearchResults(cmd.OutOrStdout(), items, format)
			}

			results, err := store.Search(query)
			if err != nil {
				return err
			}
			return writeSearchResults(cmd.OutOrStdout(), results, format)
		},
	}

	cmd.Flags().StringVar(&indexPath, "index", "", "local index file path")
	cmd.Flags().StringVar(&account, "account", "", "filter local results by account")
	cmd.Flags().StringVar(&mailbox, "mailbox", "", "filter local results by mailbox")
	cmd.Flags().StringVar(&threadID, "thread", "", "filter local results by thread id")
	cmd.Flags().IntVar(&limit, "limit", 10, "maximum number of search results")
	cmd.Flags().BoolVar(&full, "full", false, "return full indexed messages instead of compact search results")
	cmd.Flags().StringVar(&format, "format", "json", "output format: json, table")
	return cmd
}

func writeSyncResult(out io.Writer, result syncResult, format string) error {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "json":
		return writeJSON(out, result)
	case "table":
		table := tablewriter.NewWriter(out)
		table.SetHeader([]string{"Field", "Value"})
		table.AppendBulk([][]string{
			{"Account", result.Account},
			{"Mailbox", result.Mailbox},
			{"IndexedCount", fmt.Sprintf("%d", result.IndexedCount)},
			{"SkippedCount", fmt.Sprintf("%d", result.SkippedCount)},
			{"IndexPath", result.IndexPath},
		})
		table.Render()
		return nil
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func writeSearchResults(out io.Writer, results []mailindex.SearchResult, format string) error {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "json":
		return writeJSON(out, results)
	case "table":
		table := tablewriter.NewWriter(out)
		table.SetHeader([]string{"Account", "Mailbox", "ID", "From", "Subject", "Date"})
		rows := make([][]string, 0, len(results))
		for _, item := range results {
			rows = append(rows, []string{
				item.Account,
				item.Mailbox,
				item.ID,
				item.From,
				item.Subject,
				item.Date,
			})
		}
		table.AppendBulk(rows)
		table.Render()
		return nil
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func writeFullSearchResults(out io.Writer, results []mailindex.IndexedMessage, format string) error {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "json":
		return writeJSON(out, results)
	case "table":
		return fmt.Errorf("full search results do not support table format")
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}
