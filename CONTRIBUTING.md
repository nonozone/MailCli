[中文](CONTRIBUTING.zh-CN.md) | English

# Contributing to MailCLI

MailCLI is an open-source, AI-native email interface project.

The goal is not to become a traditional terminal mail client. The goal is to build stable machine-facing contracts for email-driven agent workflows.

## Before You Start

- Read [README.md](README.md)
- Read [Agent Workflows](docs/en/agent-workflows.md)
- Read [Driver Extension Spec](docs/en/spec/driver-extension.md) if you want to add a provider
- Read the relevant spec before changing a public contract
- Open an issue first for major behavior, schema, or architecture changes

## What We Want Help With

- parser quality and fixture coverage
- provider and driver compatibility
- outbound MIME correctness
- agent workflow examples
- documentation and contributor experience

## Contribution Rules

- Keep the core provider-agnostic
- Do not add provider-specific business rules to the parser unless they are clearly generalizable
- Keep protocol logic in drivers, content logic in parsers, MIME generation in the composer, and orchestration in `cmd`
- Contract changes must update docs and tests together
- Prefer small pull requests with a clear scope

## Development Basics

```bash
go test ./...
go build ./cmd/mailcli
python3 -m py_compile examples/python/*.py examples/providers/*.py
```

## Pull Requests

Please include:

- what changed
- why it changed
- how it was tested
- docs updates when contracts or workflows changed

## Contract-Sensitive Changes

Open an issue before changing:

- `StandardMessage`
- `DraftMessage`
- `ReplyDraft`
- `SendResult`
- command output shapes
- driver extension boundaries

Driver contributors should also read:

- [Parser Contributor Guide](docs/en/contributing/parser.md) if you want to improve shared parsing behavior
- [Adding a Driver](docs/en/contributing/drivers.md)

## Language And Docs

- Main project docs are maintained in English and Chinese as separate files
- Prefer cross-links between languages instead of inline bilingual duplication
- If you update a user-facing doc, update both language versions unless the content is intentionally language-specific

## Review Standard

Changes are favored when they improve at least one of these without weakening the others:

- AI workflow usability
- separation of concerns
- machine-facing stability
- contributor clarity
