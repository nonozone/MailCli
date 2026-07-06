package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nonozone/MailCli/internal/config"
	"github.com/nonozone/MailCli/pkg/schema"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage mailcli configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newConfigInitCmd())
	cmd.AddCommand(newConfigShowCmd())
	cmd.AddCommand(newConfigDoctorCmd())
	cmd.AddCommand(newConfigTestCmd())
	cmd.AddCommand(newConfigCapabilitiesCmd())
	return cmd
}

// config init ─────────────────────────────────────────────────────────────────

func newConfigInitCmd() *cobra.Command {
	var (
		configPath      string
		account         string
		driverName      string
		host            string
		port            int
		username        string
		passwordEnv     string
		tlsEnabled      bool
		mailbox         string
		smtpHost        string
		smtpPort        int
		smtpUsername    string
		smtpPasswordEnv string
		smtpTLSEnabled  bool
		dirPath         string
		force           bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a starter mailcli config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if configPath == "" {
				configPath = config.DefaultPath()
			}

			driverName = strings.ToLower(strings.TrimSpace(driverName))
			account = strings.TrimSpace(account)
			if account == "" {
				return fmt.Errorf("--account is required")
			}
			if driverName == "" {
				return fmt.Errorf("--driver is required")
			}

			if _, err := os.Stat(configPath); err == nil && !force {
				return fmt.Errorf("config already exists at %s; pass --force to overwrite", configPath)
			} else if err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("stat config: %w", err)
			}

			accountConfig, err := buildInitialAccountConfig(initialAccountOptions{
				account:         account,
				driverName:      driverName,
				host:            host,
				port:            port,
				username:        username,
				passwordEnv:     passwordEnv,
				tlsEnabled:      tlsEnabled,
				mailbox:         mailbox,
				smtpHost:        smtpHost,
				smtpPort:        smtpPort,
				smtpUsername:    smtpUsername,
				smtpPasswordEnv: smtpPasswordEnv,
				smtpTLSEnabled:  smtpTLSEnabled,
				dirPath:         dirPath,
			})
			if err != nil {
				return err
			}

			cfg := config.Config{
				CurrentAccount: account,
				Accounts:       []config.AccountConfig{accountConfig},
			}
			data, err := config.Marshal(cfg)
			if err != nil {
				return fmt.Errorf("marshal config: %w", err)
			}

			if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
				return fmt.Errorf("create config directory: %w", err)
			}
			if err := os.WriteFile(configPath, data, 0o600); err != nil {
				return fmt.Errorf("write config: %w", err)
			}
			if err := os.Chmod(configPath, 0o600); err != nil {
				return fmt.Errorf("set config permissions: %w", err)
			}

			return writeJSON(cmd.OutOrStdout(), schema.ConfigInitResult{
				Status:     "created",
				ConfigPath: configPath,
				Account:    accountConfig.Name,
				Driver:     accountConfig.Driver,
			})
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "config file path")
	cmd.Flags().StringVar(&account, "account", "", "account name")
	cmd.Flags().StringVar(&driverName, "driver", "", "driver type: imap, dir, stub")
	cmd.Flags().StringVar(&host, "host", "", "IMAP host")
	cmd.Flags().IntVar(&port, "port", 993, "IMAP port")
	cmd.Flags().StringVar(&username, "username", "", "IMAP username")
	cmd.Flags().StringVar(&passwordEnv, "password-env", "", "environment variable name for the IMAP password")
	cmd.Flags().BoolVar(&tlsEnabled, "tls", true, "enable TLS for IMAP")
	cmd.Flags().StringVar(&mailbox, "mailbox", "INBOX", "default mailbox")
	cmd.Flags().StringVar(&smtpHost, "smtp-host", "", "SMTP host")
	cmd.Flags().IntVar(&smtpPort, "smtp-port", 587, "SMTP port")
	cmd.Flags().StringVar(&smtpUsername, "smtp-username", "", "SMTP username; defaults to --username when omitted")
	cmd.Flags().StringVar(&smtpPasswordEnv, "smtp-password-env", "", "environment variable name for the SMTP password")
	cmd.Flags().BoolVar(&smtpTLSEnabled, "smtp-tls", true, "enable TLS for SMTP")
	cmd.Flags().StringVar(&dirPath, "path", "", "local .eml directory for dir driver")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing config file")
	return cmd
}

