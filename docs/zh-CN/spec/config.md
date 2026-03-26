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
