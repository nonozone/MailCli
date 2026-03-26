package main

import (
	"errors"
	"log"
	"os"

	"github.com/yourname/mailcli/cmd"
)

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		if errors.Is(err, cmd.ErrSendFailureForExit()) {
			os.Exit(1)
		}
		log.Fatal(err)
	}
}
