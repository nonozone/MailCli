package examples_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentInboxAssistantCapturesVerificationCode(t *testing.T) {
	python := requirePython(t)
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)

	cmd := exec.Command(
		python,
		filepath.Join(repoRoot, "examples/python/agent_inbox_assistant.py"),
		"--mailcli-bin", mailcliBin,
		"--email", filepath.Join(repoRoot, "testdata/emails/verification.eml"),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("agent example failed: %v\n%s", err, string(output))
	}

	var report map[string]any
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("expected json output: %v\n%s", err, string(output))
	}

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
	python := requirePython(t)
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)

	cmd := exec.Command(
		python,
		filepath.Join(repoRoot, "examples/python/agent_inbox_assistant.py"),
		"--mailcli-bin", mailcliBin,
		"--email", filepath.Join(repoRoot, "testdata/emails/plaintext.eml"),
		"--from-address", "support@nono.im",
		"--reply-text", "Thanks for your email.",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("agent example failed: %v\n%s", err, string(output))
	}

	var report map[string]any
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("expected json output: %v\n%s", err, string(output))
	}

	analysis := mustMap(t, report["analysis"])
	if analysis["decision"] != "draft_reply" {
		t.Fatalf("expected draft_reply decision, got %#v", analysis["decision"])
	}

	reply := mustMap(t, report["reply"])
	draft := mustMap(t, reply["draft"])
	if draft["reply_to_message_id"] != "<plain-123@example.com>" {
		t.Fatalf("expected reply_to_message_id to be propagated, got %#v", draft["reply_to_message_id"])
	}

	mime, ok := reply["mime"].(string)
	if !ok {
		t.Fatalf("expected reply mime string, got %#v", reply["mime"])
	}
	if !strings.Contains(mime, "In-Reply-To: <plain-123@example.com>") {
		t.Fatalf("expected reply mime to contain In-Reply-To header, got %q", mime)
	}
	if !strings.Contains(mime, "To: Example Sender <sender@example.com>") {
		t.Fatalf("expected reply mime to target original sender, got %q", mime)
	}
}

func TestAgentInboxAssistantSupportsFixtureDirConfig(t *testing.T) {
	python := requirePython(t)
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)

	cmd := exec.Command(
		python,
		filepath.Join(repoRoot, "examples/python/agent_inbox_assistant.py"),
		"--mailcli-bin", mailcliBin,
		"--config", filepath.Join(repoRoot, "examples/config/fixtures-dir.yaml"),
		"--account", "fixtures",
		"--message-id", "invoice.eml",
	)
	cmd.Dir = filepath.Join(repoRoot, "examples")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("agent example with fixture dir config failed: %v\n%s", err, string(output))
	}

	var report map[string]any
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("expected json output: %v\n%s", err, string(output))
	}

	message := mustMap(t, report["message"])
	meta := mustMap(t, message["meta"])
	if meta["subject"] != "Your April invoice is ready" {
		t.Fatalf("expected invoice subject from dir-backed config, got %#v", meta["subject"])
	}
}

func TestAgentThreadAssistantBuildsReplyDryRunFromLocalThread(t *testing.T) {
	python := requirePython(t)
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)
	configPath := writeTempFile(t, "config.yaml", "current_account: demo\naccounts:\n  - name: demo\n    driver: stub\n")
	indexPath := filepath.Join(t.TempDir(), "index.json")

	cmd := exec.Command(
		python,
		filepath.Join(repoRoot, "examples/python/agent_thread_assistant.py"),
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

	var report map[string]any
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("expected json output: %v\n%s", err, string(output))
	}

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

	mime, ok := reply["mime"].(string)
	if !ok {
		t.Fatalf("expected reply mime string, got %#v", reply["mime"])
	}
	if !strings.Contains(mime, "In-Reply-To: <stub-invoice@example.com>") {
		t.Fatalf("expected reply mime to contain In-Reply-To header, got %q", mime)
	}
}

