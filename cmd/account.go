package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nonozone/MailCli/internal/config"
)

type accountAddResult struct {
	Status      string   `json:"status"`
	ConfigPath  string   `json:"config_path"`
	Account     string   `json:"account"`
	Provider    string   `json:"provider"`
	AuthMethod  string   `json:"auth_method"`
	Email       string   `json:"email"`
	SecretEnv   string   `json:"secret_env"`
	Inbound     bool     `json:"inbound"`
	Outbound    bool     `json:"outbound"`
	NextSteps   []string `json:"next_steps"`
	Warnings    []string `json:"warnings,omitempty"`
	Description string   `json:"description,omitempty"`
}

type providerPreset struct {
	Name                string
	Aliases             []string
	DisplayName         string
	Description         string
	AuthMethod          string
	DefaultSecretSuffix string
	IMAPHost            string
	IMAPPort            int
	IMAPTLS             bool
	SMTPHost            string
	SMTPPort            int
	SMTPTLS             bool
	OutboundByDefault   bool
	NextSteps           []string
	Warnings            []string
}

var accountNamePattern = regexp.MustCompile(`[^a-z0-9]+`)

func newAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account",
		Short: "Add and manage mailbox accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newAccountAddCmd())
	return cmd
}

func newAccountAddCmd() *cobra.Command {
	var (
		configPath  string
		provider    string
		email       string
		account     string
		authMethod  string
		passwordEnv string
		mailbox     string
		host        string
		port        int
		smtpHost    string
		smtpPort    int
		outbound    bool
		force       bool
		format      string
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add an existing mailbox with provider defaults",
		RunE: func(cmd *cobra.Command, args []string) error {
			if configPath == "" {
				configPath = config.DefaultPath()
			}
			if mailbox == "" {
				mailbox = "INBOX"
			}

			reader := bufio.NewReader(cmd.InOrStdin())
			interactive := strings.ToLower(strings.TrimSpace(format)) != "json"
			opts := accountAddOptions{
				configPath:  configPath,
				provider:    provider,
				email:       email,
				account:     account,
				authMethod:  authMethod,
				passwordEnv: passwordEnv,
				mailbox:     mailbox,
				host:        host,
				port:        port,
				smtpHost:    smtpHost,
				smtpPort:    smtpPort,
				outbound:    outbound,
				force:       force,
			}
			if err := fillAccountAddOptions(cmd.OutOrStdout(), reader, &opts, interactive); err != nil {
				return err
			}

			result, err := addAccount(opts)
			if err != nil {
				return err
			}

			switch strings.ToLower(strings.TrimSpace(format)) {
			case "", "text":
				writeAccountAddText(cmd.OutOrStdout(), result)
				return nil
			case "json":
				return writeJSON(cmd.OutOrStdout(), result)
			default:
				return fmt.Errorf("unsupported format: %s", format)
			}
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "config file path")
	cmd.Flags().StringVar(&provider, "provider", "", "provider preset: gmail, outlook, microsoft365, qq, 163, generic-imap")
	cmd.Flags().StringVar(&email, "email", "", "mailbox email address")
	cmd.Flags().StringVar(&account, "account", "", "local MailCLI account name; defaults to a sanitized email address")
	cmd.Flags().StringVar(&authMethod, "auth", "", "auth method: app_password, authorization_code, password")
	cmd.Flags().StringVar(&passwordEnv, "password-env", "", "environment variable that will contain the mailbox password, app password, or authorization code")
	cmd.Flags().StringVar(&mailbox, "mailbox", "INBOX", "default mailbox")
	cmd.Flags().StringVar(&host, "host", "", "IMAP host override; required for generic-imap")
	cmd.Flags().IntVar(&port, "port", 0, "IMAP port override")
	cmd.Flags().StringVar(&smtpHost, "smtp-host", "", "SMTP host override")
	cmd.Flags().IntVar(&smtpPort, "smtp-port", 0, "SMTP port override")
	cmd.Flags().BoolVar(&outbound, "outbound", true, "configure SMTP when the provider preset supports it")
	cmd.Flags().BoolVar(&force, "force", false, "replace an existing account with the same name")
	cmd.Flags().StringVar(&format, "format", "text", "output format: text, json")
	return cmd
}

type accountAddOptions struct {
	configPath  string
	provider    string
	email       string
	account     string
	authMethod  string
	passwordEnv string
	mailbox     string
	host        string
	port        int
	smtpHost    string
	smtpPort    int
	outbound    bool
	force       bool
}

func fillAccountAddOptions(out io.Writer, reader *bufio.Reader, opts *accountAddOptions, interactive bool) error {
	if !interactive {
		if strings.TrimSpace(opts.provider) == "" {
			return fmt.Errorf("--provider is required when --format json is used")
		}
		if strings.TrimSpace(opts.email) == "" {
			return fmt.Errorf("--email is required when --format json is used")
		}
		if strings.TrimSpace(opts.passwordEnv) == "" {
			return fmt.Errorf("--password-env is required when --format json is used")
		}
		return nil
	}

	if strings.TrimSpace(opts.provider) == "" {
		fmt.Fprintln(out, "Choose mailbox provider: gmail, outlook, microsoft365, qq, 163, generic-imap")
		value, err := readPromptValue(out, reader, "Provider")
		if err != nil {
			return err
		}
		opts.provider = value
	}
	if strings.TrimSpace(opts.email) == "" {
		value, err := readPromptValue(out, reader, "Email")
		if err != nil {
			return err
		}
		opts.email = value
	}
	if strings.TrimSpace(opts.passwordEnv) == "" {
		defaultEnv := defaultSecretEnv(opts.provider)
		fmt.Fprintf(out, "Secret environment variable [%s]\n", defaultEnv)
		value, err := readPromptValue(out, reader, "Secret env")
		if err != nil {
			return err
		}
		if strings.TrimSpace(value) == "" {
			value = defaultEnv
		}
		opts.passwordEnv = value
	}
	value, err := readPromptValue(out, reader, "Enable outbound SMTP? [Y/n]")
	if err != nil && !errors.Is(err, errPromptEOF) {
		return err
	}
	if strings.EqualFold(strings.TrimSpace(value), "n") || strings.EqualFold(strings.TrimSpace(value), "no") {
		opts.outbound = false
	}
	return nil
}

var errPromptEOF = errors.New("prompt input ended")

func readPromptValue(out io.Writer, reader *bufio.Reader, label string) (string, error) {
	fmt.Fprintf(out, "%s: ", label)
	value, err := reader.ReadString('\n')
	if err != nil {
		if strings.TrimSpace(value) == "" {
			return "", errPromptEOF
		}
	}
	return strings.TrimSpace(value), nil
}

func addAccount(opts accountAddOptions) (accountAddResult, error) {
	preset, err := resolveProviderPreset(opts.provider)
	if err != nil {
		return accountAddResult{}, err
	}
	email := strings.TrimSpace(opts.email)
	if email == "" {
		return accountAddResult{}, fmt.Errorf("email is required")
	}
	secretEnv := strings.TrimSpace(opts.passwordEnv)
	if secretEnv == "" {
		secretEnv = defaultSecretEnv(preset.Name)
	}
	if !validEnvName(secretEnv) {
		return accountAddResult{}, fmt.Errorf("--password-env must be a valid environment variable name")
	}
	accountName := strings.TrimSpace(opts.account)
	if accountName == "" {
		accountName = accountNameFromEmail(email)
	}
	authMethod := strings.TrimSpace(opts.authMethod)
	if authMethod == "" {
		authMethod = preset.AuthMethod
	}
	imapHost := firstNonEmpty(opts.host, preset.IMAPHost)
	imapPort := opts.port
	if imapPort == 0 {
		imapPort = preset.IMAPPort
	}
	smtpHost := firstNonEmpty(opts.smtpHost, preset.SMTPHost)
	smtpPort := opts.smtpPort
	if smtpPort == 0 {
		smtpPort = preset.SMTPPort
	}
	if imapHost == "" {
		return accountAddResult{}, fmt.Errorf("--host is required for provider %s", preset.Name)
	}
	if imapPort <= 0 {
		return accountAddResult{}, fmt.Errorf("--port must be greater than 0")
	}

	cfg, err := loadRawConfigForAccountAdd(opts.configPath)
	if err != nil {
		return accountAddResult{}, err
	}
	account := config.AccountConfig{
		Name:       accountName,
		Provider:   preset.Name,
		Driver:     "imap",
		AuthMethod: authMethod,
		Host:       imapHost,
		Port:       imapPort,
		Username:   email,
		Password:   envReference(secretEnv),
		TLS:        preset.IMAPTLS,
		Mailbox:    firstNonEmpty(opts.mailbox, "INBOX"),
	}
	outbound := opts.outbound && (preset.OutboundByDefault || opts.smtpHost != "") && smtpHost != ""
	if !outbound && opts.smtpPort != 0 && smtpHost == "" {
		return accountAddResult{}, fmt.Errorf("--smtp-host is required when --smtp-port is provided")
	}
	if outbound && smtpPort <= 0 {
		return accountAddResult{}, fmt.Errorf("--smtp-port must be greater than 0 when SMTP is configured")
	}
	if outbound {
		account.SMTPHost = smtpHost
		account.SMTPPort = smtpPort
		account.SMTPUsername = email
		account.SMTPPassword = envReference(secretEnv)
		account.SMTPTLS = preset.SMTPTLS
	}

	replaced := false
	for i, existing := range cfg.Accounts {
		if existing.Name == accountName {
			if !opts.force {
				return accountAddResult{}, fmt.Errorf("account %q already exists; pass --force to replace it", accountName)
			}
			cfg.Accounts[i] = account
			replaced = true
			break
		}
	}
	if !replaced {
		cfg.Accounts = append(cfg.Accounts, account)
	}
	cfg.CurrentAccount = accountName

	data, err := config.Marshal(cfg)
	if err != nil {
		return accountAddResult{}, fmt.Errorf("marshal config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(opts.configPath), 0o755); err != nil {
		return accountAddResult{}, fmt.Errorf("create config directory: %w", err)
	}
	if err := os.WriteFile(opts.configPath, data, 0o600); err != nil {
		return accountAddResult{}, fmt.Errorf("write config: %w", err)
	}
	if err := os.Chmod(opts.configPath, 0o600); err != nil {
		return accountAddResult{}, fmt.Errorf("set config permissions: %w", err)
	}

	nextSteps := []string{
		fmt.Sprintf("Set %s in your shell or secret manager before running config test.", secretEnv),
		fmt.Sprintf("Run mailcli config doctor --config %s", opts.configPath),
		fmt.Sprintf("Run mailcli config test --config %s --account %s", opts.configPath, accountName),
	}
	nextSteps = append(nextSteps, preset.NextSteps...)
	if outbound {
		nextSteps = append(nextSteps, "Use mailcli config capabilities to confirm send/reply support before enabling outbound automation.")
	}

	status := "configured"
	if replaced {
		status = "replaced"
	}
	return accountAddResult{
		Status:      status,
		ConfigPath:  opts.configPath,
		Account:     accountName,
		Provider:    preset.Name,
		AuthMethod:  authMethod,
		Email:       email,
		SecretEnv:   secretEnv,
		Inbound:     true,
		Outbound:    outbound,
		NextSteps:   nextSteps,
		Warnings:    preset.Warnings,
		Description: preset.Description,
	}, nil
}

func loadRawConfigForAccountAdd(path string) (config.Config, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		if strings.TrimSpace(string(data)) == "" {
			return config.Config{}, nil
		}
		return config.UnmarshalRaw(data)
	}
	if os.IsNotExist(err) {
		return config.Config{}, nil
	}
	return config.Config{}, fmt.Errorf("read config: %w", err)
}

func writeAccountAddText(out io.Writer, result accountAddResult) {
	fmt.Fprintf(out, "Configured account %q for %s.\n", result.Account, result.Provider)
	fmt.Fprintf(out, "Config: %s\n", result.ConfigPath)
	fmt.Fprintf(out, "Email: %s\n", result.Email)
	fmt.Fprintf(out, "Auth: %s via ${%s}\n", result.AuthMethod, result.SecretEnv)
	if result.Outbound {
		fmt.Fprintln(out, "Outbound SMTP: enabled")
	} else {
		fmt.Fprintln(out, "Outbound SMTP: not configured")
	}
	if len(result.Warnings) > 0 {
		fmt.Fprintln(out, "Warnings:")
		for _, warning := range result.Warnings {
			fmt.Fprintf(out, "- %s\n", warning)
		}
	}
	fmt.Fprintln(out, "Next steps:")
	for _, step := range result.NextSteps {
		fmt.Fprintf(out, "- %s\n", step)
	}
}

func resolveProviderPreset(name string) (providerPreset, error) {
	target := strings.ToLower(strings.TrimSpace(name))
	for _, preset := range providerPresets() {
		if target == preset.Name {
			return preset, nil
		}
		for _, alias := range preset.Aliases {
			if target == alias {
				return preset, nil
			}
		}
	}
	return providerPreset{}, fmt.Errorf("unsupported provider %q; use gmail, outlook, microsoft365, qq, 163, or generic-imap", name)
}

func providerPresets() []providerPreset {
	return []providerPreset{
		{
			Name:                "gmail",
			Aliases:             []string{"google"},
			DisplayName:         "Gmail",
			Description:         "Gmail IMAP with app password authentication.",
			AuthMethod:          "app_password",
			DefaultSecretSuffix: "GMAIL_APP_PASSWORD",
			IMAPHost:            "imap.gmail.com",
			IMAPPort:            993,
			IMAPTLS:             true,
			SMTPHost:            "smtp.gmail.com",
			SMTPPort:            465,
			SMTPTLS:             true,
			OutboundByDefault:   true,
			NextSteps: []string{
				"Enable IMAP in Gmail settings if it is disabled.",
				"Use a Google app password; do not put your normal Google password in config.",
			},
		},
		{
			Name:                "outlook",
			Aliases:             []string{"hotmail", "live"},
			DisplayName:         "Outlook.com",
			Description:         "Outlook IMAP preset. Some accounts require OAuth or tenant-level SMTP AUTH settings.",
			AuthMethod:          "password",
			DefaultSecretSuffix: "OUTLOOK_PASSWORD",
			IMAPHost:            "outlook.office365.com",
			IMAPPort:            993,
			IMAPTLS:             true,
			OutboundByDefault:   false,
			NextSteps: []string{
				"Run config test. If authentication fails, the account likely requires OAuth or an admin-enabled SMTP/IMAP policy.",
			},
			Warnings: []string{
				"Outlook and Microsoft 365 OAuth is not implemented in MailCLI yet; this preset is read-first.",
			},
		},
		{
			Name:                "microsoft365",
			Aliases:             []string{"office365", "m365"},
			DisplayName:         "Microsoft 365",
			Description:         "Microsoft 365 IMAP preset. Tenant policy may disable basic IMAP/SMTP authentication.",
			AuthMethod:          "password",
			DefaultSecretSuffix: "M365_PASSWORD",
			IMAPHost:            "outlook.office365.com",
			IMAPPort:            993,
			IMAPTLS:             true,
			OutboundByDefault:   false,
			NextSteps: []string{
				"Ask your Microsoft 365 admin whether IMAP and SMTP AUTH are enabled for this mailbox.",
			},
			Warnings: []string{
				"OAuth support is planned but not implemented; this preset may not work on tenants that block password-style IMAP.",
			},
		},
		{
			Name:                "qq",
			Aliases:             []string{"qqmail"},
			DisplayName:         "QQ Mail",
			Description:         "QQ Mail IMAP/SMTP with mailbox authorization code.",
			AuthMethod:          "authorization_code",
			DefaultSecretSuffix: "QQ_AUTH_CODE",
			IMAPHost:            "imap.qq.com",
			IMAPPort:            993,
			IMAPTLS:             true,
			SMTPHost:            "smtp.qq.com",
			SMTPPort:            465,
			SMTPTLS:             true,
			OutboundByDefault:   true,
			NextSteps: []string{
				"Enable IMAP/SMTP in QQ Mail settings and generate an authorization code.",
			},
		},
		{
			Name:                "163",
			Aliases:             []string{"netease", "netease163"},
			DisplayName:         "163 Mail",
			Description:         "NetEase 163 IMAP/SMTP with authorization code.",
			AuthMethod:          "authorization_code",
			DefaultSecretSuffix: "163_AUTH_CODE",
			IMAPHost:            "imap.163.com",
			IMAPPort:            993,
			IMAPTLS:             true,
			SMTPHost:            "smtp.163.com",
			SMTPPort:            465,
			SMTPTLS:             true,
			OutboundByDefault:   true,
			NextSteps: []string{
				"Enable IMAP/SMTP in 163 Mail settings and generate an authorization code.",
			},
		},
		{
			Name:                "generic-imap",
			Aliases:             []string{"generic", "imap"},
			DisplayName:         "Generic IMAP",
			Description:         "Generic IMAP account with explicit server settings.",
			AuthMethod:          "password",
			DefaultSecretSuffix: "IMAP_PASSWORD",
			IMAPPort:            993,
			IMAPTLS:             true,
			OutboundByDefault:   false,
			Warnings: []string{
				"generic-imap does not guess host names; pass --host and optional SMTP settings explicitly.",
			},
		},
	}
}

func accountNameFromEmail(email string) string {
	name := strings.ToLower(strings.TrimSpace(email))
	name = accountNamePattern.ReplaceAllString(name, "_")
	name = strings.Trim(name, "_")
	if name == "" {
		return "mailbox"
	}
	return name
}

func defaultSecretEnv(provider string) string {
	preset, err := resolveProviderPreset(provider)
	if err == nil && preset.DefaultSecretSuffix != "" {
		return "MAILCLI_" + preset.DefaultSecretSuffix
	}
	name := strings.ToUpper(accountNamePattern.ReplaceAllString(strings.ToLower(strings.TrimSpace(provider)), "_"))
	name = strings.Trim(name, "_")
	if name == "" {
		name = "MAILBOX"
	}
	return "MAILCLI_" + name + "_PASSWORD"
}
