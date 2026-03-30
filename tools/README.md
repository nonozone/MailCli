# MailCli — AI Skill & Tool Integration Guide

MailCli exposes all its capabilities as structured JSON tools that any LLM with
function-calling / tool-use support can invoke via subprocess.

---

## Directory layout

```
tools/
  openai.json        OpenAI function-calling schema (tools[] array)
  anthropic.json     Anthropic tool-use schema (tools[] with input_schema)
  agent_example.py   Minimal Python watch → LLM → reply pipeline
  README.md          This file
```

---

## Quick start: on-demand tool use

### Python (Anthropic)

```python
import json, subprocess, anthropic

def run_mailcli(*args) -> str:
    r = subprocess.run(["mailcli", *args], capture_output=True, text=True)
    return r.stdout

with open("tools/anthropic.json") as f:
    tools = json.load(f)["tools"]

client = anthropic.Anthropic()

# Pass tools to the model
response = client.messages.create(
    model="claude-3-5-sonnet-20241022",
    max_tokens=1024,
    tools=tools,
    messages=[{"role": "user", "content": "Search my inbox for invoices from last week"}],
)

# Execute tool calls
for block in response.content:
    if block.type == "tool_use":
        name = block.name          # e.g. "mailcli_search"
        inp  = block.input         # dict of parameters

        # Map tool names to mailcli subcommands + flags
        cmd = tool_to_cmd(name, inp)
        result = run_mailcli(*cmd)
        print(result)
```

### Python (OpenAI)

```python
import json, subprocess
from openai import OpenAI

with open("tools/openai.json") as f:
    tools = json.load(f)

client = OpenAI()
response = client.chat.completions.create(
    model="gpt-4o",
    tools=tools,
    messages=[{"role": "user", "content": "Search my inbox for invoices from last week"}],
)

for choice in response.choices:
    for call in (choice.message.tool_calls or []):
        args = json.loads(call.function.arguments)
        cmd  = tool_to_cmd(call.function.name, args)
        result = subprocess.run(["mailcli", *cmd], capture_output=True, text=True)
        print(result.stdout)
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

### Example mapping function (Python)

```python
import json

def tool_to_cmd(name: str, inp: dict) -> list[str]:
    sub = name.replace("mailcli_", "")  # mailcli_search → search
    cmd = [sub]

    draft = inp.pop("draft", None)
    query = inp.pop("query", None)
    dest  = inp.pop("dest_mailbox", None)
    full  = inp.pop("full", False)
    refresh = inp.pop("refresh", False)
    unread  = inp.pop("unread", False)
    has_codes = inp.pop("has_codes", False)

    if query:
        cmd.append(query)
    if draft:
        cmd.append(json.dumps(draft))
    if dest:
        cmd.append(dest)

    for k, v in inp.items():
        flag = "--" + k.replace("_", "-")
        cmd += [flag, str(v)]

    if full:      cmd.append("--full")
    if refresh:   cmd.append("--refresh")
    if unread:    cmd.append("--unread")
    if has_codes: cmd.append("--has-codes")

    return cmd
```

---

## Passive monitoring: `mailcli watch` pipeline

For continuous inbox monitoring with automatic AI replies:

```bash
# Install deps
pip install anthropic

# Set API key
export ANTHROPIC_API_KEY=sk-ant-...
export MAILCLI_ACCOUNT=work

# Human-in-the-loop (prints draft JSONs to stdout for review)
mailcli watch --account work | python3 tools/agent_example.py

# Fully automatic (use with caution!)
MAILCLI_AUTO_SEND=1 mailcli watch --account work | python3 tools/agent_example.py

# Watch multiple mailboxes
mailcli watch --account work --mailbox INBOX --mailbox "Customer Support" \
  | python3 tools/agent_example.py
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