func TestAgentThreadAssistantSupportsFixtureDirConfig(t *testing.T) {
	python := requirePython(t)
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)
	indexPath := filepath.Join(t.TempDir(), "index.json")

	cmd := exec.Command(
		python,
		filepath.Join(repoRoot, "examples/python/agent_thread_assistant.py"),
		"--mailcli-bin", mailcliBin,
		"--config", filepath.Join(repoRoot, "examples/config/fixtures-dir.yaml"),
		"--account", "fixtures",
		"--index", indexPath,
		"--sync-limit", "20",
		"--query", "invoice",
	)
	cmd.Dir = t.TempDir()

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("thread agent example with fixture dir config failed: %v\n%s", err, string(output))
	}

	var report map[string]any
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("expected json output: %v\n%s", err, string(output))
	}

	syncResult := mustMap(t, report["sync"])
	if syncResult["indexed_count"] != float64(13) {
		t.Fatalf("expected fixture sync to index all repository fixtures, got %#v", syncResult["indexed_count"])
	}

	selection := mustMap(t, report["selection"])
	if selection["thread_id"] != "<invoice-123@example.com>" {
		t.Fatalf("expected invoice fixture thread to be selected, got %#v", selection["thread_id"])
	}
}

func TestAgentThreadAssistantBuildsLocalOnlyReplyDraftWithoutConfig(t *testing.T) {
	python := requirePython(t)
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)
	indexPath := writeTempFile(t, "index.json", `{
  "version": 1,
  "messages": [
    {
      "account": "demo",
      "mailbox": "INBOX",
      "id": "msg-root",
      "indexed_at": "2026-03-27T08:00:00Z",
      "message": {
        "id": "msg-root",
        "meta": {
          "subject": "Project update",
          "date": "2026-03-27T08:00:00Z",
          "message_id": "<root@example.com>",
          "from": {
            "name": "Example Sender",
            "address": "sender@example.com"
          }
        },
        "content": {
          "snippet": "Initial update",
          "body_md": "Initial update"
        }
      }
    }
  ]
}`)

	cmd := exec.Command(
		python,
		filepath.Join(repoRoot, "examples/python/agent_thread_assistant.py"),
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

	var report map[string]any
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("expected json output: %v\n%s", err, string(output))
	}

	reply := mustMap(t, report["reply"])
	draft := mustMap(t, reply["draft"])
	if draft["reply_to_message_id"] != "<root@example.com>" {
		t.Fatalf("expected local-only draft to use reply_to_message_id, got %#v", draft["reply_to_message_id"])
	}
	if _, ok := draft["reply_to_id"]; ok {
		t.Fatalf("expected local-only draft to avoid reply_to_id, got %#v", draft["reply_to_id"])
	}

	mime, ok := reply["mime"].(string)
	if !ok {
		t.Fatalf("expected reply mime string, got %#v", reply["mime"])
	}
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
	python := requirePython(t)
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)
	indexPath := writeTempFile(t, "index.json", `{
  "version": 1,
  "messages": [
    {
      "account": "demo",
      "mailbox": "INBOX",
      "id": "msg-root",
      "indexed_at": "2026-03-27T08:00:00Z",
      "message": {
        "id": "msg-root",
        "meta": {
          "subject": "Project update",
          "date": "2026-03-27T08:00:00Z",
          "message_id": "<root@example.com>",
          "from": {
            "name": "Older Sender",
            "address": "older@example.com"
          }
        },
        "content": {
          "snippet": "Initial update",
          "body_md": "Initial update"
        }
      }
    },
    {
      "account": "demo",
      "mailbox": "INBOX",
      "id": "msg-reply",
      "indexed_at": "2026-03-27T09:00:00Z",
      "message": {
        "id": "msg-reply",
        "meta": {
          "subject": "Re: Project update",
          "date": "2026-03-27T09:00:00Z",
          "message_id": "<reply@example.com>",
          "in_reply_to": "<root@example.com>",
          "references": [
            "<root@example.com>"
          ],
          "from": {
            "name": "Latest Sender",
            "address": "latest@example.com"
          }
        },
        "content": {
          "snippet": "Latest update",
          "body_md": "Latest update"
        }
      }
    }
  ]
}`)

	cmd := exec.Command(
		python,
		filepath.Join(repoRoot, "examples/python/agent_thread_assistant.py"),
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

	var report map[string]any
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("expected json output: %v\n%s", err, string(output))
	}

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
	python := requirePython(t)
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)
	indexPath := writeTempFile(t, "index.json", `{
  "version": 1,
  "messages": [
    {
      "account": "demo",
      "mailbox": "INBOX",
      "id": "msg-root",
      "indexed_at": "2026-03-27T08:00:00Z",
      "message": {
        "id": "msg-root",
        "meta": {
          "subject": "Project update",
          "date": "2026-03-27T08:00:00Z",
          "message_id": "<root@example.com>",
          "from": {
            "name": "Example Sender",
            "address": "sender@example.com"
          }
        },
        "content": {
          "snippet": "Initial update",
          "body_md": "Initial update"
        }
      }
    }
  ]
}`)
	payloadPath := filepath.Join(t.TempDir(), "thread_payload.json")
	providerPath := writeTempFile(t, "thread_provider.py", `import json
import os
import sys

payload = json.load(sys.stdin)
with open(os.environ["THREAD_PROVIDER_PAYLOAD_PATH"], "w", encoding="utf-8") as fh:
    json.dump(payload, fh)
latest = payload["latest_message"]["message"]
print(json.dumps({
  "decision": "draft_reply",
  "summary": latest["content"]["snippet"],
  "reply_text": "Handled by thread provider."
}))
`)

	cmd := exec.Command(
		python,
		filepath.Join(repoRoot, "examples/python/agent_thread_assistant.py"),
		"--mailcli-bin", mailcliBin,
		"--index", indexPath,
		"--skip-sync",
		"--thread-id", "<root@example.com>",
		"--from-address", "support@nono.im",
		"--agent-provider", "external",
		"--provider-command", python,
		"--provider-arg", providerPath,
	)
	cmd.Env = append(os.Environ(), "THREAD_PROVIDER_PAYLOAD_PATH="+payloadPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("thread agent external provider failed: %v\n%s", err, string(output))
	}

	var report map[string]any
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("expected json output: %v\n%s", err, string(output))
	}

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
	var payload map[string]any
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		t.Fatalf("expected payload json: %v", err)
	}
	selectionPayload := mustMap(t, payload["selection"])
	if selectionPayload["thread_id"] != "<root@example.com>" {
		t.Fatalf("expected selection thread id in payload, got %#v", selectionPayload["thread_id"])
	}
	if payload["wants_reply"] != false {
		t.Fatalf("expected wants_reply false without explicit reply text, got %#v", payload["wants_reply"])
	}
	summaries := mustSlice(t, payload["thread_summaries"])
	if len(summaries) != 1 {
		t.Fatalf("expected one thread summary in payload, got %#v", summaries)
	}
	threadMessages := mustSlice(t, payload["thread_messages"])
	if len(threadMessages) != 1 {
		t.Fatalf("expected one thread message in payload, got %#v", threadMessages)
	}
}

