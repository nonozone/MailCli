package cmd

import (
	"github.com/nonozone/MailCli/internal/config"
	"github.com/nonozone/MailCli/pkg/driver"
)

var (
	loadConfigFunc = config.Load
	driverFactoryFunc = driver.NewFromAccount
)
