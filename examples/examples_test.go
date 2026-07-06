package examples_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	mailindex "github.com/nonozone/MailCli/internal/index"
	"github.com/nonozone/MailCli/pkg/schema"
)

func TestAgentInboxAssistantCapturesVerificationCode(t *testing.T) {
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)

	cmd := goRunExample(t, repoRoot, "agent_inbox_assistant",
		"--mailcli-bin", mailcliBin,
		"--email", filepath.Join(repoRoot, "testdata/emails/verification.eml"),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("agent example failed: %v\n%s", err, string(output))
	}

	report := decodeObject(t, output)
	analysis := mustMap(t, report["analysis"])
	if analysis["decision"] != "capture_code" {
		t.Fatalf("expected capture_code decision, got %#v", analysis["decision"])
	}

	message := mustMap(t, report["message"])
	codes := mustSlice(t, message["codes"])
	if len(codes) != 1 {
		t.Fatalf("expected one code, got %#v", codes)
	}
	code := mustMap(t, codes[0])
	if code["value"] != "123456" {
		t.Fatalf("expected verification code 123456, got %#v", code["value"])
	}
}

func TestAgentInboxAssistantBuildsReplyDryRun(t *testing.T) {
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)

	cmd := goRunExample(t, repoRoot, "agent_inbox_assistant",
		"--mailcli-bin", mailcliBin,
		"--email", filepath.Join(repoRoot, "testdata/emails/plaintext.eml"),
		"--from-address", "support@nono.im",
		"--reply-text", "Thanks for your email.",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("agent example failed: %v\n%s", err, string(output))
	}

	report := decodeObject(t, output)
	analysis := mustMap(t, report["analysis"])
	if analysis["decision"] != "draft_reply" {
		t.Fatalf("expected draft_reply decision, got %#v", analysis["decision"])
	}

	reply := mustMap(t, report["reply"])
	draft := mustMap(t, reply["draft"])
	if draft["reply_to_message_id"] != "<plain-123@example.com>" {
		t.Fatalf("expected reply_to_message_id to be propagated, got %#v", draft["reply_to_message_id"])
	}

	mime := mustString(t, reply["mime"])
	if !strings.Contains(mime, "In-Reply-To: <plain-123@example.com>") {
		t.Fatalf("expected reply mime to contain In-Reply-To header, got %q", mime)
	}
	if !strings.Contains(mime, "To: Example Sender <sender@example.com>") {
		t.Fatalf("expected reply mime to target original sender, got %q", mime)
	}
}

func TestAgentInboxAssistantSupportsFixtureDirConfig(t *testing.T) {
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)

	cmd := goRunExample(t, repoRoot, "agent_inbox_assistant",
		"--mailcli-bin", mailcliBin,
		"--config", filepath.Join(repoRoot, "examples/config/fixtures-dir.yaml"),
		"--account", "fixtures",
		"--message-id", "invoice.eml",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("agent example with fixture dir config failed: %v\n%s", err, string(output))
	}

	report := decodeObject(t, output)
	message := mustMap(t, report["message"])
	meta := mustMap(t, message["meta"])
	if meta["subject"] != "Your April invoice is ready" {
		t.Fatalf("expected invoice subject from dir-backed config, got %#v", meta["subject"])
	}
}

func TestAgentThreadAssistantBuildsReplyDryRunFromLocalThread(t *testing.T) {
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)
	configPath := writeTempFile(t, "config.yaml", "current_account: demo\naccounts:\n  - name: demo\n    driver: stub\n")
	indexPath := filepath.Join(t.TempDir(), "index.json")

	cmd := goRunExample(t, repoRoot, "agent_thread_assistant",
		"--mailcli-bin", mailcliBin,
		"--config", configPath,
		"--account", "demo",
		"--index", indexPath,
		"--query", "invoice",
		"--from-address", "support@nono.im",
		"--reply-text", "Thanks for your email.",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("thread agent example failed: %v\n%s", err, string(output))
	}

	report := decodeObject(t, output)
	syncResult := mustMap(t, report["sync"])
	if syncResult["indexed_count"] != float64(2) {
		t.Fatalf("expected sync to index two stub messages, got %#v", syncResult["indexed_count"])
	}

	selection := mustMap(t, report["selection"])
	if selection["thread_id"] != "<stub-invoice@example.com>" {
		t.Fatalf("expected invoice thread to be selected, got %#v", selection["thread_id"])
	}

	reply := mustMap(t, report["reply"])
	draft := mustMap(t, reply["draft"])
	if draft["reply_to_id"] != "stub:invoice" {
		t.Fatalf("expected reply_to_id to target latest local message, got %#v", draft["reply_to_id"])
	}

	to := mustSlice(t, draft["to"])
	firstTo := mustMap(t, to[0])
	if firstTo["address"] != "billing@example.com" {
		t.Fatalf("expected reply target to use latest sender, got %#v", firstTo["address"])
	}

	mime := mustString(t, reply["mime"])
	if !strings.Contains(mime, "In-Reply-To: <stub-invoice@example.com>") {
		t.Fatalf("expected reply mime to contain In-Reply-To header, got %q", mime)
	}
}

