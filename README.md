English | [中文](README.zh-CN.md)

# MailCLI

**AI-Native Email Interface: turning messy MIME into clean, structured context for agents.**

MailCLI is an open-source command-line toolkit built for **AI agents**, **LLM workflows**, and **automation developers**.

Traditional mail tools are designed for humans reading inboxes. MailCLI is designed for systems that need to **read**, **understand**, **reply to**, and **send** email through stable machine-friendly contracts.

Instead of pushing raw MIME, bloated HTML, and provider-specific quirks into prompts, MailCLI turns email into structured JSON, clean Markdown, and composable workflows.

## Project Status

MailCLI is currently in **pre-v0.1 release candidate** stage.

Working today:

- parse local `.eml` input or stdin into `StandardMessage`
- list messages from configured IMAP accounts
- fetch and parse messages by sequence number, UID, or `Message-ID`
- sync recent messages into a local searchable index
- search a local message index without re-fetching remote mail
- compile outbound drafts and replies
- send through SMTP-backed IMAP-style accounts
- integrate with Python or shell agent workflows through stable JSON contracts

Stable enough to build against for `v0.1 RC`:

- `mailcli parse`
- `mailcli list`
- `mailcli get`
- `mailcli sync`
- `mailcli search`
- `mailcli send`
- `mailcli reply`
- `StandardMessage`
- `DraftMessage`
- `ReplyDraft`
- `SendResult`

Still evolving:

- HTML cleanup and URL normalization heuristics
- richer list/search semantics for inbox workflows
- richer outbound HTML rendering and attachment ergonomics
- broader provider coverage and extension guidance

Current built-in driver types:

- `imap` for real mailbox access
- `stub` for local development, tests, and driver-extension examples

Current parser fixture coverage now includes:

- plaintext mail
- newsletter/promo mail
- delivery failure mail
- verification mail
- invoice/payment mail
- security reset mail
- attachment-entry mail

Heuristic areas to treat as evolving:

- action extraction coverage and classification
- verification-code extraction beyond common OTP layouts
- HTML body selection and cleanup for unusual templates
- token estimates

## Vision

In the AI era, email should not be treated as a pile of HTML and transport headers.

It should behave like a structured API resource.

MailCLI exists to make email as easy for agents to consume and produce as a JSON document.

## Key Features

- **AI-first parser**
  Convert noisy raw email into normalized JSON and Markdown suitable for agent reasoning.
- **Protocol/content separation**
  Drivers handle transport, parsers handle content, composers handle outbound MIME.
- **Action extraction**
  Extract unsubscribe links, security entry points, verification codes, invoice/payment entry points, attachment entry points, bounce/error context, and thread-related metadata.
  Verification-code extraction is conservative but now handles common multilingual and next-line layouts, and can expose `expires_in_seconds` when the mail states a relative expiry.
- **Developer-friendly CLI**
  Support `json`, `yaml`, and `table` output formats, stdin pipelines, and scriptable commands.
- **Bidirectional workflow**
  Read mail with `list/get/parse`, then produce `DraftMessage` and `ReplyDraft` flows for outbound delivery.
- **Provider-agnostic architecture**
  Designed to support IMAP, SMTP, APIs, and future ecosystem integrations without redefining the core model.

## Why MailCLI Exists

Raw email is a poor interface for agents:

- MIME trees are noisy
- HTML templates are token-expensive
- reply threading is easy to break
- provider APIs differ too much

MailCLI solves that by providing a stable boundary:

- inbound email becomes `StandardMessage`
- machine-usable artifacts such as actions, codes, and bounce context are extracted
- outbound intent becomes `DraftMessage` or `ReplyDraft`
- transport stays behind drivers

## Current Capabilities

### Read path

- `mailcli parse --format json|yaml|table <file|->`
- `mailcli list --config ~/.config/mailcli/config.yaml [--account <name>] [--mailbox <name>] [--limit <n>] [--format json|table]`
- `mailcli get --config ~/.config/mailcli/config.yaml [--account <name>] <id>`
- `mailcli sync --config ~/.config/mailcli/config.yaml [--account <name>] [--mailbox <name>] [--limit <n>] [--index <path>]`
- `mailcli search [--index <path>] [--limit <n>] <query>`

### Write path

- `mailcli send --dry-run <draft.json>`
- `mailcli send --config ~/.config/mailcli/config.yaml <draft.json>`
- `mailcli reply --dry-run <reply.json>`
- `mailcli reply --config ~/.config/mailcli/config.yaml <reply.json>`

### Reply support

- `reply_to_message_id` is supported
- `reply_to_id` is supported
- when `reply_to_id` is used, MailCLI can fetch the original message and derive:
  - `In-Reply-To`
  - `References`
  - default reply subject

## Architecture

MailCLI follows a layered architecture so contributors can work on clear boundaries:

1. **Driver Layer**
   Fetch raw messages and send raw bytes.
2. **Parser Engine**
   Decode MIME, normalize charsets, clean HTML, convert to Markdown, and extract actions.
3. **Composer**
   Compile `DraftMessage` and `ReplyDraft` into standards-compliant outbound MIME.
4. **CLI Core**
   Handle account selection, command routing, output formatting, and workflow orchestration.

Core rule:

**protocol belongs to drivers, content belongs to parsers, composition belongs to composers, orchestration belongs to the CLI core**

## Agent Collaboration Model

MailCLI is not just a parser. It is the bridge between agents and email systems.

### Read loop

```text
Agent -> mailcli list/get/parse -> Driver -> Raw Email -> Parser -> StandardMessage -> Agent
```

### Local retrieval loop

```text
Agent -> mailcli sync -> Local Index -> mailcli search -> Indexed Message Context -> Agent
```

