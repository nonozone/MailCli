package driver

import (
	"fmt"
	"strings"

	"github.com/nonozone/MailCli/internal/config"
)

func NewFromAccount(account config.AccountConfig) (Driver, error) {
	switch strings.ToLower(strings.TrimSpace(account.Driver)) {
	case "imap":
		return newIMAPDriver(account)
	case "dir":
		return newDirDriver(account)
	case "stub":
		return newStubDriver(account)
	default:
		return nil, fmt.Errorf("driver not implemented: %s", account.Driver)
	}
}
