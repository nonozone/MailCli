package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	mailindex "github.com/nonozone/MailCli/internal/index"
)

func newExportCmd() *cobra.Command {
	var (
		indexPath string
		account   string
		mailbox   string
		since     string
		before    string
		format    string
		output    string
		limit     int
	)

	cmd := &cobra.Command{
		Use:   "export [query]",
		Short: "Export the local message index to JSONL, JSON, or CSV",
		Long: `Export indexed messages to stdout or a file.

Formats:
  jsonl  One JSON object per line (default, best for streaming/AI pipelines)
  json   A single JSON array
  csv    CSV with columns: account,mailbox,id,thread_id,from,subject,date,category,snippet

Filtering follows the same query/account/mailbox/since/before rules as 'search'.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := ""
			if len(args) > 0 {
				query = args[0]
			}

			store := mailindex.NewFileStore(indexPath)
			results, err := store.SearchMessages(mailindex.SearchQuery{
				Query:   query,
				Account: account,
				Mailbox: mailbox,
				Limit:   limit,
				Since:   since,
				Before:  before,
			})
			if err != nil {
				return err
			}

			// Select output writer.
			var out io.Writer = cmd.OutOrStdout()
			if strings.TrimSpace(output) != "" {
				f, err := os.Create(output)
				if err != nil {
					return fmt.Errorf("open output file: %w", err)
				}
				defer f.Close()
				out = f
			}

			switch strings.ToLower(strings.TrimSpace(format)) {
			case "", "jsonl":
				return writeJSONL(out, results)
			case "json":
				return writeJSON(out, results)
			case "csv":
				return writeCSV(out, results)
			default:
				return fmt.Errorf("unsupported format: %s (use jsonl, json, csv)", format)
			}
		},
	}

	cmd.Flags().StringVar(&indexPath, "index", "", "local index file path")
	cmd.Flags().StringVar(&account, "account", "", "filter by account")
	cmd.Flags().StringVar(&mailbox, "mailbox", "", "filter by mailbox")
	cmd.Flags().StringVar(&since, "since", "", "only export messages on or after this RFC3339 timestamp")
	cmd.Flags().StringVar(&before, "before", "", "only export messages before this RFC3339 timestamp")
	cmd.Flags().StringVar(&format, "format", "jsonl", "output format: jsonl, json, csv")
	cmd.Flags().StringVar(&output, "output", "", "write to file instead of stdout")
	cmd.Flags().IntVar(&limit, "limit", 0, "maximum number of messages to export (0 means no limit)")
	return cmd
}

// writeJSONL writes one compact JSON object per line — ideal for AI pipelines.
func writeJSONL(out io.Writer, items []mailindex.IndexedMessage) error {
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	for _, item := range items {
		if err := enc.Encode(item); err != nil {
			return err
		}
	}
	return nil
}

// writeCSV writes messages as a flat CSV with core metadata columns.
func writeCSV(out io.Writer, items []mailindex.IndexedMessage) error {
	w := csv.NewWriter(out)
	if err := w.Write([]string{
		"account", "mailbox", "id", "thread_id",
		"from", "subject", "date", "category", "snippet",
	}); err != nil {
		return err
	}

	for _, item := range items {
		from := ""
		if item.Message.Meta.From != nil {
			from = formatAddress(item.Message.Meta.From)
		}
		threadID := item.ThreadID
		if threadID == "" {
			threadID = item.ID
		}
		if err := w.Write([]string{
			item.Account,
			item.Mailbox,
			item.ID,
			threadID,
			from,
			item.Message.Meta.Subject,
			item.Message.Meta.Date,
			item.Message.Content.Category,
			item.Message.Content.Snippet,
		}); err != nil {
			return err
		}
	}

	w.Flush()
	return w.Error()
}
