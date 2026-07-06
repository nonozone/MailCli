package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	"github.com/nonozone/MailCli/examples/internal/agent"
	"github.com/nonozone/MailCli/internal/config"
)

const (
	canonicalIndexedAt = "2026-03-27T15:11:13Z"
	canonicalMIMEDate  = "Fri, 27 Mar 2026 15:11:13 +0000"
	canonicalMessageID = "<generated@mailcli.local>"
	canonicalIndexPath = "/tmp/mailcli-fixtures-index.json"
	canonicalConfig    = "examples/config/fixtures-dir.yaml"
)

type options struct {
	mailcliBin         string
	config             string
	account            string
	index              string
	outputDir          string
	query              string
	mailbox            string
	syncLimit          int
	hasSyncLimit       bool
	threadLimit        int
	threadMessageLimit int
	fromAddress        string
	replyText          string
	workdir            string
	check              bool
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	opts, err := parseArgs()
	if err != nil {
		return err
	}

	if opts.check {
		return runCheckMode(opts)
	}

	return generateArtifacts(opts, opts.outputDir)
}

func parseArgs() (options, error) {
	var opts options
	flags := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.StringVar(&opts.mailcliBin, "mailcli-bin", "mailcli", "path to the mailcli binary")
	flags.StringVar(&opts.config, "config", "", "mailcli config path")
	flags.StringVar(&opts.account, "account", "", "mailcli account name")
	flags.StringVar(&opts.index, "index", "", "local index path to build during refresh")
	flags.StringVar(&opts.outputDir, "output-dir", "", "directory where artifacts should be written")
	flags.StringVar(&opts.query, "query", "invoice", "thread query used for demo selection")
	flags.StringVar(&opts.mailbox, "mailbox", "", "optional mailbox override")
	flags.IntVar(&opts.syncLimit, "sync-limit", 0, "sync limit for the demo refresh")
	flags.IntVar(&opts.threadLimit, "thread-limit", 10, "thread summary limit")
	flags.IntVar(&opts.threadMessageLimit, "thread-message-limit", 50, "thread message limit")
	flags.StringVar(&opts.fromAddress, "from-address", "support@nono.im", "from address for reply dry-run")
	flags.StringVar(&opts.replyText, "reply-text", "Thanks, we have received the invoice notification.", "reply body text for the generated reply dry-run")
	flags.StringVar(&opts.workdir, "workdir", "", "optional working directory for command execution")
	flags.BoolVar(&opts.check, "check", false, "verify that the target artifact directory already matches freshly generated output")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return options{}, err
	}
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "sync-limit" {
			opts.hasSyncLimit = true
		}
	})

	if opts.config == "" {
		return options{}, fmt.Errorf("--config is required")
	}
	if opts.account == "" {
		return options{}, fmt.Errorf("--account is required")
	}
	if opts.index == "" {
		return options{}, fmt.Errorf("--index is required")
	}
	if opts.outputDir == "" {
		return options{}, fmt.Errorf("--output-dir is required")
	}
	return opts, nil
}

func generateArtifacts(opts options, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}

	resetIndexFile(opts.index)
	syncResult, err := agent.RunJSONInDir(buildSyncCommand(opts), "", opts.workdir)
	if err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outputDir, "sync.json"), normalizeDemoJSON(syncResult)); err != nil {
		return err
	}

	threads, err := agent.RunJSONInDir(buildThreadsCommand(opts), "", opts.workdir)
	if err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outputDir, "threads.json"), threads); err != nil {
		return err
	}
	threadItems, ok := threads.([]any)
	if !ok || len(threadItems) == 0 {
		return fmt.Errorf("mailcli threads returned no local thread summaries")
	}
	selection, ok := threadItems[0].(map[string]any)
	if !ok {
		return fmt.Errorf("selected thread is not a JSON object")
	}
	threadID := agent.StringValue(selection, "thread_id")
	if threadID == "" {
		return fmt.Errorf("selected thread is missing thread_id")
	}

	threadMessages, err := agent.RunJSONInDir(buildThreadCommand(opts, threadID), "", opts.workdir)
	if err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outputDir, "thread.json"), normalizeDemoJSON(threadMessages)); err != nil {
		return err
	}

	resetIndexFile(opts.index)
	report, err := agent.RunJSON(buildAgentCommand(opts), "")
	if err != nil {
		return err
	}
	report = normalizeDemoJSON(report)
	reportMap, ok := report.(map[string]any)
	if !ok {
		return fmt.Errorf("agent report must be a JSON object")
	}

	reply := agent.MapValue(reportMap, "reply")
	if len(reply) == 0 {
		return fmt.Errorf("agent report is missing reply section")
	}
	draft := agent.MapValue(reply, "draft")
	if len(draft) == 0 {
		return fmt.Errorf("agent report reply is missing draft object")
	}
	mime, ok := reply["mime"].(string)
	if !ok {
		return fmt.Errorf("agent report reply is missing mime output")
	}

	normalizedMIME := normalizeReplyMIME(mime)
	reply["mime"] = normalizedMIME
	if err := writeJSON(filepath.Join(outputDir, "agent-report.json"), reportMap); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outputDir, "reply.draft.json"), draft); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outputDir, "reply.mime.txt"), []byte(strings.TrimRight(normalizedMIME, "\n")+"\n"), 0o644)
}

