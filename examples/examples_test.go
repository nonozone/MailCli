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