func TestAgentThreadAssistantSupportsFixtureDirConfig(t *testing.T) {
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)
	indexPath := filepath.Join(t.TempDir(), "index.json")
	expectedFixtures := countFixtureEmails(t, filepath.Join(repoRoot, "testdata", "emails"))

	cmd := goRunExample(t, repoRoot, "agent_thread_assistant",
		"--mailcli-bin", mailcliBin,
		"--config", filepath.Join(repoRoot, "examples/config/fixtures-dir.yaml"),
		"--account", "fixtures",
		"--index", indexPath,
		"--sync-limit", strconv.Itoa(expectedFixtures),
		"--query", "invoice",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("thread agent example with fixture dir config failed: %v\n%s", err, string(output))
	}

	report := decodeObject(t, output)
	syncResult := mustMap(t, report["sync"])
	if syncResult["indexed_count"] != float64(expectedFixtures) {
		t.Fatalf("expected fixture sync to index all repository fixtures, got %#v", syncResult["indexed_count"])
	}

	selection := mustMap(t, report["selection"])
	if selection["thread_id"] != "<invoice-123@example.com>" {
		t.Fatalf("expected invoice fixture thread to be selected, got %#v", selection["thread_id"])
	}
}

func TestAgentThreadAssistantBuildsLocalOnlyReplyDraftWithoutConfig(t *testing.T) {
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)
	indexPath := agentThreadTestIndex(t)

	cmd := goRunExample(t, repoRoot, "agent_thread_assistant",
		"--mailcli-bin", mailcliBin,
		"--index", indexPath,
		"--skip-sync",
		"--thread-id", "<root@example.com>",
		"--from-address", "support@nono.im",
		"--reply-text", "Thanks for your email.",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("thread agent local-only example failed: %v\n%s", err, string(output))
	}

	report := decodeObject(t, output)
	reply := mustMap(t, report["reply"])
	draft := mustMap(t, reply["draft"])
	if draft["reply_to_message_id"] != "<root@example.com>" {
		t.Fatalf("expected local-only draft to use reply_to_message_id, got %#v", draft["reply_to_message_id"])
	}
	if _, ok := draft["reply_to_id"]; ok {
		t.Fatalf("expected local-only draft to avoid reply_to_id, got %#v", draft["reply_to_id"])
	}

	mime := mustString(t, reply["mime"])
	if !strings.Contains(mime, "In-Reply-To: <root@example.com>") {
		t.Fatalf("expected local-only reply mime to contain In-Reply-To header, got %q", mime)
	}
	references := mustSlice(t, draft["references"])
	if len(references) != 1 || references[0] != "<root@example.com>" {
		t.Fatalf("expected local-only draft to carry references, got %#v", references)
	}
	if draft["subject"] != "Project update" {
		t.Fatalf("expected local-only draft to carry subject, got %#v", draft["subject"])
	}
}

