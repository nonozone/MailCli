package cmd

import (
	"github.com/yourname/mailcli/internal/config"
	"github.com/yourname/mailcli/pkg/driver"
)

var (
	loadConfigFunc = config.Load
	driverFactoryFunc = driver.NewFromAccount
)
