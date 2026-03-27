[English](../../en/examples/agent-thread-assistant.md) | 中文

# Agent Thread Assistant

这个示例展示了一个更完整的、基于 `mailcli` 本地索引与线程导航的 agent 工作流。

它面向的是这类常见 agent 闭环：

1. 同步最近的 inbox 状态
2. 选择一个相关 thread
3. 读取本地完整 thread
4. 通过 `ReplyDraft` 草拟回复
5. 用 `mailcli reply --dry-run` 编译回复

示例脚本：

- `examples/python/agent_thread_assistant.py`

## 它演示了什么

1. 运行 `mailcli sync` 刷新本地索引
2. 使用 `mailcli threads` 对 thread 候选做排序
3. 使用 `mailcli thread` 读取选中的本地会话
4. 选择最新一封本地消息作为回复目标
5. 生成 `ReplyDraft`，而不是原始 MIME
6. 使用 `mailcli reply --dry-run` 编译草稿

## Inbox 驱动示例

```bash
python3 examples/python/agent_thread_assistant.py \
  --mailcli-bin ./mailcli \
  --config ~/.config/mailcli/config.yaml \
  --account work \
  --index /tmp/mailcli-index.json \
  --query invoice \
  --from-address support@nono.im \
  --reply-text "Thanks for your email."
```

## 复用已有本地索引示例

```bash
python3 examples/python/agent_thread_assistant.py \
  --mailcli-bin ./mailcli \
  --index /tmp/mailcli-index.json \
  --skip-sync \
  --thread-id "<root@example.com>" \
  --from-address support@nono.im \
  --reply-text "Thanks for your email."
```

## External Provider 模式

你也可以把 thread 分析步骤委托给自己的脚本或 agent runtime：

```bash
python3 examples/python/agent_thread_assistant.py \
  --mailcli-bin ./mailcli \
  --index /tmp/mailcli-index.json \
  --skip-sync \
  --thread-id "<root@example.com>" \
  --from-address support@nono.im \
  --agent-provider external \
  --provider-command python3 \
  --provider-arg ./my_provider.py
```

external provider 会收到一个 JSON payload，其中包含：

- `selection`
- `thread_summaries`
- `thread_messages`
- `latest_message`
- `wants_reply`

如果它返回：

```json
{
  "decision": "draft_reply",
  "summary": "Short analysis",
  "reply_text": "Thanks for your email."
}
```

这个示例就会自动继续编译 reply dry-run。

## Reply Draft 策略

当传入 `--config` 时，示例会优先使用最新本地消息的 id 作为 `reply_to_id`。这样 `mailcli reply` 可以重新抓取原始邮件并自动推导：

- `In-Reply-To`
- `References`
- 默认回复主题

当使用 `--skip-sync` 且没有 `--config` 时，示例会退回为纯本地 draft：

- `reply_to_message_id`
- 本地 `references`
- 本地 subject

这样即使开发者手上只有一个已有的本地索引快照，也能直接运行这个示例。

## 输出结构

脚本会输出一个 JSON 报告，其中包括：

- `sync`
- `selection`
- `thread_summaries`
- `thread_messages`
- `latest_message`
- `analysis`
- 可选的 `reply`

## 为什么需要这个示例

最小 inbox 示例更适合单封邮件分析。

这个 thread 示例则更接近真实 agent 场景：模型需要本地记忆、thread 级上下文，以及稳定的回复边界。

相关文档：

- [Examples 索引](README.md)
- [Agent 协作流程](../agent-workflows.md)
- [发送侧消息规范](../spec/outbound-message.md)
- [Agent Provider 契约](../spec/agent-provider.md)
- [Agent Inbox 示例](agent-inbox-assistant.md)