func TestAgentThreadAssistantReloadsLatestMessageWhenThreadLimitTruncates(t *testing.T) {
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)
	indexPath := filepath.Join(t.TempDir(), "index.db")
	store := mailindex.NewFileStore(indexPath)
	for _, item := range []mailindex.IndexedMessage{
		{
			Account: "demo", Mailbox: "INBOX", ID: "msg-root",
			IndexedAt: "2026-03-27T08:00:00Z",
			Message: schema.StandardMessage{
				ID: "msg-root",
				Meta: schema.MessageMeta{
					Subject: "Project update", Date: "2026-03-27T08:00:00Z",
					MessageID: "<root@example.com>",
					From:      &schema.Address{Name: "Older Sender", Address: "older@example.com"},
				},
				Content: schema.Content{Snippet: "Initial update", BodyMD: "Initial update"},
			},
		},
		{
			Account: "demo", Mailbox: "INBOX", ID: "msg-reply",
			IndexedAt: "2026-03-27T09:00:00Z",
			Message: schema.StandardMessage{
				ID: "msg-reply",
				Meta: schema.MessageMeta{
					Subject: "Re: Project update", Date: "2026-03-27T09:00:00Z",
					MessageID:  "<reply@example.com>",
					InReplyTo:  "<root@example.com>",
					References: []string{"<root@example.com>"},
					From:       &schema.Address{Name: "Latest Sender", Address: "latest@example.com"},
				},
				Content: schema.Content{Snippet: "Latest update", BodyMD: "Latest update"},
			},
		},
	} {
		if err := store.Upsert(item); err != nil {
			t.Fatalf("upsert failed: %v", err)
		}
	}

	cmd := goRunExample(t, repoRoot, "agent_thread_assistant",
		"--mailcli-bin", mailcliBin,
		"--index", indexPath,
		"--skip-sync",
		"--thread-id", "<root@example.com>",
		"--thread-message-limit", "1",
		"--from-address", "support@nono.im",
		"--reply-text", "Thanks for your email.",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("thread agent truncation example failed: %v\n%s", err, string(output))
	}

	report := decodeObject(t, output)
	latestMessage := mustMap(t, report["latest_message"])
	if latestMessage["id"] != "msg-reply" {
		t.Fatalf("expected latest message to be reloaded, got %#v", latestMessage["id"])
	}

	reply := mustMap(t, report["reply"])
	draft := mustMap(t, reply["draft"])
	if draft["reply_to_message_id"] != "<reply@example.com>" {
		t.Fatalf("expected reply target to use latest message id, got %#v", draft["reply_to_message_id"])
	}
	references := mustSlice(t, draft["references"])
	if len(references) != 2 || references[0] != "<root@example.com>" || references[1] != "<reply@example.com>" {
		t.Fatalf("expected reply references to include the full chain, got %#v", references)
	}

	to := mustSlice(t, draft["to"])
	firstTo := mustMap(t, to[0])
	if firstTo["address"] != "latest@example.com" {
		t.Fatalf("expected reply target to use latest sender, got %#v", firstTo["address"])
	}
}

func TestAgentThreadAssistantUsesExternalProvider(t *testing.T) {
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)
	indexPath := agentThreadTestIndex(t)
	payloadPath := filepath.Join(t.TempDir(), "thread_payload.json")
	providerPath := writeTempFile(t, "thread_provider.go", `package main

import (
	"encoding/json"
	"os"
)

func main() {
	var payload map[string]any
	_ = json.NewDecoder(os.Stdin).Decode(&payload)
	data, _ := json.Marshal(payload)
	_ = os.WriteFile(os.Getenv("THREAD_PROVIDER_PAYLOAD_PATH"), data, 0o644)
	latest := payload["latest_message"].(map[string]any)["message"].(map[string]any)
	content := latest["content"].(map[string]any)
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
		"decision": "draft_reply",
		"summary": content["snippet"],
		"reply_text": "Handled by thread provider.",
	})
}
`)

	cmd := goRunExample(t, repoRoot, "agent_thread_assistant",
		"--mailcli-bin", mailcliBin,
		"--index", indexPath,
		"--skip-sync",
		"--thread-id", "<root@example.com>",
		"--from-address", "support@nono.im",
		"--agent-provider", "external",
		"--provider-command", "go",
		"--provider-arg", "run",
		"--provider-arg", providerPath,
	)
	cmd.Env = append(os.Environ(), "THREAD_PROVIDER_PAYLOAD_PATH="+payloadPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("thread agent external provider failed: %v\n%s", err, string(output))
	}

	report := decodeObject(t, output)
	analysis := mustMap(t, report["analysis"])
	if analysis["decision"] != "draft_reply" {
		t.Fatalf("expected external provider to request draft_reply, got %#v", analysis["decision"])
	}
	if analysis["provider"] != "external" {
		t.Fatalf("expected external provider metadata, got %#v", analysis["provider"])
	}

	reply := mustMap(t, report["reply"])
	draft := mustMap(t, reply["draft"])
	if draft["body_text"] != "Handled by thread provider." {
		t.Fatalf("expected provider reply text, got %#v", draft["body_text"])
	}

	payloadBytes, err := os.ReadFile(payloadPath)
	if err != nil {
		t.Fatal(err)
	}
	payload := decodeObject(t, payloadBytes)
	selectionPayload := mustMap(t, payload["selection"])
	if selectionPayload["thread_id"] != "<root@example.com>" {
		t.Fatalf("expected selection thread id in payload, got %#v", selectionPayload["thread_id"])
	}
	if payload["wants_reply"] != false {
		t.Fatalf("expected wants_reply false without explicit reply text, got %#v", payload["wants_reply"])
	}
}

