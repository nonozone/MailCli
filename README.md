English | [中文](README.zh-CN.md)

# MailCLI

MailCLI is an independent, open-source toolkit that turns raw email into AI-consumable context without tying itself to any provider-specific API.

## Positioning

- AI agent developers and automation builders consume the same normalized JSON, Markdown, actions, and thread metadata.
- The parser-first approach keeps the project focused on data quality rather than reimplementing a user-facing client.
- The project remains provider agnostic so that ecosystem contributors can plug in IMAP, API, or OhRelay drivers later.

## Non-goals

- Not a traditional terminal mail client or UI replacement.
- Not a vehicle for embedding business integrations into the repository.
- Not aiming to support every provider-specific toggle in the core schema right now.
- Not a general-purpose sending/rules/GUI platform in the initial release.

## Parser-first MVP

- `mailcli parse <file>` and `cat <file> | mailcli parse -` both accept raw `.eml` input.
- Output includes normalized headers, Markdown content, actions (e.g., unsubscribe), thread context, and token estimates as defined in the schema.
- Golden tests based on representative Mercury, bounce, and plaintext samples validate parser behavior before publishing.
- The CLI stays minimal: parse command plus future hooks for output formatting and account context.

## Architecture

- `Cmd` layer manages Cobra commands, stdin/file handling, and formatter wiring.
- `Driver` interfaces provide transport (fetch/send) without touching content semantics.
- `Parser` handles MIME traversal, charset normalization, HTML cleanup, Markdown conversion, action extraction, and schema output.

## Links

- [中文文档](README.zh-CN.md)