func TestAgentThreadAssistantWorksWithTemplateExternalProvider(t *testing.T) {
	python := requirePython(t)
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)
	indexPath := writeTempFile(t, "index.json", `{
  "version": 1,
  "messages": [
    {
      "account": "demo",
      "mailbox": "INBOX",
      "id": "msg-root",
      "indexed_at": "2026-03-27T08:00:00Z",
      "message": {
        "id": "msg-root",
        "meta": {
          "subject": "Project update",
          "date": "2026-03-27T08:00:00Z",
          "message_id": "<root@example.com>",
          "from": {
            "name": "Example Sender",
            "address": "sender@example.com"
          }
        },
        "content": {
          "snippet": "Initial update",
          "body_md": "Initial update"
        }
      }
    }
  ]
}`)

	cmd := exec.Command(
		python,
		filepath.Join(repoRoot, "examples/python/agent_thread_assistant.py"),
		"--mailcli-bin", mailcliBin,
		"--index", indexPath,
		"--skip-sync",
		"--thread-id", "<root@example.com>",
		"--from-address", "support@nono.im",
		"--agent-provider", "external",
		"--provider-command", python,
		"--provider-arg", filepath.Join(repoRoot, "examples/providers/template_external_provider.py"),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("thread agent template provider failed: %v\n%s", err, string(output))
	}

	var report map[string]any
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("expected json output: %v\n%s", err, string(output))
	}

	analysis := mustMap(t, report["analysis"])
	if analysis["provider"] != "external" {
		t.Fatalf("expected external provider metadata, got %#v", analysis["provider"])
	}
	if analysis["decision"] != "review" {
		t.Fatalf("expected template provider to default to review, got %#v", analysis["decision"])
	}
}

