package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/nonozone/MailCli/examples/internal/agent"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type options struct {
	mailcliBin      string
	email           string
	messageID       string
	config          string
	account         string
	replyText       string
	fromAddress     string
	fromName        string
	agentProvider   string
	providerCommand string
	providerArgs    multiFlag
}

type multiFlag []string

func (m *multiFlag) String() string {
	return fmt.Sprint([]string(*m))
}

func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func run() error {
	opts, err := parseArgs()
	if err != nil {
		return err
	}

	message, err := loadMessage(opts)
	if err != nil {
		return err
	}

	analysis, err := analyzeWithProvider(message, opts)
	if err != nil {
		return err
	}

	report := map[string]any{
		"tool":     "mailcli-agent-example",
		"source":   buildSource(opts),
		"message":  message,
		"analysis": analysis,
	}

	replyText := opts.replyText
	if replyText == "" {
		replyText, _ = analysis["reply_text"].(string)
	}
	if replyText != "" {
		if opts.fromAddress == "" {
			return fmt.Errorf("--from-address is required when --reply-text is used")
		}
		draft, err := buildReplyDraft(message, opts)
		if err != nil {
			return err
		}
		draft["body_text"] = replyText
		draftJSON, err := agent.MarshalCompact(draft)
		if err != nil {
			return err
		}
		mime, err := agent.RunCommand([]string{opts.mailcliBin, "reply", "--dry-run", "-"}, draftJSON)
		if err != nil {
			return err
		}
		report["reply"] = map[string]any{
			"mode":  "dry_run",
			"draft": draft,
			"mime":  mime,
		}
		analysis["decision"] = "draft_reply"
	}

	return agent.WriteJSON(os.Stdout, report)
}

func parseArgs() (options, error) {
	var opts options
	flags := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.StringVar(&opts.mailcliBin, "mailcli-bin", "mailcli", "path to the mailcli binary")
	flags.StringVar(&opts.email, "email", "", "local .eml file to parse")
	flags.StringVar(&opts.messageID, "message-id", "", "message id to fetch through mailcli get")
	flags.StringVar(&opts.config, "config", "", "mailcli config path for inbox-backed commands")
	flags.StringVar(&opts.account, "account", "", "mailcli account override")
	flags.StringVar(&opts.replyText, "reply-text", "", "optional reply body text to compile with mailcli reply --dry-run")
	flags.StringVar(&opts.fromAddress, "from-address", "", "from address to use for reply dry-run")
	flags.StringVar(&opts.fromName, "from-name", "", "optional from display name for reply dry-run")
	flags.StringVar(&opts.agentProvider, "agent-provider", "builtin", "analysis provider: builtin or external")
	flags.StringVar(&opts.providerCommand, "provider-command", "", "external provider command")
	flags.Var(&opts.providerArgs, "provider-arg", "repeatable argument for the external provider command")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return options{}, err
	}

	if (opts.email == "") == (opts.messageID == "") {
		return options{}, fmt.Errorf("exactly one of --email or --message-id is required")
	}
	if opts.agentProvider != "builtin" && opts.agentProvider != "external" {
		return options{}, fmt.Errorf("--agent-provider must be builtin or external")
	}
	if opts.agentProvider == "external" && opts.providerCommand == "" {
		return options{}, fmt.Errorf("--provider-command is required when --agent-provider external is used")
	}
	return opts, nil
}

func buildSource(opts options) map[string]any {
	if opts.email != "" {
		return map[string]any{
			"mode":  "email",
			"value": opts.email,
		}
	}
	return map[string]any{
		"mode":    "message_id",
		"value":   opts.messageID,
		"config":  opts.config,
		"account": opts.account,
	}
}

func loadMessage(opts options) (map[string]any, error) {
	var command []string
	if opts.email != "" {
		command = []string{opts.mailcliBin, "parse", "--format", "json", opts.email}
	} else {
		command = []string{opts.mailcliBin, "get", "--format", "json"}
		if opts.config != "" {
			command = append(command, "--config", opts.config)
		}
		if opts.account != "" {
			command = append(command, "--account", opts.account)
		}
		command = append(command, opts.messageID)
	}

	output, err := agent.RunCommand(command, "")
	if err != nil {
		return nil, err
	}
	var message map[string]any
	if err := json.Unmarshal([]byte(output), &message); err != nil {
		return nil, err
	}
	return message, nil
}

func analyzeWithProvider(message map[string]any, opts options) (map[string]any, error) {
	if opts.agentProvider == "external" {
		payload := map[string]any{
			"source":      buildSource(opts),
			"message":     message,
			"wants_reply": opts.replyText != "",
		}
		payloadJSON, err := agent.MarshalCompact(payload)
		if err != nil {
			return nil, err
		}
		command := append([]string{opts.providerCommand}, []string(opts.providerArgs)...)
		output, err := agent.RunCommand(command, payloadJSON)
		if err != nil {
			return nil, err
		}
		analysis, err := agent.ParseExternalAnalysis(output)
		if err != nil {
			return nil, err
		}
		if _, ok := analysis["provider"]; !ok {
			analysis["provider"] = "external"
		}
		return analysis, nil
	}

	analysis := agent.AnalyzeMessage(message, opts.replyText != "")
	analysis["provider"] = "builtin"
	return analysis, nil
}

func buildReplyDraft(message map[string]any, opts options) (map[string]any, error) {
	meta := agent.MapValue(message, "meta")
	sender := agent.MapValue(meta, "from")
	senderAddress := agent.StringValue(sender, "address")
	if senderAddress == "" {
		return nil, fmt.Errorf("message does not contain a sender address for reply drafting")
	}

	references := agent.StringSlice(meta["references"])
	messageID := agent.StringValue(meta, "message_id")
	if messageID != "" && !containsString(references, messageID) {
		references = append(references, messageID)
	}

	from := map[string]any{"address": opts.fromAddress}
	if opts.fromName != "" {
		from["name"] = opts.fromName
	}
	to := map[string]any{"address": senderAddress}
	if name := agent.StringValue(sender, "name"); name != "" {
		to["name"] = name
	}

	draft := map[string]any{
		"from":                from,
		"to":                  []any{to},
		"body_text":           opts.replyText,
		"reply_to_message_id": messageID,
		"references":          references,
		"subject":             agent.StringValue(meta, "subject"),
	}
	if opts.account != "" {
		draft["account"] = opts.account
	}
	return draft, nil
}

func containsString(items []any, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}
