package main

import (
	"errors"
	"log"
	"os"

	"github.com/nonozone/MailCli/cmd"
)

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		if errors.Is(err, cmd.ErrSendFailureForExit()) {
			os.Exit(1)
		}
		log.Fatal(err)
	}
}
