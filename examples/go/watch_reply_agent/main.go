package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nonozone/MailCli/examples/internal/agent"
)

type options struct {
	mailcliBin      string
	account         string
	fromAddress     string
	fromName        string
	autoSend        bool
	dryRun          bool
	draftReplies    bool
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

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	opts := parseArgs()
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if err := handleLine(line, opts); err != nil {
			fmt.Fprintf(os.Stderr, "[agent] %v\n", err)
		}
	}
	return scanner.Err()
}

func parseArgs() options {
	var opts options
	flag.StringVar(&opts.mailcliBin, "mailcli-bin", "mailcli", "path to the mailcli binary")
	flag.StringVar(&opts.account, "account", os.Getenv("MAILCLI_ACCOUNT"), "account name passed to mailcli reply")
	flag.StringVar(&opts.fromAddress, "from-address", os.Getenv("MAILCLI_FROM_ADDRESS"), "from address for reply drafting")
	flag.StringVar(&opts.fromName, "from-name", os.Getenv("MAILCLI_FROM_NAME"), "optional from display name")
	flag.BoolVar(&opts.autoSend, "auto-send", os.Getenv("MAILCLI_AUTO_SEND") == "1", "send replies instead of printing draft JSON")
	flag.BoolVar(&opts.dryRun, "dry-run", os.Getenv("MAILCLI_DRY_RUN") == "1", "compile MIME without sending")
	flag.BoolVar(&opts.draftReplies, "draft-replies", false, "ask the provider to draft replies for reply-worthy messages")
	flag.StringVar(&opts.providerCommand, "provider-command", "", "optional external provider command")
	flag.Var(&opts.providerArgs, "provider-arg", "repeatable argument for the external provider command")
	flag.Parse()
	return opts
}

func handleLine(line string, opts options) error {
	var event map[string]any
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return nil
	}

	switch event["event"] {
	case "watching":
		fmt.Fprintf(os.Stderr, "[agent] monitoring %v/%v\n", event["account"], event["mailbox"])
	case "new_message":
		message := agent.MapValue(event, "message")
		subject := agent.StringValue(agent.MapValue(message, "meta"), "subject")
		if subject == "" {
			subject = "(no subject)"
		}
		fmt.Fprintf(os.Stderr, "[agent] new message: %s\n", subject)
		return handleMessage(message, opts)
	case "error":
		fmt.Fprintf(os.Stderr, "[agent] error: %v\n", event["error"])
	}
	return nil
}

func handleMessage(message map[string]any, opts options) error {
	analysis, err := analyze(message, opts)
	if err != nil {
		return err
	}
	replyText, _ := analysis["reply_text"].(string)
	if analysis["decision"] != "draft_reply" || replyText == "" {
		fmt.Fprintf(os.Stderr, "[agent] skip: %v\n", analysis["summary"])
		return nil
	}
	if opts.fromAddress == "" {
		return fmt.Errorf("--from-address is required to compile reply drafts")
	}

	draft, err := buildReplyDraft(message, opts, replyText)
	if err != nil {
		return err
	}
	if opts.autoSend || opts.dryRun {
		return executeReply(draft, opts)
	}

	return agent.WriteJSON(os.Stdout, map[string]any{
		"ts":              time.Now().UTC().Format(time.RFC3339),
		"in_reply_to":     draft["reply_to_message_id"],
		"subject":         draft["subject"],
		"draft_body_text": replyText,
		"analysis":        analysis,
	})
}

func analyze(message map[string]any, opts options) (map[string]any, error) {
	if opts.providerCommand == "" {
		analysis := agent.AnalyzeTemplatePayload(map[string]any{
			"message":     message,
			"wants_reply": opts.draftReplies,
		})
		analysis["provider"] = "builtin"
		return analysis, nil
	}

	payload := map[string]any{
		"message":     message,
		"wants_reply": opts.draftReplies,
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
	analysis["provider"] = "external"
	return analysis, nil
}

func buildReplyDraft(message map[string]any, opts options, replyText string) (map[string]any, error) {
	meta := agent.MapValue(message, "meta")
	sender := agent.MapValue(meta, "from")
	senderAddress := agent.StringValue(sender, "address")
	if senderAddress == "" {
		return nil, fmt.Errorf("message does not contain a sender address for reply drafting")
	}

	messageID := agent.StringValue(meta, "message_id")
	references := agent.StringSlice(meta["references"])
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
		"reply_to_message_id": messageID,
		"references":          references,
		"subject":             agent.StringValue(meta, "subject"),
		"body_text":           replyText,
	}
	if opts.account != "" {
		draft["account"] = opts.account
	}
	return draft, nil
}

func executeReply(draft map[string]any, opts options) error {
	draftJSON, err := agent.MarshalCompact(draft)
	if err != nil {
		return err
	}
	command := []string{opts.mailcliBin, "reply"}
	if opts.dryRun {
		command = append(command, "--dry-run")
	}
	if opts.account != "" {
		command = append(command, "--account", opts.account)
	}
	command = append(command, "-")
	output, err := agent.RunCommand(command, draftJSON)
	if err != nil {
		return err
	}
	fmt.Fprint(os.Stderr, output)
	return nil
}

func containsString(items []any, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}
