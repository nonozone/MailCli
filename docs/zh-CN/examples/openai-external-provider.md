[English](../../en/examples/openai-external-provider.md) | 中文

# OpenAI External Provider

这个示例演示如何把基于 OpenAI 的分析器接到 MailCLI 的 external provider 契约上。

它被刻意设计为可选示例：

- 主仓库不引入 OpenAI SDK 依赖
- provider 以独立子进程示例存在
- 契约仍然是 stdin / stdout JSON

示例文件：

- `examples/providers/openai_external_provider.py`

## 安装

```bash
pip install openai
```

## 环境变量

```bash
export OPENAI_API_KEY=...
export OPENAI_MODEL=gpt-5-mini
```

`OPENAI_MODEL` 是可选的，默认值为 `gpt-5-mini`。

## 配合 Agent 示例运行

```bash
python3 examples/python/agent_inbox_assistant.py \
  --mailcli-bin ./mailcli \
  --email testdata/emails/plaintext.eml \
  --from-address support@nono.im \
  --agent-provider external \
  --provider-command python3 \
  --provider-arg examples/providers/openai_external_provider.py
```

同一个 provider 也可以直接用于 thread 场景示例：

```bash
python3 examples/python/agent_thread_assistant.py \
  --mailcli-bin ./mailcli \
  --index /tmp/mailcli-index.json \
  --skip-sync \
  --thread-id "<root@example.com>" \
  --from-address support@nono.im \
  --agent-provider external \
  --provider-command python3 \
  --provider-arg examples/providers/openai_external_provider.py
```

## 它做了什么

1. 从 stdin 读取 MailCLI provider payload
2. 调用 OpenAI Responses API
3. 通过 JSON Schema 请求结构化输出
4. 在 stdout 返回经过校验的 provider 响应

## 输出契约

这个示例返回与共享 external provider 契约一致的 JSON：

```json
{
  "decision": "review",
  "summary": "Short analysis"
}
```

如果适合回复，也可以返回：

```json
{
  "decision": "draft_reply",
  "summary": "Short analysis",
  "reply_text": "Thanks for your email."
}
```

相关参考：

- [Examples 索引](README.md)
- [Agent Provider 契约](../spec/agent-provider.md)
- [Agent Decision 规范](../spec/agent-decisions.md)
- [Agent Inbox 示例](agent-inbox-assistant.md)
- [Agent Thread 示例](agent-thread-assistant.md)