func TestAgentThreadAssistantTemplateProviderSupportsReplyBranch(t *testing.T) {
	python := requirePython(t)
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)
	indexPath := writeTempFile(t, "index.json", `{
  "version": 1,
  "messages": [
    {
      "account": "demo",
      "mailbox": "INBOX",
      "id": "msg-root",
      "indexed_at": "2026-03-27T08:00:00Z",
      "message": {
        "id": "msg-root",
        "meta": {
          "subject": "Project update",
          "date": "2026-03-27T08:00:00Z",
          "message_id": "<root@example.com>",
          "from": {
            "name": "Example Sender",
            "address": "sender@example.com"
          }
        },
        "content": {
          "snippet": "Initial update",
          "body_md": "Initial update"
        }
      }
    }
  ]
}`)

	cmd := exec.Command(
		python,
		filepath.Join(repoRoot, "examples/python/agent_thread_assistant.py"),
		"--mailcli-bin", mailcliBin,
		"--index", indexPath,
		"--skip-sync",
		"--thread-id", "<root@example.com>",
		"--from-address", "support@nono.im",
		"--reply-text", "Please draft a reply.",
		"--agent-provider", "external",
		"--provider-command", python,
		"--provider-arg", filepath.Join(repoRoot, "examples/providers/template_external_provider.py"),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("thread agent template provider reply branch failed: %v\n%s", err, string(output))
	}

	var report map[string]any
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("expected json output: %v\n%s", err, string(output))
	}

	analysis := mustMap(t, report["analysis"])
	if analysis["decision"] != "draft_reply" {
		t.Fatalf("expected template provider to request draft reply, got %#v", analysis["decision"])
	}
}

func TestAgentThreadAssistantRejectsInvalidExternalProviderResponse(t *testing.T) {
	python := requirePython(t)
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)
	indexPath := writeTempFile(t, "index.json", `{
  "version": 1,
  "messages": [
    {
      "account": "demo",
      "mailbox": "INBOX",
      "id": "msg-root",
      "indexed_at": "2026-03-27T08:00:00Z",
      "message": {
        "id": "msg-root",
        "meta": {
          "subject": "Project update",
          "date": "2026-03-27T08:00:00Z",
          "message_id": "<root@example.com>",
          "from": {
            "name": "Example Sender",
            "address": "sender@example.com"
          }
        },
        "content": {
          "snippet": "Initial update",
          "body_md": "Initial update"
        }
      }
    }
  ]
}`)
	providerPath := writeTempFile(t, "thread_provider_invalid.py", `import json
print(json.dumps({"summary": "missing decision"}))
`)

	cmd := exec.Command(
		python,
		filepath.Join(repoRoot, "examples/python/agent_thread_assistant.py"),
		"--mailcli-bin", mailcliBin,
		"--index", indexPath,
		"--skip-sync",
		"--thread-id", "<root@example.com>",
		"--agent-provider", "external",
		"--provider-command", python,
		"--provider-arg", providerPath,
	)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected invalid thread provider response to fail, got success: %s", string(output))
	}
	if !strings.Contains(string(output), "external provider response must include a non-empty decision") {
		t.Fatalf("expected contract error, got %s", string(output))
	}
}

func TestAgentThreadAssistantRejectsInvalidExternalProviderJSON(t *testing.T) {
	python := requirePython(t)
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)
	indexPath := writeTempFile(t, "index.json", `{
  "version": 1,
  "messages": [
    {
      "account": "demo",
      "mailbox": "INBOX",
      "id": "msg-root",
      "indexed_at": "2026-03-27T08:00:00Z",
      "message": {
        "id": "msg-root",
        "meta": {
          "subject": "Project update",
          "date": "2026-03-27T08:00:00Z",
          "message_id": "<root@example.com>",
          "from": {
            "name": "Example Sender",
            "address": "sender@example.com"
          }
        },
        "content": {
          "snippet": "Initial update",
          "body_md": "Initial update"
        }
      }
    }
  ]
}`)
	providerPath := writeTempFile(t, "thread_provider_bad_json.py", `print("not-json")`)

	cmd := exec.Command(
		python,
		filepath.Join(repoRoot, "examples/python/agent_thread_assistant.py"),
		"--mailcli-bin", mailcliBin,
		"--index", indexPath,
		"--skip-sync",
		"--thread-id", "<root@example.com>",
		"--agent-provider", "external",
		"--provider-command", python,
		"--provider-arg", providerPath,
	)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected invalid thread provider json to fail, got success: %s", string(output))
	}
	if !strings.Contains(string(output), "external provider returned invalid JSON") {
		t.Fatalf("expected invalid json contract error, got %s", string(output))
	}
}

