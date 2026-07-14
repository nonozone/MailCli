package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestMCPServeInitializesAndListsSafeTools(t *testing.T) {
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
	}, "\n") + "\n"

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetIn(strings.NewReader(input))
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"mcp", "serve"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected mcp serve to process stdio requests: %v\n%s", err, out.String())
	}

	responses := decodeJSONLines(t, out.Bytes())
	if len(responses) != 2 {
		t.Fatalf("expected initialize and tools/list responses, got %d: %s", len(responses), out.String())
	}
	if responses[0]["id"].(float64) != 1 {
		t.Fatalf("expected initialize response id 1, got %#v", responses[0])
	}

	result := responses[1]["result"].(map[string]any)
	tools := result["tools"].([]any)
	names := make(map[string]bool, len(tools))
	for _, raw := range tools {
		tool := raw.(map[string]any)
		names[tool["name"].(string)] = true
	}
	for _, want := range []string{
		"mailcli_parse",
		"mailcli_list",
		"mailcli_get",
		"mailcli_sync",
		"mailcli_search",
		"mailcli_threads",
		"mailcli_thread",
		"mailcli_triage_message",
		"mailcli_triage_thread",
		"mailcli_config_doctor",
		"mailcli_config_capabilities",
	} {
		if !names[want] {
			t.Fatalf("expected safe MCP tool %q in %#v", want, names)
		}
	}
	for _, dangerous := range []string{"mailcli_send", "mailcli_reply", "mailcli_delete", "mailcli_move", "mailcli_mark"} {
		if names[dangerous] {
			t.Fatalf("dangerous tool %q must not be exposed by default", dangerous)
		}
	}
}

func TestMCPServeCallsParseTool(t *testing.T) {
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":"call-1","method":"tools/call","params":{"name":"mailcli_parse","arguments":{"file":"../testdata/emails/plaintext.eml"}}}`,
	}, "\n") + "\n"

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetIn(strings.NewReader(input))
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"mcp", "serve"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected mcp parse call to succeed: %v\n%s", err, out.String())
	}

	responses := decodeJSONLines(t, out.Bytes())
	if len(responses) != 1 {
		t.Fatalf("expected one response, got %d: %s", len(responses), out.String())
	}
	if responses[0]["id"].(string) != "call-1" {
		t.Fatalf("expected call id to be echoed, got %#v", responses[0])
	}
	result := responses[0]["result"].(map[string]any)
	content := result["content"].([]any)
	first := content[0].(map[string]any)
	if first["type"] != "text" {
		t.Fatalf("expected text content, got %#v", first)
	}
	text := first["text"].(string)
	if !strings.Contains(text, `"subject": "Plaintext message"`) {
		t.Fatalf("expected parse output in MCP response, got %s", text)
	}
}

func TestMCPServeCallsTriageMessageTool(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":"call-triage","method":"tools/call","params":{"name":"mailcli_triage_message","arguments":{"file":"../testdata/emails/mime_attachment.eml"}}}` + "\n"

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetIn(strings.NewReader(input))
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"mcp", "serve"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected MCP triage call to succeed: %v\n%s", err, out.String())
	}

	responses := decodeJSONLines(t, out.Bytes())
	result := responses[0]["result"].(map[string]any)
	content := result["content"].([]any)
	text := content[0].(map[string]any)["text"].(string)
	for _, want := range []string{`"source": "deterministic"`, `"attachment_count": 1`} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected triage evidence to contain %q, got %s", want, text)
		}
	}
}

func decodeJSONLines(t *testing.T, raw []byte) []map[string]any {
	t.Helper()

	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	responses := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var response map[string]any
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			t.Fatalf("expected JSON response line: %v\n%s", err, line)
		}
		responses = append(responses, response)
	}
	return responses
}