### Reply loop

```text
Agent -> ReplyDraft -> mailcli reply -> Composer -> Raw MIME -> Driver -> Provider
```

### New outbound message loop

```text
Agent -> DraftMessage -> mailcli send -> Composer -> Raw MIME -> Driver -> Provider
```

Detailed workflow docs:

- [Agent Workflows](docs/en/agent-workflows.md)
- [Outbound Message Spec](docs/en/spec/outbound-message.md)
- [Local Index Spec](docs/en/spec/local-index.md)

## Build From Source

```bash
go build -o mailcli ./cmd/mailcli
./mailcli --help
```

## Minimal Config Example

```yaml
current_account: work
accounts:
  - name: work
    driver: imap
    host: imap.example.com
    port: 993
    username: you@example.com
    password: ${MAILCLI_IMAP_PASSWORD}
    tls: true
    mailbox: INBOX
    smtp_host: smtp.example.com
    smtp_port: 587
    smtp_username: you@example.com
    smtp_password: ${MAILCLI_SMTP_PASSWORD}
```

### Development Config Example

Use the built-in `stub` driver when you want to validate agent flows, CLI output, or parser integration without connecting a real mailbox:

```yaml
current_account: demo
accounts:
  - name: demo
    driver: stub
    mailbox: INBOX
```

Secret fields currently support environment-variable expansion:

- `password`
- `smtp_password`

Recommended usage:

- use app passwords or provider-issued tokens
- inject them through environment variables
- avoid committing real mailbox secrets into config files

## Quick Start

### Parse a local email

```bash
cat test.eml | mailcli parse --format json -
```

### List messages from a configured account

```bash
mailcli list --config ~/.config/mailcli/config.yaml --format table
```

### Fetch and parse a message by id

```bash
mailcli get --config ~/.config/mailcli/config.yaml "<message-id>"
```

### Sync recent messages into the local index

```bash
mailcli sync --config ~/.config/mailcli/config.yaml --limit 10
```

### Search the local index

```bash
mailcli search invoice
```

### Dry-run an outbound draft

```bash
mailcli send --dry-run draft.json
```

### Dry-run a reply

```bash
mailcli reply --dry-run reply.json
```

### Run the agent example

```bash
python3 examples/python/agent_inbox_assistant.py \
  --mailcli-bin ./mailcli \
  --email testdata/emails/verification.eml
```

## Roadmap

### Phase 1: The Brain

- [x] Define `StandardMessage`, `DraftMessage`, `ReplyDraft`, and `SendResult`
- [x] Build parser-first MVP with representative fixtures and golden tests
- [x] Support `parse`, `list`, `get`, `send`, and `reply` command skeletons
- [x] Add local MIME composer for outbound drafts and replies
- [ ] Improve HTML noise filtering with stronger body extraction and URL cleaning
- [x] Expand fixture corpus for newsletters, transactional mail, alerts, and edge cases

### Phase 2: The Hands

- [x] Add config-backed account selection
- [x] Add baseline IMAP read path
- [x] Add SMTP-backed send path for IMAP-style accounts
- [x] Implement IMAP `FetchRaw` for sequence numbers, UIDs, and `Message-ID`
- [x] Add a local file-backed indexing/search baseline for agent retrieval loops
- [x] Map operational send failures into stable result codes

### Phase 3: The Memory

- [ ] Add local indexing and mailbox cache
- [ ] Add search and thread navigation
- [ ] Add structured metadata for larger local agent workflows

### Phase 4: The Ecosystem

- [ ] Add more providers beyond IMAP/SMTP
- [x] Document ecosystem integrations and driver extension patterns
- [ ] Stabilize RFC-driven extension points for contributors

## Examples

- Python parse example: `examples/python/parse_email.py`
- Python reply dry-run example: `examples/python/reply_dry_run.py`
- Python inbox agent example: `examples/python/agent_inbox_assistant.py`
  Supports a built-in rule provider or an external command provider.
- External provider template: `examples/providers/template_external_provider.py`
- Optional OpenAI provider example: `examples/providers/openai_external_provider.py`
- Shell parse example: `examples/shell/parse_email.sh`
- Shell reply dry-run example: `examples/shell/reply_dry_run.sh`

## Contributing

MailCLI is still early, but the direction is intentional.

We want contributors in these areas:

1. **Parser quality**
   Better MIME handling, HTML cleanup, charset handling, and Markdown fidelity.
2. **Semantic contracts**
   Better shared specs for agent-facing email workflows.
3. **Drivers**
   More providers, safer transport behavior, and better compatibility layers.
4. **Agent tooling**
   Better examples, prompt patterns, and workflow integrations.

Major changes should be discussed first.

Start here:

- [Contribution Guide](CONTRIBUTING.md)

The project is community-open, but it is still directionally curated to stay focused on:

- AI-native workflows
- clean separation of concerns
- stable machine-facing contracts

## License

Apache-2.0

## Related

- [Agent Workflows](docs/en/agent-workflows.md)
- [Agent Decision Spec](docs/en/spec/agent-decisions.md)
- [Outbound Message Spec](docs/en/spec/outbound-message.md)
- [Agent Provider Contract](docs/en/spec/agent-provider.md)
- [Driver Extension Spec](docs/en/spec/driver-extension.md)
- [Adding a Driver](docs/en/contributing/drivers.md)
- [Config Spec](docs/en/spec/config.md)
- [v0.1 RC Release Notes](docs/en/release/v0.1-rc.md)
- [Agent Inbox Example](docs/en/examples/agent-inbox-assistant.md)
- [OpenAI External Provider Example](docs/en/examples/openai-external-provider.md)
- [Contribution Guide](CONTRIBUTING.md)
- [中文文档](README.zh-CN.md)
