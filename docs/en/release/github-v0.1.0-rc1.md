# MailCLI v0.1.0-rc1

First release candidate for MailCLI as an open, AI-native email interface.

## Highlights

- Parse raw email into `StandardMessage` JSON and Markdown
- Read inboxes through the baseline IMAP driver with `list` and `get`
- Send new outbound drafts and replies through SMTP-backed IMAP-style accounts
- Support `reply_to_message_id` and `reply_to_id` reply flows
- Compile outbound MIME with `multipart/alternative` and attachment packaging
- Expose stable machine-facing contracts for agents:
  - `StandardMessage`
  - `DraftMessage`
  - `ReplyDraft`
  - `SendResult`
- Ship Python and shell examples, including an external provider contract and an optional OpenAI provider example

## Included In This RC

- `mailcli parse`
- `mailcli list`
- `mailcli get`
- `mailcli send`
- `mailcli reply`
- IMAP raw fetch by sequence number, UID, and `Message-ID`
- Stable outbound result codes such as `auth_failed`, `message_not_found`, and `transport_not_configured`
- Config secret injection via environment variables for password fields
- Driver extension guidance and contributor docs
- Expanded parser fixture corpus covering:
  - plaintext mail
  - promo/newsletter mail
  - bounce mail
  - verification mail
  - invoice/payment mail
  - security reset mail
  - attachment-entry mail

## Still Evolving

- HTML cleanup and URL normalization heuristics
- action extraction coverage
- verification-code extraction outside common layouts
- richer search and local indexing
- additional providers beyond the baseline IMAP/SMTP path

## Recommended Stable Boundary For Integrators

For this RC, the intended stable boundary is:

- `mailcli parse`
- `mailcli list`
- `mailcli get`
- `mailcli send`
- `mailcli reply`
- `StandardMessage`
- `DraftMessage`
- `ReplyDraft`
- `SendResult`

## Verification

- `go test ./...`
- `go build ./cmd/mailcli`
- `python3 -m py_compile examples/python/*.py examples/providers/*.py`

## Docs

- Release notes: `docs/en/release/v0.1-rc.md`
- Agent workflows: `docs/en/agent-workflows.md`
- Outbound spec: `docs/en/spec/outbound-message.md`
- Driver extension spec: `docs/en/spec/driver-extension.md`
