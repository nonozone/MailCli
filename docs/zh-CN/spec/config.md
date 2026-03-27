[English](../../en/spec/config.md) | 中文

# 配置规范

## 目的

MailCLI 使用本地 YAML 配置文件来完成账户选择和传输设置。

配置被刻意保持简单。它服务于本地 agent 工作流，而不是在核心项目里内建完整的 OAuth 或凭据管理系统。

## 默认路径

默认路径：

```text
~/.config/mailcli/config.yaml
```

## 示例

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

## 支持字段

每个账户当前支持：

- `name`
- `driver`
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

## 当前已知的 Driver 类型

在 `v0.1 RC` 阶段，当前内置的 driver 类型有：

- `imap`
- `dir`
- `stub`

`imap` 使用上面主示例中的邮箱和 SMTP 字段。

`dir` 是一个面向本地 `.eml` 目录的文件系统 driver。它要求：

- `path`

它支持相对路径或绝对路径。相对路径会相对于配置文件所在位置解析。它会递归列出 `.eml` 文件，返回相对文件 id，并支持通过相对文件 id 或 `Message-ID` 获取原始邮件。

它不提供出站发送能力。

`stub` 是一个本地、确定性的开发 driver，不要求 `host`、`port`、`username` 或 `password`。

示例：

```yaml
current_account: demo
accounts:
  - name: demo
    driver: stub
    mailbox: INBOX
```

本地目录示例：

```yaml
current_account: fixtures
accounts:
  - name: fixtures
    driver: dir
    path: ./testdata/emails
    mailbox: INBOX
```

## Secret 处理

当前环境变量展开能力刻意保持很窄。

支持展开环境变量的字段：

- `password`
- `smtp_password`

示例：

```yaml
password: ${MAILCLI_IMAP_PASSWORD}
smtp_password: ${MAILCLI_SMTP_PASSWORD}
```

非秘密字段不会被展开。

## 推荐实践

- 优先使用 app password 或 API token
- 通过环境变量注入秘密值
- 不要提交真实账户密码

## v0.1 RC 明确不做的事

- 暂不内建 OAuth 流程
- 暂不集成 keyring
- 暂不集成 secret manager