func TestTemplateExternalProviderBranches(t *testing.T) {
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)

	tests := []struct {
		name     string
		email    string
		decision string
		summary  string
	}{
		{
			name:     "verification",
			email:    "verification.eml",
			decision: "capture_code",
			summary:  "expires in 600 seconds",
		},
		{
			name:     "bounce",
			email:    "bounce.eml",
			decision: "escalate_delivery_error",
			summary:  "Authentication credentials invalid",
		},
		{
			name:     "unsubscribe",
			email:    "unsubscribe_mixed.eml",
			decision: "review",
			summary:  "Subscription email with 2 unsubscribe action(s).",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{
				"--mailcli-bin", mailcliBin,
				"--email", filepath.Join(repoRoot, "testdata/emails", tc.email),
				"--agent-provider", "external",
			}
			args = append(args, templateProviderArgs(repoRoot)...)
			cmd := goRunExample(t, repoRoot, "agent_inbox_assistant", args...)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("template provider flow failed: %v\n%s", err, string(output))
			}

			report := decodeObject(t, output)
			analysis := mustMap(t, report["analysis"])
			if analysis["decision"] != tc.decision {
				t.Fatalf("expected decision %s, got %#v", tc.decision, analysis["decision"])
			}
			if !strings.Contains(mustString(t, analysis["summary"]), tc.summary) {
				t.Fatalf("expected summary to contain %q, got %#v", tc.summary, analysis["summary"])
			}
		})
	}
}

func TestAgentInboxAssistantExternalProviderContractErrors(t *testing.T) {
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)

	tests := []struct {
		name       string
		source     string
		wantOutput string
	}{
		{
			name:       "missing decision",
			source:     `package main; import ("encoding/json"; "os"); func main(){ _ = json.NewEncoder(os.Stdout).Encode(map[string]any{"summary":"missing decision"}) }`,
			wantOutput: "external provider response must include a non-empty decision",
		},
		{
			name:       "bad json",
			source:     `package main; import "fmt"; func main(){ fmt.Print("not-json") }`,
			wantOutput: "external provider returned invalid JSON",
		},
		{
			name:       "unknown decision",
			source:     `package main; import ("encoding/json"; "os"); func main(){ _ = json.NewEncoder(os.Stdout).Encode(map[string]any{"decision":"archive_now","summary":"unsupported"}) }`,
			wantOutput: "external provider decision must be one of",
		},
		{
			name:       "non-object",
			source:     `package main; import "fmt"; func main(){ fmt.Print("[]") }`,
			wantOutput: "external provider must return a JSON object",
		},
		{
			name:       "bad reply_text",
			source:     `package main; import ("encoding/json"; "os"); func main(){ _ = json.NewEncoder(os.Stdout).Encode(map[string]any{"decision":"draft_reply","summary":"bad","reply_text":123}) }`,
			wantOutput: "external provider reply_text must be a string when present",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			providerPath := writeTempFile(t, "provider.go", tc.source)
			cmd := goRunExample(t, repoRoot, "agent_inbox_assistant",
				"--mailcli-bin", mailcliBin,
				"--email", filepath.Join(repoRoot, "testdata/emails/plaintext.eml"),
				"--agent-provider", "external",
				"--provider-command", "go",
				"--provider-arg", "run",
				"--provider-arg", providerPath,
			)
			output, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("expected provider failure, got success: %s", string(output))
			}
			if !strings.Contains(string(output), tc.wantOutput) {
				t.Fatalf("expected %q, got %s", tc.wantOutput, string(output))
			}
		})
	}
}