type initialAccountOptions struct {
	account         string
	driverName      string
	host            string
	port            int
	username        string
	passwordEnv     string
	tlsEnabled      bool
	mailbox         string
	smtpHost        string
	smtpPort        int
	smtpUsername    string
	smtpPasswordEnv string
	smtpTLSEnabled  bool
	dirPath         string
}

func buildInitialAccountConfig(opts initialAccountOptions) (config.AccountConfig, error) {
	account := config.AccountConfig{
		Name:    opts.account,
		Driver:  opts.driverName,
		Mailbox: strings.TrimSpace(opts.mailbox),
	}
	if account.Mailbox == "" {
		account.Mailbox = "INBOX"
	}

	switch opts.driverName {
	case "imap":
		if strings.TrimSpace(opts.host) == "" {
			return config.AccountConfig{}, fmt.Errorf("--host is required for imap config")
		}
		if opts.port <= 0 {
			return config.AccountConfig{}, fmt.Errorf("--port must be greater than 0")
		}
		if strings.TrimSpace(opts.username) == "" {
			return config.AccountConfig{}, fmt.Errorf("--username is required for imap config")
		}
		if strings.TrimSpace(opts.passwordEnv) == "" {
			return config.AccountConfig{}, fmt.Errorf("--password-env is required for imap config")
		}
		if !validEnvName(opts.passwordEnv) {
			return config.AccountConfig{}, fmt.Errorf("--password-env must be a valid environment variable name")
		}
		account.Host = strings.TrimSpace(opts.host)
		account.Port = opts.port
		account.Username = strings.TrimSpace(opts.username)
		account.Password = envReference(opts.passwordEnv)
		account.TLS = opts.tlsEnabled

		if strings.TrimSpace(opts.smtpHost) != "" || opts.smtpPort != 587 || strings.TrimSpace(opts.smtpUsername) != "" || strings.TrimSpace(opts.smtpPasswordEnv) != "" {
			if strings.TrimSpace(opts.smtpHost) == "" {
				return config.AccountConfig{}, fmt.Errorf("--smtp-host is required when SMTP options are provided")
			}
			if opts.smtpPort <= 0 {
				return config.AccountConfig{}, fmt.Errorf("--smtp-port must be greater than 0")
			}
			if strings.TrimSpace(opts.smtpPasswordEnv) == "" {
				return config.AccountConfig{}, fmt.Errorf("--smtp-password-env is required when SMTP options are provided")
			}
			if !validEnvName(opts.smtpPasswordEnv) {
				return config.AccountConfig{}, fmt.Errorf("--smtp-password-env must be a valid environment variable name")
			}
			account.SMTPHost = strings.TrimSpace(opts.smtpHost)
			account.SMTPPort = opts.smtpPort
			account.SMTPUsername = strings.TrimSpace(opts.smtpUsername)
			if account.SMTPUsername == "" {
				account.SMTPUsername = account.Username
			}
			account.SMTPPassword = envReference(opts.smtpPasswordEnv)
			account.SMTPTLS = opts.smtpTLSEnabled
		}
	case "dir":
		if strings.TrimSpace(opts.dirPath) == "" {
			return config.AccountConfig{}, fmt.Errorf("--path is required for dir config")
		}
		account.Path = strings.TrimSpace(opts.dirPath)
		account.Username = strings.TrimSpace(opts.username)
	case "stub":
		account.Username = strings.TrimSpace(opts.username)
	default:
		return config.AccountConfig{}, fmt.Errorf("unsupported driver: %s", opts.driverName)
	}

	return account, nil
}

func envReference(name string) string {
	return "${" + strings.TrimSpace(name) + "}"
}

