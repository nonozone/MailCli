[English](../../en/spec/outbound-message.md) | 中文

# 发送侧消息规范

## 目的

发送链路不应该要求 AI agent 直接拼装原始 MIME。

更合理的方式是：agent 产出意图级草稿对象，由 `mailcli` 负责把这些草稿编译成符合标准的邮件，包括正确的头部、MIME 结构和 provider 传输行为。

## 核心对象

### `DraftMessage`

用于发送一封新的邮件。

```json
{
  "account": "work",
  "from": {
    "name": "Nono",
    "address": "support@nono.im"
  },
  "to": [
    { "address": "user@example.com" }
  ],
  "cc": [],
  "bcc": [],
  "subject": "Welcome",
  "body_md": "Hello, welcome to MailCLI.",
  "body_text": "Hello, welcome to MailCLI.",
  "headers": {
    "X-MailCLI-Agent": "assistant"
  },
  "attachments": []
}
```

### `ReplyDraft`

用于回复已有邮件。

```json
{
  "account": "work",
  "reply_to_message_id": "<orig-123@example.com>",
  "to": [
    { "address": "user@example.com" }
  ],
  "subject": "Re: Your question",
  "body_md": "Thanks for your email.\n\nHere is the answer...",
  "attachments": []
}
```

也可以通过内部 id 进行回复：

```json
{
  "account": "work",
  "reply_to_id": "imap:uid:12345",
  "body_md": "Thanks, got it."
}
```

当使用 `reply_to_id` 时，`mailcli` 可以通过配置好的 driver 抓取原邮件，并自动推导出：

- `reply_to_message_id`
- `references`
- 默认回复主题
- 当 `to` 省略时，默认回复收件人

### `SendResult`

用于返回发送或回复的执行结果。

```json
{
  "ok": true,
  "message_id": "<sent-456@example.com>",
  "thread_id": "<orig-123@example.com>",
  "provider": "imap-smtp",
  "account": "work"
}
```

失败示例：

```json
{
  "ok": false,
  "error": {
    "code": "auth_failed",
    "message": "SMTP authentication failed"
  },
  "provider": "imap-smtp",
  "account": "work"
}
```

当前稳定错误码：

- `auth_failed`
- `account_not_found`
- `account_not_selected`
- `message_not_found`
- `transport_not_configured`
- `invalid_draft`
- `transport_failed`

在内部实现上，若条件允许，应优先使用 typed operational errors，再映射成这组稳定的公开错误码。

## 编译器职责

`mailcli` 应负责把 `DraftMessage` 和 `ReplyDraft` 编译成真实邮件，并自动处理：

- `Message-ID`
- `Date`
- `From`
- `In-Reply-To`
- `References`
- `multipart/alternative`
- text / HTML 正文生成
- 字符集归一化
- `Content-Transfer-Encoding`
- 附件打包

## v0.1 RC 支持矩阵

当前已经具备的基础行为：

- 只有 `body_text`：输出单一 `text/plain` 邮件
- 有 `body_md`，并可选带 `body_text`：输出 `multipart/alternative`
- 有 `attachments`：输出 `multipart/mixed`，正文作为第一部分
- `reply_to_message_id` 和 `references`：写入回复头部
- 对非 dry-run 的发送命令，如果配置里有 `smtp_username` 或 `username`，可用于补全缺失的 `from.address`
- 当 `reply_to_id` 存在且 `to` 省略时，可从原邮件推导默认回复收件人

当前限制：

- HTML 输出由一个刻意保持简单的 Markdown 渲染器生成
- 附件当前是基于本地文件路径，不包含 inline attachment 或远程拉取
- 这是一套适合 agent 和自动化的基础发送能力，不是完整的富邮件引擎

当前 Markdown 渲染基线已经能保留：

- 标题
- Markdown 链接，会转换成可点击的 HTML anchor
- 无序列表，会输出为 `ul/li`
- 有序列表，会输出为 `ol/li`
- 引用块
- 简单 Markdown 表格
- 可读的纯文本链接回退，例如 `Label: https://...`

如果你想直接看固定的 JSON 与 MIME 对照样例，可以看：

- [Outbound Draft Patterns](../examples/outbound-draft-patterns.md)

## 可确认发送 Intent

对于 agent 自动化，全新出站邮件通常应该先准备，再发送：

```bash
mailcli send prepare --config ~/.config/mailcli/config.yaml draft.json
mailcli send confirm --config ~/.config/mailcli/config.yaml <intent-id>
mailcli operations list
mailcli operations show <operation-id|intent-id>
```

`send prepare` 会校验 draft，编译足够的 MIME 来生成稳定 `Message-ID`，写入发送 intent，并返回 `OperationIntentResult` JSON。它不会初始化 transport driver，也不会发信。

prepare 结果示例：

```json
{
  "status": "prepared",
  "intent_id": "intent_...",
  "operation": "send",
  "account": "work",
  "message_id": "<...>",
  "operations_path": "/home/user/.local/state/mailcli/operations.jsonl",
  "confirm_command": "mailcli send confirm --operations /home/user/.local/state/mailcli/operations.jsonl intent_...",
  "summary": {
    "subject": "Welcome",
    "to": [{"address": "user@example.com"}]
  }
}
```

`send confirm` 会重新加载已保存的 intent，并发送同一份 draft。成功时，`SendResult` 会包含 `intent_id` 和 `operation_id`。发生业务失败时，只要 intent 已经成功加载，MailCLI 仍会返回带 `error.code` 的结构化 `SendResult` JSON，并追加一条 failed operation log。confirm 失败返回也会包含 `intent_id` 和失败的 `operation_id`，agent 可以直接继续调用 `mailcli operations show` 追踪。
如果某个 intent 已经有成功的 `sent` operation 记录，后续再次 `send confirm` 会在调用 transport driver 前被拒绝，并返回 `error.code = "intent_already_sent"`。

operation log 是 JSONL，默认位置是：

```text
~/.local/state/mailcli/operations.jsonl
```

intent payload 会存放在 log 旁边：

```text
~/.local/state/mailcli/operations.jsonl.intents/
```

log 只保存摘要：主题、发件人、可见收件人、Bcc 数量、附件数量、状态、ID、时间和结构化错误。它不能保存完整 `body_text`、完整 `body_md`、raw MIME 或配置里的 secret。intent payload 文件为了确认时能发送同一个 draft，会保存完整 draft，并使用仅 owner 可读写的权限。

## 命令方向

推荐命令形态：

```bash
cat draft.json | mailcli send -
cat reply.json | mailcli reply -
mailcli send prepare draft.json
mailcli send confirm <intent-id>
```

这样接口保持语言无关，适合 agent、shell、Go、Node.js 和其他 runtime 调用。

## 当前状态

仓库中已经有 `pkg/composer`，可以在本地把 `DraftMessage` 和 `ReplyDraft` 编译成 MIME。

这个 composer 现在已经支持：

- 纯 `text/plain` 输出
- 当存在 Markdown 正文时输出 `multipart/alternative`
- 当存在附件时输出 `multipart/mixed`

`mailcli send` 已经接通，在有账户配置时可以把 MIME 交给 driver。

`mailcli send prepare` 和 `mailcli send confirm` 已经接通，作为新邮件发送的第一阶段 operation-intent 契约。

`mailcli reply` 也已经接通，在有账户配置时可以把 MIME 交给 driver。

对于非 dry-run 的发送命令，MailCLI 现在会在成功和业务失败时都返回 `SendResult` JSON，agent 可以直接根据 `error.code` 分支，而不必解析原始 stderr 文本。
