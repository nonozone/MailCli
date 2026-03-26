[English](../../en/examples/agent-inbox-assistant.md) | 中文

# Agent Inbox Assistant

这个示例展示了一个基于 `mailcli` 的最小 agent 收件箱闭环。

它刻意保持简单：

- 不依赖 LLM SDK
- 不依赖 agent framework
- 所有邮件 I/O 都通过 `mailcli`

示例脚本：

- `examples/python/agent_inbox_assistant.py`

## 它演示了什么

1. 通过 `mailcli parse` 或 `mailcli get` 读取邮件
2. 基于结构化的 `StandardMessage` 做推理
3. 提取 `codes`、`actions` 等高价值结果
4. 可选地生成 `ReplyDraft`
5. 通过 `mailcli reply --dry-run` 编译回复

## 本地 `.eml` 示例

```bash
python3 examples/python/agent_inbox_assistant.py \
  --mailcli-bin ./mailcli \
  --email testdata/emails/verification.eml
```

## 回复 Dry-Run 示例

```bash
python3 examples/python/agent_inbox_assistant.py \
  --mailcli-bin ./mailcli \
  --email testdata/emails/plaintext.eml \
  --from-address support@nono.im \
  --reply-text "Thanks for your email."
```

这会在输出里增加：

- `reply.draft`
- `reply.mime`

## 已配置 inbox 示例

```bash
python3 examples/python/agent_inbox_assistant.py \
  --mailcli-bin ./mailcli \
  --config ~/.config/mailcli/config.yaml \
  --account work \
  --message-id "<message-id>"
```

## 说明

- 这个示例使用规则式分析，方便开发者直接看懂来回流程。
- 你可以把 `analyze_message` 函数替换成自己的 LLM 或 agent framework。
- `mailcli` 应继续作为解析、回复编译和协议访问的稳定边界。

## 外部 Provider 模式

你也可以把分析步骤委托给自己的脚本或 agent runtime：

```bash
python3 examples/python/agent_inbox_assistant.py \
  --mailcli-bin ./mailcli \
  --email testdata/emails/plaintext.eml \
  --from-address support@nono.im \
  --agent-provider external \
  --provider-command python3 \
  --provider-arg ./my_provider.py
```

外部 provider 会从 stdin 收到 JSON：

```json
{
  "source": {},
  "message": {},
  "wants_reply": false
}
```

它应该返回类似这样的 JSON：

```json
{
  "decision": "draft_reply",
  "summary": "Short analysis",
  "reply_text": "Thanks for your email."
}
```

契约参考：

- [Agent Provider 契约](../spec/agent-provider.md)
- 模板 provider：`examples/providers/template_external_provider.py`