func validEnvName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	for i, r := range name {
		if i == 0 {
			if !isEnvNameStart(r) {
				return false
			}
			continue
		}
		if !isEnvNamePart(r) {
			return false
		}
	}
	return true
}

func isEnvNameStart(r rune) bool {
	return r == '_' || ('A' <= r && r <= 'Z') || ('a' <= r && r <= 'z')
}

func isEnvNamePart(r rune) bool {
	return isEnvNameStart(r) || ('0' <= r && r <= '9')
}

// config show ─────────────────────────────────────────────────────────────────

func newConfigShowCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Print the current configuration (passwords redacted)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if configPath == "" {
				configPath = config.DefaultPath()
			}

			cfg, err := loadConfigFunc(configPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "config file:     %s\n", configPath)
			fmt.Fprintf(out, "current account: %s\n", cfg.CurrentAccount)
			fmt.Fprintf(out, "accounts:\n")
			for _, acc := range cfg.Accounts {
				fmt.Fprintf(out, "\n  name:     %s\n", acc.Name)
				fmt.Fprintf(out, "  driver:   %s\n", acc.Driver)
				switch strings.ToLower(acc.Driver) {
				case "imap":
					fmt.Fprintf(out, "  imap:\n")
					fmt.Fprintf(out, "    host:     %s:%d\n", acc.Host, acc.Port)
					fmt.Fprintf(out, "    username: %s\n", acc.Username)
					fmt.Fprintf(out, "    tls:      %v\n", acc.TLS)
					if acc.Mailbox != "" {
						fmt.Fprintf(out, "    mailbox:  %s\n", acc.Mailbox)
					}
					if acc.SMTPHost != "" {
						fmt.Fprintf(out, "  smtp:\n")
						fmt.Fprintf(out, "    host:     %s:%d\n", acc.SMTPHost, acc.SMTPPort)
						smtpUser := acc.SMTPUsername
						if smtpUser == "" {
							smtpUser = acc.Username + " (inherited)"
						}
						fmt.Fprintf(out, "    username: %s\n", smtpUser)
						fmt.Fprintf(out, "    tls:      %v\n", acc.SMTPTLS)
					}
				case "dir":
					fmt.Fprintf(out, "  path:     %s\n", acc.Path)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "config file path")
	return cmd
}

// config test ─────────────────────────────────────────────────────────────────

