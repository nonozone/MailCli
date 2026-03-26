package cmd

import (
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/nonozone/MailCli/pkg/parser"
)

func newParseCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "parse [file|-]",
		Short: "Parse an email into normalized JSON",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, err := readInput(cmd, args[0])
			if err != nil {
				return err
			}

			msg, err := parser.Parse(raw)
			if err != nil {
				return err
			}

			return writeMessage(cmd.OutOrStdout(), msg, format)
		},
	}

	cmd.Flags().StringVar(&format, "format", "json", "output format: json, yaml, table")
	return cmd
}

func readInput(cmd *cobra.Command, arg string) ([]byte, error) {
	if arg == "-" {
		return io.ReadAll(cmd.InOrStdin())
	}

	return os.ReadFile(arg)
}
