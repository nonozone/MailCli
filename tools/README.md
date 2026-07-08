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

Standalone tool schemas still include write-capable tools for integrations that
explicitly opt in. Agent-initiated new outbound mail should use
`mailcli_send_prepare` followed by `mailcli_send_confirm`, not direct
`mailcli_send`.

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
	"strings"
)

func runMailCLI(stdin string, args ...string) (string, error) {
	cmd := exec.Command("mailcli", args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func main() {
	cmdArgs, stdin := toolToCommand("mailcli_search", map[string]any{
		"query": "invoice",
		"limit": 10,
	})
	out, err := runMailCLI(stdin, cmdArgs...)
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
| `config`       | `--config <value>`  |
| `account`      | `--account <value>` |
| `mailbox`      | `--mailbox <value>` |
| `limit`        | `--limit <value>`   |
| `since`        | `--since <value>`   |
| `before`       | `--before <value>`  |
| `operations`   | `--operations <value>` |
| `query`        | positional arg `[0]`|
| `id`           | positional arg `[0]`|
| `intent_id`    | positional arg `[0]`|
| `thread_id`    | positional arg `[0]`|
| `full`         | `--full`            |
| `refresh`      | `--refresh`         |
| `unread`       | `--unread`          |
| `has_codes`    | `--has-codes`       |
| `format`       | `--format <value>`  |
| `dest_mailbox` | positional arg `[1]`|
| `draft`        | JSON on stdin with positional `-` |

### Example mapping function (Go)

```go
package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

func toolToCommand(name string, input map[string]any) ([]string, string) {
	switch name {
	case "mailcli_search":
		return commandWithFlags([]string{"search"}, input, "query")
	case "mailcli_thread":
		return commandWithFlags([]string{"thread"}, input, "thread_id")
	case "mailcli_get", "mailcli_delete", "mailcli_mark":
		return commandWithFlags([]string{strings.TrimPrefix(name, "mailcli_")}, input, "id")
	case "mailcli_move":
		return commandWithFlags([]string{"move"}, input, "id", "dest_mailbox")
	case "mailcli_send", "mailcli_reply":
		return commandWithFlags([]string{strings.TrimPrefix(name, "mailcli_")}, input, "draft")
	case "mailcli_send_prepare":
		return commandWithFlags([]string{"send", "prepare"}, input, "draft")
	case "mailcli_send_confirm":
		return commandWithFlags([]string{"send", "confirm"}, input, "intent_id")
	case "mailcli_operations_list":
		return commandWithFlags([]string{"operations", "list"}, input)
	case "mailcli_operations_show":
		return commandWithFlags([]string{"operations", "show"}, input, "id")
	default:
		return commandWithFlags([]string{strings.TrimPrefix(name, "mailcli_")}, input)
	}
}

func commandWithFlags(cmd []string, input map[string]any, positionalKeys ...string) ([]string, string) {
	stdin := ""
	for _, key := range positionalKeys {
		value, ok := input[key]
		if !ok {
			continue
		}
		delete(input, key)
		if key == "draft" {
			raw, _ := json.Marshal(value)
			cmd = append(cmd, "-")
			stdin = string(raw)
			continue
		}
		if text := fmt.Sprint(value); text != "" {
			cmd = append(cmd, text)
		}
	}

	for _, key := range []string{"full", "refresh", "unread", "has_codes"} {
		enabled, _ := input[key].(bool)
		delete(input, key)
		if enabled {
			cmd = append(cmd, "--"+strings.ReplaceAll(key, "_", "-"))
		}
	}

	for key, value := range input {
		if value == nil {
			continue
		}
		flag := "--" + strings.ReplaceAll(key, "_", "-")
		cmd = append(cmd, flag, fmt.Sprint(value))
	}
	return cmd, stdin
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
   c. If reply → cat reply.json | mailcli reply --dry-run - first, then require local approval before direct send
   d. If new outbound mail → cat draft.json | mailcli send prepare -, inspect summary, then mailcli send confirm <intent-id>
   e. If archive / flag → require local policy approval before mailcli move or mailcli mark until mutation intents are implemented
4. Periodically:
   a. mailcli search <topic>   ← on-demand context retrieval
   b. mailcli threads          ← summarise recent conversations
   c. mailcli operations list   ← inspect prepared, sent, and failed operations
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
