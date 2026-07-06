package main

import (
	"fmt"
	"os"

	"github.com/nonozone/MailCli/examples/internal/agent"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go run ./examples/go/reply_dry_run -- <reply.json>")
		os.Exit(1)
	}

	output, err := agent.RunCommand([]string{"mailcli", "reply", "--dry-run", os.Args[1]}, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Print(output)
}
