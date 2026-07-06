package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/nonozone/MailCli/examples/internal/agent"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go run ./examples/go/parse_email -- <email.eml>")
		os.Exit(1)
	}

	output, err := agent.RunCommand([]string{"mailcli", "parse", os.Args[1]}, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var message map[string]any
	if err := json.Unmarshal([]byte(output), &message); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	content := agent.MapValue(message, "content")
	fmt.Println(agent.StringValue(content, "body_md"))
}
