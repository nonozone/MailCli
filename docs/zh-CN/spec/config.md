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

## 创建配置

可以直接用 Go binary 提供的 `config init` 创建 starter config：

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
  --smtp-port 587 \
  --smtp-password-env MAILCLI_SMTP_PASSWORD
```

`config init` 会把秘密字段写成 `${MAILCLI_IMAP_PASSWORD}`、`${MAILCLI_SMTP_PASSWORD}` 这样的环境变量引用。它不会要求输入、写入或打印原始密码。

如果配置文件已经存在，该命令默认拒绝覆盖；需要显式传入 `--force`。由该命令创建的配置文件权限为 `0600`。

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

## 配置诊断

Agent 和 setup 脚本可以在不连接 IMAP / SMTP 的前提下做本地静态诊断：

```bash
mailcli config doctor --config ~/.config/mailcli/config.yaml
```

该命令输出 JSON。顶层 `status` 为 `ok`、`warning` 或 `error`，并包含每个账户的检查结果、账户能力，以及当存在警告或错误时的扁平 `problems` 列表。

秘密字段会同时看 raw config 和 resolved config。如果配置里存在 `password: ${MAILCLI_IMAP_PASSWORD}`，但对应环境变量没有设置，`config doctor` 会报告 `imap_password_env_unset`，而不是打印或保存秘密值。

缩略输出形状示例：

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

`config doctor` 不会输出配置中的 `password` 或 `smtp_password` 值。需要真正联网检查连接时，使用 `mailcli config test`。

## 能力发现

Agent 可以通过下面的命令读取当前账户的机器可用能力：

```bash
mailcli config capabilities --config ~/.config/mailcli/config.yaml --account work
```

该命令只读取本地配置和内置 driver 的已知能力，不会连接 IMAP / SMTP，也不会输出 `password` 或 `smtp_password`。

输出示例：

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

用途：

- onboarding 时判断账户配置是否完整
- 让 Agent 在调用 `send`、`watch`、`delete` 等命令前先判断能力
- 为后续 prepare / confirm / operation log 工作流提供账户能力边界

## v0.1 RC 明确不做的事

- 暂不内建 OAuth 流程
- 暂不集成 keyring
- 暂不集成 secret manager
