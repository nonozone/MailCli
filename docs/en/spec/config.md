[中文](../../zh-CN/spec/config.md) | English

# Config Spec

## Purpose

MailCLI uses a local YAML config file for account selection and transport settings.

The config is intentionally simple. It is designed for local agent workflows, not for embedding full OAuth or credential management systems into the core project.

## Default Path

Default path:

```text
~/.config/mailcli/config.yaml
```

## Config Creation

Use `account add` for the human-friendly existing-mailbox setup path:

```bash
mailcli account add
```

For agent or script setup, pass the provider and secret reference explicitly:

```bash
mailcli account add \
  --provider gmail \
  --email you@gmail.com \
  --password-env MAILCLI_GMAIL_APP_PASSWORD \
  --format json
```

`account add` currently ships provider presets for:

- `gmail`
- `outlook`
- `microsoft365`
- `qq`
- `163`
- `generic-imap`

The command writes provider metadata and an `auth_method`, but it never asks for
or writes raw passwords, app passwords, authorization codes, or OAuth tokens.
It writes environment references into `password` and `smtp_password`.

Use `config init` to create a starter config from the Go binary:

```bash
mailcli config init \
  --config ~/.config/mailcli/config.yaml \
  --account work \
  --driver imap \
  --host imap.example.com \
  --port 993 \
  --username you@example.com \
  --password-env MAILCLI_IMAP_PASSWORD \
  --smtp-host smtp.example.com \
  --smtp-port 465 \
  --smtp-password-env MAILCLI_SMTP_PASSWORD
```

`config init` writes secret fields as environment references such as `${MAILCLI_IMAP_PASSWORD}` and `${MAILCLI_SMTP_PASSWORD}`. It does not ask for, write, or print raw password values.

The command refuses to overwrite an existing config unless `--force` is passed. Config files created by this command are written with `0600` permissions.

## Example

```yaml
current_account: work
accounts:
  - name: work
    provider: gmail
    driver: imap
    auth_method: app_password
    host: imap.example.com
    port: 993
    username: you@example.com
    password: ${MAILCLI_IMAP_PASSWORD}
    tls: true
    mailbox: INBOX
    smtp_host: smtp.example.com
    smtp_port: 465
    smtp_username: you@example.com
    smtp_password: ${MAILCLI_SMTP_PASSWORD}
```

## Supported Fields

Per account:

- `name`
- `provider`
- `driver`
- `auth_method`
- `path`
- `host`
- `port`
- `username`
- `password`
- `tls`
- `mailbox`
- `smtp_host`
- `smtp_port`
- `smtp_username`
- `smtp_password`
- `smtp_tls`

## Known Driver Types

Current built-in driver types are:

- `imap`
- `dir`
- `stub`

`imap` uses the mailbox and SMTP fields shown in the main example.

`dir` is a local filesystem driver for `.eml` directories. It requires:

- `path`

It accepts relative or absolute paths. Relative paths are resolved from the config file location. It lists `.eml` files recursively, returns relative file ids, and supports raw fetch by either relative file id or `Message-ID`.

It does not provide outbound transport.

`stub` is a local deterministic development driver. It does not require host, port, username, or password.

Example:

```yaml
current_account: demo
accounts:
  - name: demo
    driver: stub
    mailbox: INBOX
```

Local directory example:

```yaml
current_account: fixtures
accounts:
  - name: fixtures
    driver: dir
    path: ./testdata/emails
    mailbox: INBOX
```

## Secret Handling

Current secret expansion is intentionally narrow.

Non-secret metadata:

- `provider`
- `auth_method`

These fields describe how the account was configured. They do not contain
credentials.

Fields that expand environment variables:

- `password`
- `smtp_password`

Example:

```yaml
password: ${MAILCLI_IMAP_PASSWORD}
smtp_password: ${MAILCLI_SMTP_PASSWORD}
```

Non-secret fields are not expanded.

## Recommended Practice

- use app passwords or API tokens where supported
- inject secrets with environment variables
- do not commit real account secrets
- use `account add` for provider defaults before falling back to fully manual
  `config init`

## Provider Preset Boundary

Provider presets belong to onboarding and configuration generation. The IMAP
driver still consumes the same normalized account fields: host, port, username,
password reference, mailbox, and optional SMTP settings.

Current presets cover password-style IMAP and SMTP setup only. Gmail, QQ Mail,
and 163 Mail are expected to use app passwords or mailbox authorization codes.
Outlook and Microsoft 365 are read-first presets because many accounts require
OAuth or tenant-level IMAP/SMTP AUTH settings that MailCLI does not implement
yet.

`generic-imap` does not guess hostnames. Use:

```bash
mailcli account add \
  --provider generic-imap \
  --email you@example.com \
  --host imap.example.com \
  --smtp-host smtp.example.com \
  --smtp-port 465 \
  --password-env MAILCLI_GENERIC_PASSWORD
```

## Config Diagnostics

Agents and setup scripts can run local diagnostics without connecting to IMAP or SMTP:

```bash
mailcli config doctor --config ~/.config/mailcli/config.yaml
```

The command returns JSON with a top-level `status` of `ok`, `warning`, or `error`, per-account checks, account capabilities, and a flattened `problems` list when warnings or errors are present.

Secret checks use both raw and resolved config. If `password: ${MAILCLI_IMAP_PASSWORD}` is present but the environment variable is not set, `config doctor` reports `imap_password_env_unset` instead of printing or storing the secret value.

Abbreviated output shape:

```json
{
  "config_path": "/Users/you/.config/mailcli/config.yaml",
  "status": "warning",
  "accounts": [
    {
      "name": "work",
      "driver": "imap",
      "status": "warning",
      "capabilities": {
        "account": "work",
        "driver": "imap",
        "mailbox": "INBOX"
      },
      "checks": [
        {
          "status": "warning",
          "code": "smtp_port_missing",
          "message": "SMTP port must be greater than 0",
          "field": "smtp_port"
        }
      ]
    }
  ],
  "problems": [
    {
      "status": "warning",
      "code": "smtp_port_missing",
      "message": "SMTP port must be greater than 0",
      "field": "smtp_port"
    }
  ]
}
```

`config doctor` never prints configured `password` or `smtp_password` values. Use `mailcli config test` when you need a live connection check.

## Capability Discovery

Agents can inspect machine-readable capabilities for the selected account:

```bash
mailcli config capabilities --config ~/.config/mailcli/config.yaml --account work
```

This command reads only local config and known built-in driver behavior. It does not connect to IMAP / SMTP, and it does not print `password` or `smtp_password`.

Example output:

```json
{
  "account": "work",
  "driver": "imap",
  "mailbox": "INBOX",
  "capabilities": {
    "list": true,
    "fetch_raw": true,
    "search": true,
    "threads": true,
    "watch": true,
    "send": true,
    "reply": true,
    "delete": true,
    "move": true,
    "mark_read": true,
    "local_index": true
  },
  "configuration": {
    "inbound_configured": true,
    "outbound_configured": true,
    "uses_local_storage": false
  }
}
```

Use this for:

- checking whether account setup is complete during onboarding
- letting agents inspect support before calling `send`, `watch`, `delete`, or similar commands
- establishing the account boundary for later prepare / confirm / operation-log workflows

## Explicit Non-Goals For The Current Config Layer

- no built-in OAuth flow
- no keyring integration yet
- no secret manager integration yet
