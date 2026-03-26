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

## Example

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

## Supported Fields

Per account:

- `name`
- `driver`
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

For `v0.1 RC`, built-in driver types are:

- `imap`
- `stub`

`imap` uses the mailbox and SMTP fields shown in the main example.

`stub` is a local deterministic development driver. It does not require host, port, username, or password.

Example:

```yaml
current_account: demo
accounts:
  - name: demo
    driver: stub
    mailbox: INBOX
```

## Secret Handling

Current secret expansion is intentionally narrow.

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

## Explicit Non-Goals For v0.1 RC

- no built-in OAuth flow
- no keyring integration yet
- no secret manager integration yet
