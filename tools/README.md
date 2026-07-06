# MailCli — AI Skill & Tool Integration Guide

MailCli exposes all its capabilities as structured JSON tools that any LLM with
function-calling / tool-use support can invoke via subprocess.

---

## Directory layout

```
tools/
  openai.json        OpenAI function-calling schema (tools[] array)
  anthropic.json     Anthropic tool-use schema (tools[] with input_schema)
  README.md          This file
```

## Codex / Claude Code via MCP

MailCLI also ships a local stdio MCP server:

```bash
mailcli mcp serve
```

The one-command installer can detect local agent CLIs and register that MCP
server:

```bash
curl -fsSL https://raw.githubusercontent.com/nonozone/MailCli/main/install.sh | sh -s -- --auto-configure
```

You can inspect or run the agent setup manually:

```bash
mailcli agent doctor
mailcli agent configure --agent codex
mailcli agent configure --agent claude
```

Default MCP tools are read/setup only:

- `mailcli_parse`
- `mailcli_list`
- `mailcli_get`
- `mailcli_sync`
- `mailcli_search`
- `mailcli_threads`
- `mailcli_thread`
- `mailcli_config_doctor`
- `mailcli_config_capabilities`

Mutating commands such as send, reply, delete, move, and mark are intentionally
not exposed by the default MCP server.

---

## Quick start: on-demand tool use from Go

Load a schema JSON file, pass it to your model runtime, then map tool calls back
to `mailcli` subprocess commands.

```go
package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

func runMailCLI(args ...string) (string, error) {
	cmd := exec.Command("mailcli", args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func main() {
	cmdArgs := toolToCommand("mailcli_search", map[string]any{
		"query": "invoice",
		"limit": 10,
	})
	out, err := runMailCLI(cmdArgs...)
	if err != nil {
		panic(err)
	}
	fmt.Print(out)
}
```

---

## Tool → CLI flag mapping

| Tool parameter  | mailcli flag        |
|----------------|---------------------|
| `account`      | `--account <value>` |
| `mailbox`      | `--mailbox <value>` |
| `limit`        | `--limit <value>`   |
| `since`        | `--since <value>`   |
| `before`       | `--before <value>`  |
| `query`        | positional arg `[0]`|
| `full`         | `--full`            |
| `refresh`      | `--refresh`         |
| `unread`       | `--unread`          |
| `has_codes`    | `--has-codes`       |
| `format`       | `--format <value>`  |
| `dest_mailbox` | positional arg `[1]`|
| `draft`        | JSON string arg     |

### Example mapping function (Go)

```go
package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

func toolToCommand(name string, input map[string]any) []string {
	sub := strings.TrimPrefix(name, "mailcli_")
	cmd := []string{sub}

	query, _ := input["query"].(string)
	dest, _ := input["dest_mailbox"].(string)
	draft := input["draft"]
	full, _ := input["full"].(bool)
	refresh, _ := input["refresh"].(bool)
	unread, _ := input["unread"].(bool)
	hasCodes, _ := input["has_codes"].(bool)

	delete(input, "query")
	delete(input, "dest_mailbox")
	delete(input, "draft")
	delete(input, "full")
	delete(input, "refresh")
	delete(input, "unread")
	delete(input, "has_codes")

	if query != "" {
		cmd = append(cmd, query)
	}
	if draft != nil {
		raw, _ := json.Marshal(draft)
		cmd = append(cmd, string(raw))
	}
	if dest != "" {
		cmd = append(cmd, dest)
	}

	for key, value := range input {
		flag := "--" + strings.ReplaceAll(key, "_", "-")
		cmd = append(cmd, flag, fmt.Sprint(value))
	}
	if full {
		cmd = append(cmd, "--full")
	}
	if refresh {
		cmd = append(cmd, "--refresh")
	}
	if unread {
		cmd = append(cmd, "--unread")
	}
	if hasCodes {
		cmd = append(cmd, "--has-codes")
	}
	return cmd
}
```

---

## Passive monitoring: `mailcli watch` pipeline

For continuous inbox monitoring with agent-side draft generation:

```bash
# Configure the sending identity used for reply dry-runs or sends
export MAILCLI_ACCOUNT=work
export MAILCLI_FROM_ADDRESS=support@example.com

# Human-in-the-loop (prints draft JSONs to stdout for review)
mailcli watch --account work \
  | go run ./examples/go/watch_reply_agent --draft-replies

# Fully automatic (use with caution!)
mailcli watch --account work \
  | MAILCLI_AUTO_SEND=1 go run ./examples/go/watch_reply_agent --draft-replies

# Watch multiple mailboxes
mailcli watch --account work --mailbox INBOX --mailbox "Customer Support" \
  | go run ./examples/go/watch_reply_agent --draft-replies
```

### Watch event schema (JSONL)

```jsonc
// Emitted once per mailbox on startup
{"event":"watching","account":"work","mailbox":"INBOX","ts":"2026-03-30T12:00:00Z"}

// Emitted when a new message arrives (full parsed content)
{"event":"new_message","account":"work","mailbox":"INBOX","id":"<abc@example.com>",
 "message":{...StandardMessage...},"ts":"2026-03-30T12:01:00Z"}

// Non-fatal errors (watch continues)
{"event":"error","account":"work","mailbox":"INBOX","error":"connection reset","ts":"..."}

// Optional keepalive (--heartbeat 5m)
{"event":"heartbeat","account":"work","mailbox":"INBOX","ts":"..."}
```

---

## Recommended workflow for an AI email agent

```
1. mailcli sync            ← populate index on startup
2. mailcli watch           ← start listening for new mail
3. On new_message event:
   a. Pass StandardMessage to LLM with system prompt
   b. LLM decides: reply / archive / flag / ignore
   c. If reply → mailcli reply '<draft-json>'
   d. If archive → mailcli move <id> Archive
   e. If flag → mailcli mark <id>  (mark read)
4. Periodically:
   a. mailcli search <topic>   ← on-demand context retrieval
   b. mailcli threads          ← summarise recent conversations
```

---

## StandardMessage schema reference

The `message` field in every `get`, `search`, `export`, and `new_message` event
follows this structure:

```jsonc
{
  "id": "<message-id@example.com>",
  "references": ["<parent@example.com>"],
  "meta": {
    "from": {"name": "Alice", "address": "alice@example.com"},
    "to":   [{"address": "me@example.com"}],
    "subject": "Invoice INV-2026-031",
    "date": "2026-03-30T08:00:00Z"
  },
  "content": {
    "format": "markdown",
    "body": "Your invoice is ready...",
    "snippet": "Your invoice is ready",
    "category": "transactional",
    "language": "en"
  },
  "actions": [
    {"type": "view", "label": "View Invoice", "url": "https://example.com/inv/031"}
  ],
  "codes": ["482991"],          // extracted verification / OTP codes
  "labels": ["invoice"],
  "attachments": [
    {"filename": "invoice.pdf", "content_type": "application/pdf", "size": 42000}
  ]
}
```