func newConfigTestCmd() *cobra.Command {
	var (
		configPath string
		account    string
	)

	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test the connection for the selected account",
		RunE: func(cmd *cobra.Command, args []string) error {
			selectedAccount, err := resolveSelectedAccount(configPath, account, "")
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "testing account: %s (driver: %s)\n",
				selectedAccount.Name, selectedAccount.Driver)

			drv, err := driverFactoryFunc(selectedAccount)
			if err != nil {
				return fmt.Errorf("driver init failed: %w", err)
			}

			// Attempt a List with limit=1 as a lightweight connectivity probe.
			_, err = drv.List(cmd.Context(), schema.SearchQuery{Limit: 1})
			if err != nil {
				return fmt.Errorf("connection test failed: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "ok: connection successful")
			return nil
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "config file path")
	cmd.Flags().StringVar(&account, "account", "", "account name to test")
	return cmd
}

// config capabilities ─────────────────────────────────────────────────────────

func newConfigCapabilitiesCmd() *cobra.Command {
	var (
		configPath string
		account    string
	)

	cmd := &cobra.Command{
		Use:   "capabilities",
		Short: "Print machine-readable capabilities for the selected account",
		RunE: func(cmd *cobra.Command, args []string) error {
			selectedAccount, err := resolveSelectedAccount(configPath, account, "")
			if err != nil {
				return err
			}

			result := accountCapabilities(selectedAccount)
			return writeJSON(cmd.OutOrStdout(), &result)
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "config file path")
	cmd.Flags().StringVar(&account, "account", "", "account name to inspect")
	return cmd
}

// config doctor ───────────────────────────────────────────────────────────────

func newConfigDoctorCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Validate local configuration without connecting to mailbox servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			if configPath == "" {
				configPath = config.DefaultPath()
			}

			rawCfg, err := config.LoadRaw(configPath)
			if err != nil {
				return fmt.Errorf("load raw config: %w", err)
			}

			cfg, err := loadConfigFunc(configPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			result := diagnoseConfig(configPath, rawCfg, cfg)
			return writeJSON(cmd.OutOrStdout(), &result)
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "config file path")
	return cmd
}

func diagnoseConfig(configPath string, rawCfg, resolvedCfg config.Config) schema.ConfigDiagnostics {
	result := schema.ConfigDiagnostics{
		ConfigPath: configPath,
		Status:     "ok",
		Accounts:   make([]schema.AccountDiagnostic, 0, len(rawCfg.Accounts)),
	}

	if strings.TrimSpace(rawCfg.CurrentAccount) == "" {
		result.Problems = append(result.Problems, schema.ConfigDiagnostic{
			Status:  "warning",
			Code:    "current_account_missing",
			Message: "current_account is not set; commands will require --account",
			Field:   "current_account",
		})
	} else if !configHasAccount(rawCfg, rawCfg.CurrentAccount) {
		result.Problems = append(result.Problems, schema.ConfigDiagnostic{
			Status:  "error",
			Code:    "current_account_not_found",
			Message: "current_account does not match any configured account",
			Field:   "current_account",
		})
	}
	if len(rawCfg.Accounts) == 0 {
		result.Problems = append(result.Problems, schema.ConfigDiagnostic{
			Status:  "error",
			Code:    "accounts_missing",
			Message: "no accounts are configured",
			Field:   "accounts",
		})
	}

	for i, rawAccount := range rawCfg.Accounts {
		resolvedAccount := rawAccount
		if i < len(resolvedCfg.Accounts) {
			resolvedAccount = resolvedCfg.Accounts[i]
		}
		accountResult := diagnoseAccount(rawAccount, resolvedAccount)
		result.Accounts = append(result.Accounts, accountResult)
		for _, check := range accountResult.Checks {
			if check.Status == "warning" || check.Status == "error" {
				result.Problems = append(result.Problems, check)
			}
		}
	}

	result.Status = aggregateDiagnosticsStatus(result.Problems)
	return result
}

func configHasAccount(cfg config.Config, name string) bool {
	target := strings.TrimSpace(name)
	for _, account := range cfg.Accounts {
		if strings.TrimSpace(account.Name) == target {
			return true
		}
	}
	return false
}

func diagnoseAccount(rawAccount, resolvedAccount config.AccountConfig) schema.AccountDiagnostic {
	driverName := strings.ToLower(strings.TrimSpace(rawAccount.Driver))
	result := schema.AccountDiagnostic{
		Name:         rawAccount.Name,
		Driver:       driverName,
		Status:       "ok",
		Capabilities: accountCapabilities(resolvedAccount),
	}

	if strings.TrimSpace(rawAccount.Name) == "" {
		result.Checks = append(result.Checks, diagnostic("error", "account_name_missing", "account name is required", "name"))
	}
	if driverName == "" {
		result.Checks = append(result.Checks, diagnostic("error", "driver_missing", "driver is required", "driver"))
		result.Status = aggregateDiagnosticsStatus(result.Checks)
		return result
	}

	switch driverName {
	case "imap":
		result.Checks = append(result.Checks, requireString(rawAccount.Host, "imap_host_missing", "IMAP host is required", "host"))
		result.Checks = append(result.Checks, requirePort(rawAccount.Port, "imap_port_missing", "IMAP port must be greater than 0", "port"))
		result.Checks = append(result.Checks, requireString(rawAccount.Username, "imap_username_missing", "IMAP username is required", "username"))
		result.Checks = append(result.Checks, requireRequiredSecret(
			rawAccount.Password,
			resolvedAccount.Password,
			"imap_password_missing",
			"imap_password_env_unset",
			"IMAP password or password env reference is required",
			"IMAP password environment reference is not set",
			"password",
		))
		result.Checks = append(result.Checks, diagnoseSMTP(rawAccount, resolvedAccount)...)
	case "dir":
		result.Checks = append(result.Checks, requireString(rawAccount.Path, "dir_path_missing", "dir driver requires path", "path"))
	case "stub":
		result.Checks = append(result.Checks, diagnostic("ok", "stub_configured", "stub driver is always locally available", "driver"))
	default:
		result.Checks = append(result.Checks, diagnostic("error", "driver_unsupported", "driver must be one of: imap, dir, stub", "driver"))
	}

	result.Status = aggregateDiagnosticsStatus(result.Checks)
	return result
}

func diagnoseSMTP(rawAccount, resolvedAccount config.AccountConfig) []schema.ConfigDiagnostic {
	if strings.TrimSpace(rawAccount.SMTPHost) == "" && rawAccount.SMTPPort == 0 && strings.TrimSpace(rawAccount.SMTPUsername) == "" && strings.TrimSpace(rawAccount.SMTPPassword) == "" {
		return []schema.ConfigDiagnostic{diagnostic("warning", "smtp_not_configured", "SMTP is not configured; send and reply are disabled", "smtp_host")}
	}

	checks := []schema.ConfigDiagnostic{
		requireOptionalString(rawAccount.SMTPHost, "smtp_host_missing", "SMTP host is required for outbound mail", "smtp_host"),
		requireOptionalPort(rawAccount.SMTPPort, "smtp_port_missing", "SMTP port must be greater than 0", "smtp_port"),
	}

	if strings.TrimSpace(firstNonEmpty(rawAccount.SMTPUsername, rawAccount.Username)) == "" {
		checks = append(checks, diagnostic("warning", "smtp_username_missing", "SMTP username is required or must inherit from username", "smtp_username"))
	}
	checks = append(checks, requireOptionalSecret(
		firstNonEmpty(rawAccount.SMTPPassword, rawAccount.Password),
		firstNonEmpty(resolvedAccount.SMTPPassword, resolvedAccount.Password),
		"smtp_password_missing",
		"smtp_password_env_unset",
		"SMTP password or password env reference is required for outbound mail",
		"SMTP password environment reference is not set",
		"smtp_password",
	))
	return checks
}

func requireRequiredSecret(rawValue, resolvedValue, missingCode, envUnsetCode, missingMessage, envUnsetMessage, field string) schema.ConfigDiagnostic {
	if strings.TrimSpace(rawValue) == "" {
		return diagnostic("error", missingCode, missingMessage, field)
	}
	if strings.TrimSpace(resolvedValue) == "" {
		return diagnostic("error", envUnsetCode, envUnsetMessage, field)
	}
	return diagnostic("ok", strings.TrimSuffix(missingCode, "_missing")+"_present", field+" is configured", field)
}

func requireOptionalSecret(rawValue, resolvedValue, missingCode, envUnsetCode, missingMessage, envUnsetMessage, field string) schema.ConfigDiagnostic {
	if strings.TrimSpace(rawValue) == "" {
		return diagnostic("warning", missingCode, missingMessage, field)
	}
	if strings.TrimSpace(resolvedValue) == "" {
		return diagnostic("warning", envUnsetCode, envUnsetMessage, field)
	}
	return diagnostic("ok", strings.TrimSuffix(missingCode, "_missing")+"_present", field+" is configured", field)
}

func requireOptionalString(value, code, message, field string) schema.ConfigDiagnostic {
	if strings.TrimSpace(value) == "" {
		return diagnostic("warning", code, message, field)
	}
	return diagnostic("ok", strings.TrimSuffix(code, "_missing")+"_present", field+" is configured", field)
}

func requireOptionalPort(value int, code, message, field string) schema.ConfigDiagnostic {
	if value <= 0 {
		return diagnostic("warning", code, message, field)
	}
	return diagnostic("ok", strings.TrimSuffix(code, "_missing")+"_present", field+" is configured", field)
}

func requireString(value, code, message, field string) schema.ConfigDiagnostic {
	if strings.TrimSpace(value) == "" {
		return diagnostic("error", code, message, field)
	}
	return diagnostic("ok", strings.TrimSuffix(code, "_missing")+"_present", strings.TrimSuffix(message, " is required")+" is configured", field)
}

func requirePort(value int, code, message, field string) schema.ConfigDiagnostic {
	if value <= 0 {
		return diagnostic("error", code, message, field)
	}
	return diagnostic("ok", strings.TrimSuffix(code, "_missing")+"_present", field+" is configured", field)
}

func diagnostic(status, code, message, field string) schema.ConfigDiagnostic {
	return schema.ConfigDiagnostic{
		Status:  status,
		Code:    code,
		Message: message,
		Field:   field,
	}
}

func aggregateDiagnosticsStatus(checks []schema.ConfigDiagnostic) string {
	status := "ok"
	for _, check := range checks {
		switch check.Status {
		case "error":
			return "error"
		case "warning":
			status = "warning"
		}
	}
	return status
}

func accountCapabilities(account config.AccountConfig) schema.AccountCapabilities {
	driverName := strings.ToLower(strings.TrimSpace(account.Driver))
	mailbox := strings.TrimSpace(account.Mailbox)
	if mailbox == "" {
		mailbox = "INBOX"
	}

	result := schema.AccountCapabilities{
		Account: account.Name,
		Driver:  driverName,
		Mailbox: mailbox,
		Configuration: schema.AccountCapabilityConfiguration{
			InboundConfigured:  inboundConfigured(account),
			OutboundConfigured: outboundConfigured(account),
			UsesLocalStorage:   driverName == "dir",
		},
	}

	switch driverName {
	case "imap":
		result.Capabilities = schema.MailCapabilities{
			List:       result.Configuration.InboundConfigured,
			FetchRaw:   result.Configuration.InboundConfigured,
			Search:     result.Configuration.InboundConfigured,
			Threads:    result.Configuration.InboundConfigured,
			Watch:      result.Configuration.InboundConfigured,
			Send:       result.Configuration.OutboundConfigured,
			Reply:      result.Configuration.OutboundConfigured,
			Delete:     result.Configuration.InboundConfigured,
			Move:       result.Configuration.InboundConfigured,
			MarkRead:   result.Configuration.InboundConfigured,
			LocalIndex: result.Configuration.InboundConfigured,
		}
	case "dir":
		result.Capabilities = schema.MailCapabilities{
			List:       result.Configuration.InboundConfigured,
			FetchRaw:   result.Configuration.InboundConfigured,
			Search:     result.Configuration.InboundConfigured,
			Threads:    result.Configuration.InboundConfigured,
			Delete:     result.Configuration.InboundConfigured,
			Move:       result.Configuration.InboundConfigured,
			MarkRead:   result.Configuration.InboundConfigured,
			LocalIndex: result.Configuration.InboundConfigured,
		}
	case "stub":
		result.Capabilities = schema.MailCapabilities{
			List:       true,
			FetchRaw:   true,
			Search:     true,
			Threads:    true,
			Send:       true,
			Reply:      true,
			LocalIndex: true,
		}
		result.Configuration.InboundConfigured = true
		result.Configuration.OutboundConfigured = true
	default:
		result.Capabilities = schema.MailCapabilities{}
	}

	return result
}

func inboundConfigured(account config.AccountConfig) bool {
	switch strings.ToLower(strings.TrimSpace(account.Driver)) {
	case "imap":
		return strings.TrimSpace(account.Host) != "" && account.Port != 0 && strings.TrimSpace(account.Username) != "" && strings.TrimSpace(account.Password) != ""
	case "dir":
		return strings.TrimSpace(account.Path) != ""
	case "stub":
		return true
	default:
		return false
	}
}

func outboundConfigured(account config.AccountConfig) bool {
	switch strings.ToLower(strings.TrimSpace(account.Driver)) {
	case "imap":
		username := firstNonEmpty(account.SMTPUsername, account.Username)
		password := firstNonEmpty(account.SMTPPassword, account.Password)
		return strings.TrimSpace(account.SMTPHost) != "" && account.SMTPPort != 0 && username != "" && password != ""
	case "stub":
		return true
	default:
		return false
	}
}
