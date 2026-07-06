package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/nonozone/MailCli/examples/internal/agent"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type options struct {
	mailcliBin         string
	config             string
	account            string
	mailbox            string
	index              string
	query              string
	threadID           string
	syncLimit          int
	threadLimit        int
	threadMessageLimit int
	skipSync           bool
	replyText          string
	fromAddress        string
	fromName           string
	agentProvider      string
	providerCommand    string
	providerArgs       multiFlag
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

	var syncResult any
	if !opts.skipSync {
		syncResult, err = runSync(opts)
		if err != nil {
			return err
		}
	}

	threadSummaries, err := loadThreadSummaries(opts)
	if err != nil {
		return err
	}
	selection, err := selectThread(threadSummaries, opts)
	if err != nil {
		return err
	}
	threadMessages, err := loadThreadMessages(agent.StringValue(selection, "thread_id"), opts, opts.threadMessageLimit)
	if err != nil {
		return err
	}
	if len(threadMessages) == 0 {
		return fmt.Errorf("selected thread did not return any local messages")
	}

	threadMessages, latestMessage, err := ensureLatestMessage(selection, threadMessages, opts)
	if err != nil {
		return err
	}

	analysis, err := analyzeWithProvider(selection, threadSummaries, threadMessages, latestMessage, opts)
	if err != nil {
		return err
	}

	report := map[string]any{
		"tool":             "mailcli-thread-agent-example",
		"source":           buildSource(opts),
		"selection":        selection,
		"thread_summaries": threadSummaries,
		"thread_messages":  threadMessages,
		"latest_message":   latestMessage,
		"analysis":         analysis,
	}
	if syncResult != nil {
		report["sync"] = syncResult
	}

	replyText := opts.replyText
	if replyText == "" {
		replyText, _ = analysis["reply_text"].(string)
	}
	if replyText != "" {
		if opts.fromAddress == "" {
			return fmt.Errorf("--from-address is required when --reply-text is used")
		}
		draft, err := buildReplyDraft(selection, latestMessage, opts)
		if err != nil {
			return err
		}
		draft["body_text"] = replyText
		mime, err := compileReplyDryRun(draft, opts)
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
	flags.StringVar(&opts.config, "config", "", "mailcli config path")
	flags.StringVar(&opts.account, "account", "", "mailcli account override")
	flags.StringVar(&opts.mailbox, "mailbox", "", "mailcli mailbox override")
	flags.StringVar(&opts.index, "index", "", "local index path")
	flags.StringVar(&opts.query, "query", "", "thread query used for selection")
	flags.StringVar(&opts.threadID, "thread-id", "", "explicit thread id override")
	flags.IntVar(&opts.syncLimit, "sync-limit", 10, "maximum messages to sync before thread selection")
	flags.IntVar(&opts.threadLimit, "thread-limit", 10, "maximum thread summaries to load")
	flags.IntVar(&opts.threadMessageLimit, "thread-message-limit", 50, "maximum local thread messages to load")
	flags.BoolVar(&opts.skipSync, "skip-sync", false, "skip mailcli sync and use the existing local index")
	flags.StringVar(&opts.replyText, "reply-text", "", "optional reply body text to compile with mailcli reply --dry-run")
	flags.StringVar(&opts.fromAddress, "from-address", "", "from address to use for reply dry-run")
	flags.StringVar(&opts.fromName, "from-name", "", "optional from display name for reply dry-run")
	flags.StringVar(&opts.agentProvider, "agent-provider", "builtin", "analysis provider: builtin or external")
	flags.StringVar(&opts.providerCommand, "provider-command", "", "external provider command")
	flags.Var(&opts.providerArgs, "provider-arg", "repeatable argument for the external provider command")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return options{}, err
	}

	if opts.index == "" {
		return options{}, fmt.Errorf("--index is required")
	}
	if !opts.skipSync && opts.config == "" {
		return options{}, fmt.Errorf("--config is required unless --skip-sync is used")
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
	return map[string]any{
		"mode":      "local_thread",
		"config":    opts.config,
		"account":   opts.account,
		"mailbox":   opts.mailbox,
		"index":     opts.index,
		"query":     opts.query,
		"thread_id": opts.threadID,
		"skip_sync": opts.skipSync,
	}
}

func runSync(opts options) (any, error) {
	command := []string{
		opts.mailcliBin,
		"sync",
		"--format",
		"json",
		"--config",
		opts.config,
		"--index",
		opts.index,
		"--limit",
		fmt.Sprint(opts.syncLimit),
	}
	if opts.account != "" {
		command = append(command, "--account", opts.account)
	}
	if opts.mailbox != "" {
		command = append(command, "--mailbox", opts.mailbox)
	}
	return agent.RunJSON(command, "")
}

func loadThreadSummaries(opts options) ([]map[string]any, error) {
	command := []string{
		opts.mailcliBin,
		"threads",
		"--format",
		"json",
		"--index",
		opts.index,
		"--limit",
		fmt.Sprint(opts.threadLimit),
	}
	if opts.account != "" {
		command = append(command, "--account", opts.account)
	}
	if opts.mailbox != "" {
		command = append(command, "--mailbox", opts.mailbox)
	}
	if opts.query != "" {
		command = append(command, opts.query)
	}

	value, err := agent.RunJSON(command, "")
	if err != nil {
		return nil, err
	}
	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("mailcli threads must return a JSON array")
	}
	return mapsFromSlice(items), nil
}