func TestAgentThreadAssistantRejectsUnknownExternalDecision(t *testing.T) {
	python := requirePython(t)
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)
	indexPath := writeTempFile(t, "index.json", `{
  "version": 1,
  "messages": [
    {
      "account": "demo",
      "mailbox": "INBOX",
      "id": "msg-root",
      "indexed_at": "2026-03-27T08:00:00Z",
      "message": {
        "id": "msg-root",
        "meta": {
          "subject": "Project update",
          "date": "2026-03-27T08:00:00Z",
          "message_id": "<root@example.com>",
          "from": {
            "name": "Example Sender",
            "address": "sender@example.com"
          }
        },
        "content": {
          "snippet": "Initial update",
          "body_md": "Initial update"
        }
      }
    }
  ]
}`)
	providerPath := writeTempFile(t, "thread_provider_unknown.py", `import json
print(json.dumps({"decision": "archive_now", "summary": "unsupported"}))
`)

	cmd := exec.Command(
		python,
		filepath.Join(repoRoot, "examples/python/agent_thread_assistant.py"),
		"--mailcli-bin", mailcliBin,
		"--index", indexPath,
		"--skip-sync",
		"--thread-id", "<root@example.com>",
		"--agent-provider", "external",
		"--provider-command", python,
		"--provider-arg", providerPath,
	)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected unknown thread provider decision to fail, got success: %s", string(output))
	}
	if !strings.Contains(string(output), "external provider decision must be one of") {
		t.Fatalf("expected decision enum error, got %s", string(output))
	}
}

func TestAgentThreadAssistantRejectsInvalidReplyTextType(t *testing.T) {
	python := requirePython(t)
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)
	indexPath := writeTempFile(t, "index.json", `{
  "version": 1,
  "messages": [
    {
      "account": "demo",
      "mailbox": "INBOX",
      "id": "msg-root",
      "indexed_at": "2026-03-27T08:00:00Z",
      "message": {
        "id": "msg-root",
        "meta": {
          "subject": "Project update",
          "date": "2026-03-27T08:00:00Z",
          "message_id": "<root@example.com>",
          "from": {
            "name": "Example Sender",
            "address": "sender@example.com"
          }
        },
        "content": {
          "snippet": "Initial update",
          "body_md": "Initial update"
        }
      }
    }
  ]
}`)
	providerPath := writeTempFile(t, "thread_provider_bad_reply_text.py", `import json
print(json.dumps({"decision": "draft_reply", "summary": "bad", "reply_text": 123}))
`)

	cmd := exec.Command(
		python,
		filepath.Join(repoRoot, "examples/python/agent_thread_assistant.py"),
		"--mailcli-bin", mailcliBin,
		"--index", indexPath,
		"--skip-sync",
		"--thread-id", "<root@example.com>",
		"--agent-provider", "external",
		"--provider-command", python,
		"--provider-arg", providerPath,
	)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected invalid thread provider reply_text to fail, got success: %s", string(output))
	}
	if !strings.Contains(string(output), "external provider reply_text must be a string when present") {
		t.Fatalf("expected reply_text contract error, got %s", string(output))
	}
}

func TestAgentInboxAssistantUsesExternalProvider(t *testing.T) {
	python := requirePython(t)
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)
	providerPath := writeTempFile(t, "provider.py", `import json
import sys

payload = json.load(sys.stdin)
message = payload["message"]
print(json.dumps({
  "decision": "draft_reply",
  "summary": message["content"]["snippet"],
  "reply_text": "Handled by external provider."
}))
`)

	cmd := exec.Command(
		python,
		filepath.Join(repoRoot, "examples/python/agent_inbox_assistant.py"),
		"--mailcli-bin", mailcliBin,
		"--email", filepath.Join(repoRoot, "testdata/emails/plaintext.eml"),
		"--from-address", "support@nono.im",
		"--agent-provider", "external",
		"--provider-command", python,
		"--provider-arg", providerPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("agent example failed: %v\n%s", err, string(output))
	}

	var report map[string]any
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("expected json output: %v\n%s", err, string(output))
	}

	analysis := mustMap(t, report["analysis"])
	if analysis["decision"] != "draft_reply" {
		t.Fatalf("expected provider-driven draft_reply decision, got %#v", analysis["decision"])
	}
	if analysis["provider"] != "external" {
		t.Fatalf("expected provider metadata, got %#v", analysis["provider"])
	}

	reply := mustMap(t, report["reply"])
	draft := mustMap(t, reply["draft"])
	if draft["body_text"] != "Handled by external provider." {
		t.Fatalf("expected provider reply text, got %#v", draft["body_text"])
	}
}

