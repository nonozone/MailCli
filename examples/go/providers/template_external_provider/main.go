package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/nonozone/MailCli/examples/internal/agent"
)

func main() {
	var payload map[string]any
	if err := json.NewDecoder(os.Stdin).Decode(&payload); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := agent.WriteJSON(os.Stdout, agent.AnalyzeTemplatePayload(payload)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