func TestOutboundPatternArtifactsCompile(t *testing.T) {
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)

	tests := []struct {
		name     string
		args     []string
		contains []string
	}{
		{
			name: "ack reply via reply_to_id",
			args: []string{
				"reply",
				"--config", filepath.Join(repoRoot, "examples/config/fixtures-dir.yaml"),
				"--account", "fixtures",
				"--dry-run",
				filepath.Join(repoRoot, "examples/artifacts/outbound-patterns/ack-reply.draft.json"),
			},
			contains: []string{
				"Subject: Re: Your April invoice is ready",
				"In-Reply-To: <invoice-123@example.com>",
				"queued it for processing",
			},
		},
		{
			name: "minimal reply derives sender and recipient from config and source message",
			args: []string{
				"reply",
				"--config", filepath.Join(repoRoot, "examples/config/fixtures-dir.yaml"),
				"--account", "fixtures",
				"--dry-run",
				filepath.Join(repoRoot, "examples/artifacts/outbound-patterns/minimal-reply.reply.json"),
			},
			contains: []string{
				"From: support@nono.im",
				"To: Billing Team <billing@example.com>",
				"In-Reply-To: <invoice-123@example.com>",
			},
		},
		{
			name: "support follow-up reply preserves multipart output",
			args: []string{
				"reply",
				"--dry-run",
				filepath.Join(repoRoot, "examples/artifacts/outbound-patterns/support-followup.reply.json"),
			},
			contains: []string{
				"Content-Type: multipart/alternative;",
				"In-Reply-To: <plain-123@example.com>",
				"<blockquote>",
			},
		},
		{
			name: "release update draft renders markdown html",
			args: []string{
				"send",
				"--dry-run",
				filepath.Join(repoRoot, "examples/artifacts/outbound-patterns/release-update.draft.json"),
			},
			contains: []string{
				"Subject: Weekly parser status",
				"Content-Type: multipart/alternative;",
				"<table>",
				"<a href=\"https://github.com/nonozone/MailCli/tree/main/docs/en/examples/README.md\">examples index</a>",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(mailcliBin, tc.args...)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("mailcli command failed: %v\n%s", err, string(output))
			}

			text := string(output)
			for _, want := range tc.contains {
				if !strings.Contains(text, want) {
					t.Fatalf("expected output to contain %q, got %s", want, text)
				}
			}
		})
	}
}