func selectThread(threadSummaries []map[string]any, opts options) (map[string]any, error) {
	explicitThreadID := strings.TrimSpace(opts.threadID)
	if explicitThreadID != "" {
		for _, item := range threadSummaries {
			if agent.StringValue(item, "thread_id") == explicitThreadID {
				result := cloneMap(item)
				result["selection_strategy"] = "explicit_thread_id"
				return result, nil
			}
		}
		return map[string]any{
			"thread_id":          explicitThreadID,
			"selection_strategy": "explicit_thread_id",
		}, nil
	}

	if len(threadSummaries) == 0 {
		return nil, fmt.Errorf("no local thread matched the current query")
	}

	result := cloneMap(threadSummaries[0])
	result["selection_strategy"] = "top_thread"
	return result, nil
}

func loadThreadMessages(threadID string, opts options, limit int) ([]map[string]any, error) {
	command := []string{
		opts.mailcliBin,
		"thread",
		"--format",
		"json",
		"--index",
		opts.index,
	}
	if limit >= 0 {
		command = append(command, "--limit", fmt.Sprint(limit))
	}
	if opts.account != "" {
		command = append(command, "--account", opts.account)
	}
	if opts.mailbox != "" {
		command = append(command, "--mailbox", opts.mailbox)
	}
	command = append(command, threadID)

	value, err := agent.RunJSON(command, "")
	if err != nil {
		return nil, err
	}
	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("mailcli thread must return a JSON array")
	}
	return mapsFromSlice(items), nil
}

func ensureLatestMessage(selection map[string]any, threadMessages []map[string]any, opts options) ([]map[string]any, map[string]any, error) {
	expectedLatestID := strings.TrimSpace(agent.StringValue(selection, "last_message_id"))
	if expectedLatestID == "" {
		return threadMessages, threadMessages[len(threadMessages)-1], nil
	}

	for _, item := range threadMessages {
		if agent.StringValue(item, "id") == expectedLatestID {
			return threadMessages, item, nil
		}
	}

	reloadedMessages, err := loadThreadMessages(agent.StringValue(selection, "thread_id"), opts, 0)
	if err != nil {
		return nil, nil, err
	}
	if len(reloadedMessages) == 0 {
		return nil, nil, fmt.Errorf("selected thread did not return any local messages after reload")
	}
	for _, item := range reloadedMessages {
		if agent.StringValue(item, "id") == expectedLatestID {
			return reloadedMessages, item, nil
		}
	}
	return reloadedMessages, reloadedMessages[len(reloadedMessages)-1], nil
}

func analyzeWithProvider(selection map[string]any, threadSummaries, threadMessages []map[string]any, latestMessage map[string]any, opts options) (map[string]any, error) {
	if opts.agentProvider == "external" {
		payload := map[string]any{
			"source":           buildSource(opts),
			"selection":        selection,
			"thread_summaries": threadSummaries,
			"thread_messages":  threadMessages,
			"latest_message":   latestMessage,
			"wants_reply":      opts.replyText != "",
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

	analysis := analyzeThread(selection, latestMessage, opts.replyText != "")
	analysis["provider"] = "builtin"
	return analysis, nil
}

func analyzeThread(selection, latestMessage map[string]any, wantsReply bool) map[string]any {
	message := agent.MapValue(latestMessage, "message")
	analysis := agent.AnalyzeMessage(message, wantsReply)
	if analysis["summary"] == "" {
		content := agent.MapValue(message, "content")
		summary := agent.StringValue(content, "snippet")
		if summary == "" {
			summary = agent.StringValue(content, "body_md")
		}
		if summary == "" {
			summary = agent.StringValue(selection, "last_message_preview")
		}
		analysis["summary"] = summary
	}
	return analysis
}

func buildReplyDraft(selection, latestMessage map[string]any, opts options) (map[string]any, error) {
	message := agent.MapValue(latestMessage, "message")
	meta := agent.MapValue(message, "meta")
	sender := agent.MapValue(meta, "from")
	senderAddress := agent.StringValue(sender, "address")
	if senderAddress == "" {
		return nil, fmt.Errorf("latest thread message does not contain a sender address for reply drafting")
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
		"from":      from,
		"to":        []any{to},
		"body_text": "",
	}

	account := strings.TrimSpace(opts.account)
	if account == "" {
		account = agent.StringValue(latestMessage, "account")
	}
	if account != "" {
		draft["account"] = account
	}

	if opts.config != "" {
		draft["reply_to_id"] = latestMessage["id"]
	} else {
		references := agent.StringSlice(meta["references"])
		messageID := agent.StringValue(meta, "message_id")
		if messageID != "" && !containsString(references, messageID) {
			references = append(references, messageID)
		}
		draft["reply_to_message_id"] = messageID
		draft["references"] = references
		subject := agent.StringValue(meta, "subject")
		if subject == "" {
			subject = agent.StringValue(selection, "subject")
		}
		draft["subject"] = subject
	}

	return draft, nil
}

func compileReplyDryRun(draft map[string]any, opts options) (string, error) {
	command := []string{opts.mailcliBin, "reply", "--dry-run"}
	if opts.config != "" {
		command = append(command, "--config", opts.config)
	}
	if opts.account != "" {
		command = append(command, "--account", opts.account)
	}
	command = append(command, "-")

	draftJSON, err := agent.MarshalCompact(draft)
	if err != nil {
		return "", err
	}
	return agent.RunCommand(command, draftJSON)
}

func mapsFromSlice(items []any) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if value, ok := item.(map[string]any); ok {
			out = append(out, value)
		}
	}
	return out
}

func cloneMap(input map[string]any) map[string]any {
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func containsString(items []any, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}
