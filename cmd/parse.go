package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/yourname/mailcli/pkg/parser"
)

func newParseCmd() *cobra.Command {
	return &cobra.Command{
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

			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(msg); err != nil {
				return fmt.Errorf("encode json: %w", err)
			}

			return nil
		},
	}
}

func readInput(cmd *cobra.Command, arg string) ([]byte, error) {
	if arg == "-" {
		return io.ReadAll(cmd.InOrStdin())
	}

	return os.ReadFile(arg)
}