func TestRefreshLocalThreadDemoCommand(t *testing.T) {
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)
	outputDir := filepath.Join(t.TempDir(), "local-thread-demo")
	indexPath := filepath.Join(t.TempDir(), "index.json")

	cmd := goRunExample(t, repoRoot, "refresh_local_thread_demo",
		"--mailcli-bin", mailcliBin,
		"--config", filepath.Join(repoRoot, "examples/config/fixtures-dir.yaml"),
		"--account", "fixtures",
		"--index", indexPath,
		"--output-dir", outputDir,
		"--query", "invoice",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("refresh local thread demo command failed: %v\n%s", err, string(output))
	}

	for _, name := range []string{
		"sync.json",
		"threads.json",
		"thread.json",
		"reply.draft.json",
		"reply.mime.txt",
		"agent-report.json",
	} {
		if _, err := os.Stat(filepath.Join(outputDir, name)); err != nil {
			t.Fatalf("expected generated artifact %s: %v", name, err)
		}
	}

	syncResult := readJSONFile(t, filepath.Join(outputDir, "sync.json"))
	expectedFixtures := countFixtureEmails(t, filepath.Join(repoRoot, "testdata", "emails"))
	if syncResult["indexed_count"] != float64(expectedFixtures) {
		t.Fatalf("expected generated sync artifact to match fixture corpus count, got %#v", syncResult["indexed_count"])
	}
	if syncResult["skipped_count"] != float64(0) {
		t.Fatalf("expected generated sync artifact to start from a clean index, got %#v", syncResult["skipped_count"])
	}

	report := readJSONFile(t, filepath.Join(outputDir, "agent-report.json"))
	reportSync := mustMap(t, report["sync"])
	if reportSync["indexed_count"] != float64(expectedFixtures) {
		t.Fatalf("expected generated agent report sync stats to match fixture corpus count, got %#v", reportSync["indexed_count"])
	}
	if reportSync["index_path"] != "/tmp/mailcli-fixtures-index.json" {
		t.Fatalf("expected generated agent report sync path to be normalized, got %#v", reportSync["index_path"])
	}

	replyMimeBytes, err := os.ReadFile(filepath.Join(outputDir, "reply.mime.txt"))
	if err != nil {
		t.Fatal(err)
	}
	replyMime := string(replyMimeBytes)
	if !strings.Contains(replyMime, "Message-ID: <generated@mailcli.local>") {
		t.Fatalf("expected generated reply mime to normalize message id, got %s", replyMime)
	}

	threadBytes, err := os.ReadFile(filepath.Join(outputDir, "thread.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(threadBytes), `"indexed_at": "2026-03-27T15:11:13Z"`) {
		t.Fatalf("expected generated thread artifact to normalize indexed_at, got %s", string(threadBytes))
	}
	source := mustMap(t, report["source"])
	if source["index"] != "/tmp/mailcli-fixtures-index.json" {
		t.Fatalf("expected generated agent report source index to be normalized, got %#v", source["index"])
	}
	if source["config"] != "examples/config/fixtures-dir.yaml" {
		t.Fatalf("expected generated agent report config path to be normalized, got %#v", source["config"])
	}
}

func TestRefreshLocalThreadDemoCommandCheckMode(t *testing.T) {
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)

	cmd := goRunExample(t, repoRoot, "refresh_local_thread_demo",
		"--mailcli-bin", mailcliBin,
		"--config", filepath.Join(repoRoot, "examples/config/fixtures-dir.yaml"),
		"--account", "fixtures",
		"--index", filepath.Join(t.TempDir(), "index.json"),
		"--output-dir", filepath.Join(repoRoot, "examples/artifacts/local-thread-demo"),
		"--query", "invoice",
		"--check",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected check mode to succeed when artifacts are current: %v\n%s", err, string(output))
	}
	if !strings.Contains(string(output), "artifacts are up to date") {
		t.Fatalf("expected check mode confirmation output, got %s", string(output))
	}
}

func TestRefreshLocalThreadDemoCommandUsesSelectedDirAccountForDefaultSyncLimit(t *testing.T) {
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)
	outputDir := filepath.Join(t.TempDir(), "local-thread-demo")
	indexPath := filepath.Join(t.TempDir(), "index.json")

	bogusRoot := filepath.Join(t.TempDir(), "bogus-fixtures")
	if err := os.MkdirAll(bogusRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	bogusMail := "From: Bogus <bogus@example.com>\nTo: nono@example.com\nSubject: Bogus\nMessage-ID: <bogus@example.com>\nDate: Sun, 29 Mar 2026 01:00:00 +0000\nContent-Type: text/plain; charset=UTF-8\n\nBogus corpus entry.\n"
	if err := os.WriteFile(filepath.Join(bogusRoot, "aaa-first.eml"), []byte(bogusMail), 0o644); err != nil {
		t.Fatal(err)
	}

	configPath := writeTempFile(t, "fixtures-multi.yaml", "current_account: other\naccounts:\n  - name: other\n    driver: dir\n    path: "+bogusRoot+"\n    mailbox: INBOX\n  - name: fixtures\n    driver: dir\n    path: "+filepath.Join(repoRoot, "testdata", "emails")+"\n    mailbox: INBOX\n")

	cmd := goRunExample(t, repoRoot, "refresh_local_thread_demo",
		"--mailcli-bin", mailcliBin,
		"--config", configPath,
		"--account", "fixtures",
		"--index", indexPath,
		"--output-dir", outputDir,
		"--query", "invoice",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("refresh local thread demo command with multi-account config failed: %v\n%s", err, string(output))
	}

	syncResult := readJSONFile(t, filepath.Join(outputDir, "sync.json"))
	expectedFixtures := countFixtureEmails(t, filepath.Join(repoRoot, "testdata", "emails"))
	if syncResult["indexed_count"] != float64(expectedFixtures) {
		t.Fatalf("expected selected account fixture corpus count, got %#v", syncResult["indexed_count"])
	}
}