func TestAgentInboxAssistantRejectsInvalidExternalProviderResponse(t *testing.T) {
	python := requirePython(t)
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)
	providerPath := writeTempFile(t, "provider_invalid.py", `import json
print(json.dumps({"summary": "missing decision"}))
`)

	cmd := exec.Command(
		python,
		filepath.Join(repoRoot, "examples/python/agent_inbox_assistant.py"),
		"--mailcli-bin", mailcliBin,
		"--email", filepath.Join(repoRoot, "testdata/emails/plaintext.eml"),
		"--agent-provider", "external",
		"--provider-command", python,
		"--provider-arg", providerPath,
	)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected invalid provider response to fail, got success: %s", string(output))
	}
	if !strings.Contains(string(output), "external provider response must include a non-empty decision") {
		t.Fatalf("expected contract error, got %s", string(output))
	}
}

func TestAgentInboxAssistantRejectsInvalidExternalProviderJSON(t *testing.T) {
	python := requirePython(t)
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)
	providerPath := writeTempFile(t, "provider_bad_json.py", `print("not-json")`)

	cmd := exec.Command(
		python,
		filepath.Join(repoRoot, "examples/python/agent_inbox_assistant.py"),
		"--mailcli-bin", mailcliBin,
		"--email", filepath.Join(repoRoot, "testdata/emails/plaintext.eml"),
		"--agent-provider", "external",
		"--provider-command", python,
		"--provider-arg", providerPath,
	)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected invalid provider json to fail, got success: %s", string(output))
	}
	if !strings.Contains(string(output), "external provider returned invalid JSON") {
		t.Fatalf("expected invalid json contract error, got %s", string(output))
	}
}

func TestAgentInboxAssistantRejectsUnknownExternalDecision(t *testing.T) {
	python := requirePython(t)
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)
	providerPath := writeTempFile(t, "provider_unknown.py", `import json
print(json.dumps({"decision": "archive_now", "summary": "unsupported"}))
`)

	cmd := exec.Command(
		python,
		filepath.Join(repoRoot, "examples/python/agent_inbox_assistant.py"),
		"--mailcli-bin", mailcliBin,
		"--email", filepath.Join(repoRoot, "testdata/emails/plaintext.eml"),
		"--agent-provider", "external",
		"--provider-command", python,
		"--provider-arg", providerPath,
	)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected unknown decision to fail, got success: %s", string(output))
	}
	if !strings.Contains(string(output), "external provider decision must be one of") {
		t.Fatalf("expected decision enum error, got %s", string(output))
	}
}

func TestAgentInboxAssistantRejectsNonObjectExternalProviderJSON(t *testing.T) {
	python := requirePython(t)
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)
	providerPath := writeTempFile(t, "provider_array_json.py", `print("[]")`)

	cmd := exec.Command(
		python,
		filepath.Join(repoRoot, "examples/python/agent_inbox_assistant.py"),
		"--mailcli-bin", mailcliBin,
		"--email", filepath.Join(repoRoot, "testdata/emails/plaintext.eml"),
		"--agent-provider", "external",
		"--provider-command", python,
		"--provider-arg", providerPath,
	)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-object provider json to fail, got success: %s", string(output))
	}
	if !strings.Contains(string(output), "external provider must return a JSON object") {
		t.Fatalf("expected object contract error, got %s", string(output))
	}
}

