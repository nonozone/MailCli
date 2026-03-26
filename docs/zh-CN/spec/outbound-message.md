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

## 命令方向

建议的未来命令形态：

```bash
cat draft.json | mailcli send -
cat reply.json | mailcli reply -
```

这样接口保持语言无关，适合 agent、shell、Python 和 Node.js 调用。

## 当前状态

仓库中已经有 `pkg/composer`，可以在本地把 `DraftMessage` 和 `ReplyDraft` 编译成 MIME。

`mailcli send` 已经接通，在有账户配置时可以把 MIME 交给 driver。

`mailcli reply` 也已经接通，在有账户配置时可以把 MIME 交给 driver。