func runCheckMode(opts options) error {
	tempRoot, err := os.MkdirTemp("", "mailcli-local-thread-demo-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempRoot)

	tempOpts := opts
	tempOpts.outputDir = filepath.Join(tempRoot, "generated")
	tempOpts.index = filepath.Join(tempRoot, "index.json")
	tempOpts.check = false
	if err := generateArtifacts(tempOpts, tempOpts.outputDir); err != nil {
		return err
	}

	mismatches, err := compareArtifactDirs(tempOpts.outputDir, opts.outputDir)
	if err != nil {
		return err
	}
	if len(mismatches) > 0 {
		for _, mismatch := range mismatches {
			fmt.Fprintln(os.Stderr, mismatch)
		}
		return fmt.Errorf("local-thread-demo artifacts are out of date")
	}

	fmt.Println("local-thread-demo artifacts are up to date")
	return nil
}

func buildSyncCommand(opts options) []string {
	command := []string{
		opts.mailcliBin,
		"sync",
		"--format",
		"json",
		"--config",
		opts.config,
		"--account",
		opts.account,
		"--index",
		opts.index,
		"--limit",
		fmt.Sprint(resolveSyncLimit(opts)),
	}
	if opts.mailbox != "" {
		command = append(command, "--mailbox", opts.mailbox)
	}
	return command
}

func buildThreadsCommand(opts options) []string {
	command := []string{
		opts.mailcliBin,
		"threads",
		"--format",
		"json",
		"--index",
		opts.index,
		"--account",
		opts.account,
		"--limit",
		fmt.Sprint(opts.threadLimit),
	}
	if opts.mailbox != "" {
		command = append(command, "--mailbox", opts.mailbox)
	}
	command = append(command, opts.query)
	return command
}

func buildThreadCommand(opts options, threadID string) []string {
	command := []string{
		opts.mailcliBin,
		"thread",
		"--format",
		"json",
		"--index",
		opts.index,
		"--account",
		opts.account,
		"--limit",
		fmt.Sprint(opts.threadMessageLimit),
	}
	if opts.mailbox != "" {
		command = append(command, "--mailbox", opts.mailbox)
	}
	command = append(command, threadID)
	return command
}

func buildAgentCommand(opts options) []string {
	command := []string{
		"go",
		"-C",
		exampleRepoRoot(),
		"run",
		"./examples/go/agent_thread_assistant",
		"--mailcli-bin",
		resolveMaybePath(opts.mailcliBin, opts.workdir),
		"--config",
		resolvePath(opts.config, opts.workdir),
		"--account",
		opts.account,
		"--index",
		resolvePath(opts.index, opts.workdir),
		"--sync-limit",
		fmt.Sprint(resolveSyncLimit(opts)),
		"--thread-limit",
		fmt.Sprint(opts.threadLimit),
		"--thread-message-limit",
		fmt.Sprint(opts.threadMessageLimit),
		"--query",
		opts.query,
		"--from-address",
		opts.fromAddress,
		"--reply-text",
		opts.replyText,
	}
	if opts.mailbox != "" {
		command = append(command, "--mailbox", opts.mailbox)
	}
	return command
}

func exampleRepoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func resolveSyncLimit(opts options) int {
	if opts.hasSyncLimit {
		return opts.syncLimit
	}

	fixtureRoot, err := discoverFixtureRoot(opts.config, opts.account, opts.workdir)
	if err != nil || fixtureRoot == "" {
		return 20
	}
	return countEMLFiles(fixtureRoot)
}

func discoverFixtureRoot(configPath, accountName, workdir string) (string, error) {
	configFile := resolvePath(configPath, workdir)
	cfg, err := config.Load(configFile)
	if err != nil {
		return "", err
	}
	accountCfg, err := cfg.ResolveAccount(accountName)
	if err != nil {
		return "", err
	}
	if accountCfg.Driver != "dir" || accountCfg.Path == "" {
		return "", nil
	}
	info, err := os.Stat(accountCfg.Path)
	if err != nil || !info.IsDir() {
		return "", err
	}
	return accountCfg.Path, nil
}

func resolvePath(path, workdir string) string {
	if filepath.IsAbs(path) {
		return path
	}
	base := workdir
	if base == "" {
		base, _ = os.Getwd()
	}
	return filepath.Clean(filepath.Join(base, path))
}

func resolveMaybePath(path, workdir string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	if !strings.ContainsAny(path, `/\`) && !strings.HasPrefix(path, ".") {
		return path
	}
	return resolvePath(path, workdir)
}

func countEMLFiles(root string) int {
	count := 0
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.EqualFold(filepath.Ext(path), ".eml") {
			count++
		}
		return nil
	})
	return count
}

func resetIndexFile(path string) {
	_ = os.Remove(path)
	if filepath.Ext(path) == ".json" {
		_ = os.Remove(strings.TrimSuffix(path, ".json") + ".db")
	}
}

func normalizeDemoJSON(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			switch {
			case key == "indexed_at":
				if _, ok := item.(string); ok {
					out[key] = canonicalIndexedAt
					continue
				}
			case key == "index" || key == "index_path":
				if _, ok := item.(string); ok {
					out[key] = canonicalIndexPath
					continue
				}
			case key == "config":
				if _, ok := item.(string); ok {
					out[key] = canonicalConfig
					continue
				}
			}
			out[key] = normalizeDemoJSON(item)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = normalizeDemoJSON(item)
		}
		return out
	default:
		return value
	}
}

func normalizeReplyMIME(mime string) string {
	mime = strings.ReplaceAll(mime, "\r\n", "\n")
	mime = strings.ReplaceAll(mime, "\r", "\n")
	lines := strings.Split(mime, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "Message-ID: ") {
			lines[i] = "Message-ID: " + canonicalMessageID
			continue
		}
		if strings.HasPrefix(line, "Date: ") {
			lines[i] = "Date: " + canonicalMIMEDate
		}
	}
	return strings.Join(lines, "\n")
}

func writeJSON(path string, value any) error {
	var buf bytes.Buffer
	if err := agent.WriteJSON(&buf, value); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func compareArtifactDirs(generated, target string) ([]string, error) {
	generatedEntries, err := os.ReadDir(generated)
	if err != nil {
		return nil, err
	}
	targetEntries, err := os.ReadDir(target)
	if err != nil {
		return nil, err
	}

	generatedFiles := fileSet(generatedEntries)
	targetFiles := fileSet(targetEntries)
	mismatches := []string{}

	for name := range generatedFiles {
		if !targetFiles[name] {
			mismatches = append(mismatches, "missing artifact: "+filepath.Join(target, name))
			continue
		}
		equal, err := filesEqual(filepath.Join(generated, name), filepath.Join(target, name))
		if err != nil {
			return nil, err
		}
		if !equal {
			mismatches = append(mismatches, "artifact drift: "+filepath.Join(target, name))
		}
	}

	for name := range targetFiles {
		if !generatedFiles[name] {
			mismatches = append(mismatches, "unexpected artifact: "+filepath.Join(target, name))
		}
	}
	return mismatches, nil
}

func fileSet(entries []os.DirEntry) map[string]bool {
	out := map[string]bool{}
	for _, entry := range entries {
		if !entry.IsDir() {
			out[entry.Name()] = true
		}
	}
	return out
}

func filesEqual(left, right string) (bool, error) {
	leftBytes, err := os.ReadFile(left)
	if err != nil {
		return false, err
	}
	rightBytes, err := os.ReadFile(right)
	if err != nil {
		return false, err
	}
	return reflect.DeepEqual(leftBytes, rightBytes), nil
}