func TestAgentInboxAssistantWorksWithTemplateExternalProvider(t *testing.T) {
	python := requirePython(t)
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)

	cmd := exec.Command(
		python,
		filepath.Join(repoRoot, "examples/python/agent_inbox_assistant.py"),
		"--mailcli-bin", mailcliBin,
		"--email", filepath.Join(repoRoot, "testdata/emails/plaintext.eml"),
		"--from-address", "support@nono.im",
		"--agent-provider", "external",
		"--provider-command", python,
		"--provider-arg", filepath.Join(repoRoot, "examples/providers/template_external_provider.py"),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("template provider failed: %v\n%s", err, string(output))
	}

	var report map[string]any
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("expected json output: %v\n%s", err, string(output))
	}

	analysis := mustMap(t, report["analysis"])
	if analysis["provider"] != "external" {
		t.Fatalf("expected external provider metadata, got %#v", analysis["provider"])
	}
	if analysis["decision"] != "review" {
		t.Fatalf("expected template provider to default to review, got %#v", analysis["decision"])
	}
}

func TestAgentInboxAssistantTemplateProviderCapturesVerificationCodes(t *testing.T) {
	python := requirePython(t)
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)

	cmd := exec.Command(
		python,
		filepath.Join(repoRoot, "examples/python/agent_inbox_assistant.py"),
		"--mailcli-bin", mailcliBin,
		"--email", filepath.Join(repoRoot, "testdata/emails/verification.eml"),
		"--agent-provider", "external",
		"--provider-command", python,
		"--provider-arg", filepath.Join(repoRoot, "examples/providers/template_external_provider.py"),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("template provider verification flow failed: %v\n%s", err, string(output))
	}

	var report map[string]any
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("expected json output: %v\n%s", err, string(output))
	}

	analysis := mustMap(t, report["analysis"])
	if analysis["decision"] != "capture_code" {
		t.Fatalf("expected template provider to capture code, got %#v", analysis["decision"])
	}
	if !strings.Contains(mustString(t, analysis["summary"]), "expires in 600 seconds") {
		t.Fatalf("expected verification summary to mention expiry, got %#v", analysis["summary"])
	}
}

func TestAgentInboxAssistantTemplateProviderEscalatesBounceMail(t *testing.T) {
	python := requirePython(t)
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)

	cmd := exec.Command(
		python,
		filepath.Join(repoRoot, "examples/python/agent_inbox_assistant.py"),
		"--mailcli-bin", mailcliBin,
		"--email", filepath.Join(repoRoot, "testdata/emails/bounce.eml"),
		"--agent-provider", "external",
		"--provider-command", python,
		"--provider-arg", filepath.Join(repoRoot, "examples/providers/template_external_provider.py"),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("template provider bounce flow failed: %v\n%s", err, string(output))
	}

	var report map[string]any
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("expected json output: %v\n%s", err, string(output))
	}

	analysis := mustMap(t, report["analysis"])
	if analysis["decision"] != "escalate_delivery_error" {
		t.Fatalf("expected template provider to escalate bounce mail, got %#v", analysis["decision"])
	}
	if !strings.Contains(mustString(t, analysis["summary"]), "Authentication credentials invalid") {
		t.Fatalf("expected bounce summary to include diagnostic code, got %#v", analysis["summary"])
	}
}