func TestOpenAIExternalProviderRequiresAPIKey(t *testing.T) {
	repoRoot := repoRoot(t)

	cmd := goRunExample(t, repoRoot, "providers/openai_external_provider")
	cmd.Stdin = strings.NewReader(`{"message":{"content":{"snippet":"hello"}},"source":{"mode":"email","value":"x"},"wants_reply":false}`)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected missing key to fail, got success: %s", string(output))
	}
	if !strings.Contains(string(output), "OPENAI_API_KEY is required") {
		t.Fatalf("expected missing key error, got %s", string(output))
	}
}

func TestOpenAIExternalProviderUsesResponsesAPIShape(t *testing.T) {
	repoRoot := repoRoot(t)
	var request map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" {
			t.Fatalf("expected /responses request, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("expected bearer auth, got %q", r.Header.Get("Authorization"))
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"output_text":"{\"decision\":\"review\",\"summary\":\"stubbed openai provider\"}"}`))
	}))
	defer server.Close()

	cmd := goRunExample(t, repoRoot, "providers/openai_external_provider")
	cmd.Stdin = strings.NewReader(`{"message":{"content":{"snippet":"hello"}},"source":{"mode":"email","value":"x"},"wants_reply":false}`)
	cmd.Env = append(os.Environ(),
		"OPENAI_API_KEY=test-key",
		"OPENAI_MODEL=gpt-5-mini",
		"OPENAI_BASE_URL="+server.URL,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("openai provider failed: %v\n%s", err, string(output))
	}

	result := decodeObject(t, output)
	if result["decision"] != "review" {
		t.Fatalf("expected stubbed decision, got %#v", result["decision"])
	}
	if request["model"] != "gpt-5-mini" {
		t.Fatalf("expected OPENAI_MODEL to be used, got %#v", request["model"])
	}
	text := mustMap(t, request["text"])
	format := mustMap(t, text["format"])
	if format["type"] != "json_schema" {
		t.Fatalf("expected structured output request, got %#v", format["type"])
	}
}

func TestOpenAIExternalProviderNormalizesThreadPayload(t *testing.T) {
	repoRoot := repoRoot(t)
	var request map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"output_text":"{\"decision\":\"review\",\"summary\":\"stubbed openai provider\"}"}`))
	}))
	defer server.Close()

	cmd := goRunExample(t, repoRoot, "providers/openai_external_provider")
	cmd.Stdin = strings.NewReader(`{
  "source": {"mode": "local_thread", "thread_id": "<root@example.com>"},
  "selection": {"thread_id": "<root@example.com>", "last_message_id": "imap:uid:123"},
  "thread_messages": [],
  "latest_message": {
    "id": "imap:uid:123",
    "message": {
      "content": {"snippet": "hello from latest"},
      "codes": [{"value": "123456"}]
    }
  },
  "wants_reply": false
}`)
	cmd.Env = append(os.Environ(),
		"OPENAI_API_KEY=test-key",
		"OPENAI_MODEL=gpt-5-mini",
		"OPENAI_BASE_URL="+server.URL,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("openai provider thread mode failed: %v\n%s", err, string(output))
	}

	input := mustSlice(t, request["input"])
	user := mustMap(t, input[1])
	content := mustSlice(t, user["content"])
	part := mustMap(t, content[0])
	payloadText := mustString(t, part["text"])
	normalized := decodeObject(t, []byte(payloadText))
	message := mustMap(t, normalized["message"])
	codes := mustSlice(t, message["codes"])
	if len(codes) != 1 {
		t.Fatalf("expected normalized payload to expose latest message codes, got %#v", codes)
	}
}

