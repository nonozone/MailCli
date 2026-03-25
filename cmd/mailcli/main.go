package main

import (
	"log"

	"github.com/yourname/mailcli/cmd"
)

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		log.Fatal(err)
	}
}