func TestAgentInboxAssistantTemplateProviderSummarizesUnsubscribeActions(t *testing.T) {
	python := requirePython(t)
	repoRoot := repoRoot(t)
	mailcliBin := buildMailcliBinary(t, repoRoot)

	cmd := exec.Command(
		python,
		filepath.Join(repoRoot, "examples/python/agent_inbox_assistant.py"),
		"--mailcli-bin", mailcliBin,
		"--email", filepath.Join(repoRoot, "testdata/emails/unsubscribe_mixed.eml"),
		"--agent-provider", "external",
		"--provider-command", python,
		"--provider-arg", filepath.Join(repoRoot, "examples/providers/template_external_provider.py"),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("template provider unsubscribe flow failed: %v\n%s", err, string(output))
	}

	var report map[string]any
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("expected json output: %v\n%s", err, string(output))
	}

	analysis := mustMap(t, report["analysis"])
	if analysis["decision"] != "review" {
		t.Fatalf("expected template provider to keep unsubscribe mail in review, got %#v", analysis["decision"])
	}
	if mustString(t, analysis["summary"]) != "Subscription email with 2 unsubscribe action(s)." {
		t.Fatalf("expected unsubscribe-aware summary, got %#v", analysis["summary"])
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

func TestOpenAIExternalProviderRequiresAPIKey(t *testing.T) {
	python := requirePython(t)
	repoRoot := repoRoot(t)

	cmd := exec.Command(
		python,
		filepath.Join(repoRoot, "examples/providers/openai_external_provider.py"),
	)
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
	python := requirePython(t)
	repoRoot := repoRoot(t)

	stubDir := t.TempDir()
	requestPath := filepath.Join(stubDir, "request.json")
	stubModule := `import json
import os

class _Response:
    def __init__(self, output_text):
        self.output_text = output_text

class _Responses:
    def create(self, **kwargs):
        with open(os.environ["OPENAI_STUB_REQUEST_PATH"], "w", encoding="utf-8") as fh:
            json.dump(kwargs, fh)
        return _Response(json.dumps({
            "decision": "review",
            "summary": "stubbed openai provider"
        }))

class OpenAI:
    def __init__(self):
        self.responses = _Responses()
`
	stubModulePath := filepath.Join(stubDir, "openai.py")
	if err := os.WriteFile(stubModulePath, []byte(stubModule), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(
		python,
		filepath.Join(repoRoot, "examples/providers/openai_external_provider.py"),
	)
	cmd.Stdin = strings.NewReader(`{"message":{"content":{"snippet":"hello"}},"source":{"mode":"email","value":"x"},"wants_reply":false}`)
	cmd.Env = append(os.Environ(),
		"OPENAI_API_KEY=test-key",
		"OPENAI_MODEL=gpt-5-mini",
		"PYTHONPATH="+stubDir,
		"OPENAI_STUB_REQUEST_PATH="+requestPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("openai provider failed: %v\n%s", err, string(output))
	}

	var result map[string]any
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("expected json output: %v\n%s", err, string(output))
	}
	if result["decision"] != "review" {
		t.Fatalf("expected stubbed decision, got %#v", result["decision"])
	}

	requestBytes, err := os.ReadFile(requestPath)
	if err != nil {
		t.Fatal(err)
	}
	var request map[string]any
	if err := json.Unmarshal(requestBytes, &request); err != nil {
		t.Fatalf("expected request json: %v", err)
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
	python := requirePython(t)
	repoRoot := repoRoot(t)

	stubDir := t.TempDir()
	requestPath := filepath.Join(stubDir, "request.json")
	stubModule := `import json
import os

class _Response:
    def __init__(self, output_text):
        self.output_text = output_text

class _Responses:
    def create(self, **kwargs):
        with open(os.environ["OPENAI_STUB_REQUEST_PATH"], "w", encoding="utf-8") as fh:
            json.dump(kwargs, fh)
        return _Response(json.dumps({
            "decision": "review",
            "summary": "stubbed openai provider"
        }))

class OpenAI:
    def __init__(self):
        self.responses = _Responses()
`
	stubModulePath := filepath.Join(stubDir, "openai.py")
	if err := os.WriteFile(stubModulePath, []byte(stubModule), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(
		python,
		filepath.Join(repoRoot, "examples/providers/openai_external_provider.py"),
	)
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
		"PYTHONPATH="+stubDir,
		"OPENAI_STUB_REQUEST_PATH="+requestPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("openai provider thread mode failed: %v\n%s", err, string(output))
	}

	requestBytes, err := os.ReadFile(requestPath)
	if err != nil {
		t.Fatal(err)
	}
	var request map[string]any
	if err := json.Unmarshal(requestBytes, &request); err != nil {
		t.Fatalf("expected request json: %v", err)
	}
	input := mustSlice(t, request["input"])
	user := mustMap(t, input[1])
	content := mustSlice(t, user["content"])
	part := mustMap(t, content[0])
	payloadText, ok := part["text"].(string)
	if !ok {
		t.Fatalf("expected payload text, got %#v", part["text"])
	}
	var normalized map[string]any
	if err := json.Unmarshal([]byte(payloadText), &normalized); err != nil {
		t.Fatalf("expected normalized payload json: %v", err)
	}
	message := mustMap(t, normalized["message"])
	codes := mustSlice(t, message["codes"])
	if len(codes) != 1 {
		t.Fatalf("expected normalized payload to expose latest message codes, got %#v", codes)
	}
}

func requirePython(t *testing.T) string {
	t.Helper()

	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not available")
	}
	return python
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
