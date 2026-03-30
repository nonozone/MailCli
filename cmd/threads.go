package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	mailindex "github.com/nonozone/MailCli/internal/index"
)

func newThreadsCmd() *cobra.Command {
	var (
		indexPath string
		account   string
		mailbox   string
		category  string
		action    string
		hasCodes  bool
		format    string
		limit     int
		since     string
		before    string
	)

	cmd := &cobra.Command{
		Use:   "threads [query]",
		Short: "List local thread summaries from the indexed messages",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := ""
			if len(args) > 0 {
				query = args[0]
			}

			store := mailindex.NewFileStore(indexPath)
			results, err := store.Threads(mailindex.ThreadQuery{
				Query:    query,
				Account:  account,
				Mailbox:  mailbox,
				Category: category,
				Action:   action,
				HasCodes: hasCodes,
				Limit:    limit,
				Since:    since,
				Before:   before,
			})
			if err != nil {
				return err
			}

			return writeThreadResults(cmd.OutOrStdout(), results, format)
		},
	}

	cmd.Flags().StringVar(&indexPath, "index", "", "local index file path")
	cmd.Flags().StringVar(&account, "account", "", "filter local threads by account")
	cmd.Flags().StringVar(&mailbox, "mailbox", "", "filter local threads by mailbox")
	cmd.Flags().StringVar(&category, "category", "", "filter local threads by aggregated category")
	cmd.Flags().StringVar(&action, "action", "", "filter local threads by aggregated action type")
	cmd.Flags().BoolVar(&hasCodes, "has-codes", false, "filter local threads that include extracted codes")
	cmd.Flags().IntVar(&limit, "limit", 10, "maximum number of thread results")
	cmd.Flags().StringVar(&since, "since", "", "only return threads with messages on or after this RFC3339 timestamp")
	cmd.Flags().StringVar(&before, "before", "", "only return threads with messages before this RFC3339 timestamp")
	cmd.Flags().StringVar(&format, "format", "json", "output format: json, table")
	return cmd
}

func writeThreadResults(out io.Writer, results []mailindex.ThreadSummary, format string) error {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "json":
		return writeJSON(out, results)
	case "table":
		table := tablewriter.NewWriter(out)
		table.SetHeader([]string{"ThreadID", "Subject", "Count", "LatestDate", "LastFrom", "Preview", "Score"})
		rows := make([][]string, 0, len(results))
		for _, item := range results {
			rows = append(rows, []string{
				item.ThreadID,
				item.Subject,
				fmt.Sprintf("%d", item.MessageCount),
				item.LatestDate,
				item.LastMessageFrom,
				item.LastMessagePreview,
				fmt.Sprintf("%d", item.Score),
			})
		}
		table.AppendBulk(rows)
		table.Render()
		return nil
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}
