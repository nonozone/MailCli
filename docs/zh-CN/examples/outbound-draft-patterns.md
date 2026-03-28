[English](../../en/examples/outbound-draft-patterns.md) | 中文

# Outbound Draft Patterns

这个页面专门展示 agent 在“已经做完决策之后”，应该把什么样的最小出站对象交还给 MailCLI。

重点不是教 MIME。

重点是让开发者快速看懂三种最常见的出站边界：

1. 简短确认回复
2. 稍复杂的支持型回复
3. 全新发起的一封邮件

## 固定样例产物

- [ack-reply.draft.json](../../../examples/artifacts/outbound-patterns/ack-reply.draft.json)
- [ack-reply.mime.txt](../../../examples/artifacts/outbound-patterns/ack-reply.mime.txt)
- [support-followup.reply.json](../../../examples/artifacts/outbound-patterns/support-followup.reply.json)
- [support-followup.mime.txt](../../../examples/artifacts/outbound-patterns/support-followup.mime.txt)
- [release-update.draft.json](../../../examples/artifacts/outbound-patterns/release-update.draft.json)
- [release-update.mime.txt](../../../examples/artifacts/outbound-patterns/release-update.mime.txt)

这些固定 MIME 文件对运行时生成的 `Message-ID` 和 multipart boundary 做了归一化处理，方便阅读和版本管理。

## 模式 1：简短确认回复

当 agent 已经选中了本地消息或 thread，只需要回一条安全、简短的确认时，使用这个模式。

草稿边界：

```json
{
  "account": "fixtures",
  "from": { "address": "support@nono.im" },
  "to": [{ "address": "billing@example.com", "name": "Billing Team" }],
  "body_text": "Thanks, we received the invoice notification and queued it for processing.",
  "reply_to_id": "invoice.eml"
}
```

编译命令：

```bash
mailcli reply --config examples/config/fixtures-dir.yaml --account fixtures --dry-run examples/artifacts/outbound-patterns/ack-reply.draft.json
```

这个形态的价值：

- `reply_to_id` 允许 MailCLI 回抓原邮件并自动推导线程头
- agent 不需要自己拼 `In-Reply-To` 或 `References`
- 它非常适合零网络 demo 和本地检索闭环

## 模式 2：支持型跟进回复

当上游系统已经知道原始 `Message-ID`，而 agent 需要返回一条更完整的支持回复时，使用这个模式。

草稿边界：

```json
{
  "from": { "name": "Nono Support", "address": "support@nono.im" },
  "to": [{ "address": "sender@example.com", "name": "Example Sender" }],
  "subject": "Re: Plaintext message",
  "body_text": "Hi,\n\nWe checked your request and need one detail before we can proceed:\n\n- workspace name\n- expected delivery date\n\nYou can reply directly to this thread.\n\nOriginal request received and queued.",
  "body_md": "Hi,\n\nWe checked your request and need one detail before we can proceed:\n\n- workspace name\n- expected delivery date\n\nYou can reply directly to this thread.\n\n> Original request received and queued.",
  "reply_to_message_id": "<plain-123@example.com>",
  "references": ["<plain-123@example.com>"]
}
```

编译命令：

```bash
mailcli reply --dry-run examples/artifacts/outbound-patterns/support-followup.reply.json
```

这个形态的价值：

- `reply_to_message_id` 让草稿在无法重新抓取原邮件时仍然可用
- 显式 `references` 能跨 transport 保持 thread 连续性
- `body_md` 保留 richer 结构，而 `body_text` 提供稳妥的纯文本回退

## 模式 3：全新出站邮件

当 agent 不是在回复现有线程，而是在主动发起一封新邮件时，使用这个模式。

草稿边界：

```json
{
  "from": { "name": "MailCLI Bot", "address": "updates@mailcli.dev" },
  "to": [{ "address": "team@example.com", "name": "MailCLI Contributors" }],
  "subject": "Weekly parser status",
  "body_text": "Weekly parser status\n\nWe shipped three improvements:\n1. Better thread examples\n2. Safer zero-network onboarding\n3. Cleaner outbound MIME docs\n\nArea / Status\nParser fixtures / Stable\nThread examples / Expanded\nDriver docs / Ready for review\n\nExamples index: https://github.com/nonozone/MailCli/tree/main/docs/en/examples/README.md",
  "body_md": "## Weekly parser status\n\nWe shipped three improvements:\n\n1. Better thread examples\n2. Safer zero-network onboarding\n3. Cleaner outbound MIME docs\n\n| Area | Status |\n| --- | --- |\n| Parser fixtures | Stable |\n| Thread examples | Expanded |\n| Driver docs | Ready for review |\n\nRead the [examples index](https://github.com/nonozone/MailCli/tree/main/docs/en/examples/README.md) for runnable flows."
}
```

编译命令：

```bash
mailcli send --dry-run examples/artifacts/outbound-patterns/release-update.draft.json
```

这个形态的价值：

- agent 停留在意图层，不需要手写 MIME
- MailCLI 可以自动输出 `multipart/alternative`
- 这是状态通知、摘要汇总和自动化报告的基础模式

## 怎么选

- 当 MailCLI 已经能访问原邮件时，优先用 `reply_to_id`。
- 当 thread 元数据来自其他系统、需要保持可移植性时，用 `reply_to_message_id` 加 `references`。
- 当你要发起一个全新会话，不需要任何回复头时，用 `DraftMessage`。

## 相关文档

- [发送侧消息规范](../spec/outbound-message.md)
- [Agent 协作流程](../agent-workflows.md)
- [Local Thread Demo](local-thread-demo.md)
- [Examples 索引](README.md)