func TestRepositoryProvidesGoOnlyLocalThreadDemoMaintenanceEntrypoints(t *testing.T) {
	repoRoot := repoRoot(t)

	makefileBytes, err := os.ReadFile(filepath.Join(repoRoot, "Makefile"))
	if err != nil {
		t.Fatalf("expected repository Makefile: %v", err)
	}
	makefile := string(makefileBytes)
	bytecodeEnv := "PY" + "THON" + "DONTWRITEBYTECODE"
	if !strings.Contains(makefile, "demo-local-thread-refresh:") {
		t.Fatalf("expected Makefile to expose demo-local-thread-refresh target")
	}
	if !strings.Contains(makefile, "demo-local-thread-check:") {
		t.Fatalf("expected Makefile to expose demo-local-thread-check target")
	}
	if !strings.Contains(makefile, "MAILCLI_BIN ?= /tmp/mailcli") {
		t.Fatalf("expected Makefile to keep maintenance builds out of the repository root")
	}
	pythonToken := "py" + "thon"
	if strings.Contains(strings.ToLower(makefile), pythonToken) || strings.Contains(makefile, bytecodeEnv) {
		t.Fatalf("expected Makefile maintenance targets to avoid non-Go runtimes")
	}

	workflowBytes, err := os.ReadFile(filepath.Join(repoRoot, ".github", "workflows", "test.yml"))
	if err != nil {
		t.Fatalf("expected workflow file: %v", err)
	}
	workflow := string(workflowBytes)
	if !strings.Contains(workflow, "make demo-local-thread-check") {
		t.Fatalf("expected CI to run make demo-local-thread-check")
	}
	compileToken := "py_" + "compile"
	if strings.Contains(workflow, "setup-"+pythonToken) || strings.Contains(workflow, compileToken) {
		t.Fatalf("expected CI to avoid non-Go setup and compile checks")
	}
}

func goRunExample(t *testing.T, repoRoot, example string, args ...string) *exec.Cmd {
	t.Helper()

	examplePath := filepath.Join(repoRoot, "examples", "go", filepath.FromSlash(example))
	command := append([]string{"run", examplePath}, args...)
	cmd := exec.Command("go", command...)
	cmd.Dir = repoRoot
	return cmd
}

func templateProviderArgs(repoRoot string) []string {
	return []string{
		"--provider-command", "go",
		"--provider-arg", "run",
		"--provider-arg", filepath.Join(repoRoot, "examples/go/providers/template_external_provider"),
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()

	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func buildMailcliBinary(t *testing.T, repoRoot string) string {
	t.Helper()

	binPath := filepath.Join(t.TempDir(), "mailcli")
	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/mailcli")
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build mailcli binary: %v\n%s", err, string(output))
	}
	return binPath
}

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func decodeObject(t *testing.T, data []byte) map[string]any {
	t.Helper()

	var report map[string]any
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("expected json object: %v\n%s", err, string(data))
	}
	return report
}

func readJSONFile(t *testing.T, path string) map[string]any {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return decodeObject(t, data)
}

func mustMap(t *testing.T, value any) map[string]any {
	t.Helper()

	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected object, got %#v", value)
	}
	return result
}

func mustSlice(t *testing.T, value any) []any {
	t.Helper()

	result, ok := value.([]any)
	if !ok {
		t.Fatalf("expected array, got %#v", value)
	}
	return result
}

func mustString(t *testing.T, value any) string {
	t.Helper()

	result, ok := value.(string)
	if !ok {
		t.Fatalf("expected string, got %#v", value)
	}
	return result
}

func countFixtureEmails(t *testing.T, root string) int {
	t.Helper()

	count := 0
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".eml" {
			count++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected fixture email count to succeed: %v", err)
	}
	return count
}

func agentThreadTestIndex(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "index.db")
	store := mailindex.NewFileStore(path)
	err := store.Upsert(mailindex.IndexedMessage{
		Account:   "demo",
		Mailbox:   "INBOX",
		ID:        "msg-root",
		IndexedAt: "2026-03-27T08:00:00Z",
		Message: schema.StandardMessage{
			ID: "msg-root",
			Meta: schema.MessageMeta{
				Subject:   "Project update",
				Date:      "2026-03-27T08:00:00Z",
				MessageID: "<root@example.com>",
				From: &schema.Address{
					Name:    "Example Sender",
					Address: "sender@example.com",
				},
			},
			Content: schema.Content{
				Snippet: "Initial update",
				BodyMD:  "Initial update",
			},
		},
	})
	if err != nil {
		t.Fatalf("agentThreadTestIndex: upsert failed: %v", err)
	}
	return path
}
