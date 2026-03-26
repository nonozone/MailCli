[English](../../en/spec/agent-provider.md) | 中文

# Agent Provider 契约

## 目的

Python inbox agent 示例支持可插拔的 external provider 模式。

这样仓库本身可以保持零 LLM SDK 依赖，同时又为 OpenAI、Claude、本地模型或自定义 agent runtime 提供稳定的接入点。

## 入口方式

示例调用：

```bash
python3 examples/python/agent_inbox_assistant.py \
  --mailcli-bin ./mailcli \
  --email testdata/emails/plaintext.eml \
  --from-address support@nono.im \
  --agent-provider external \
  --provider-command python3 \
  --provider-arg ./my_provider.py
```

外部 provider 会以子进程方式运行。

## 输入契约

provider 会从 stdin 收到一个 JSON 对象：

```json
{
  "source": {
    "mode": "email",
    "value": "testdata/emails/plaintext.eml"
  },
  "message": {
    "id": "<plain-123@example.com>",
    "meta": {},
    "content": {},
    "actions": [],
    "codes": []
  },
  "wants_reply": false
}
```

字段说明：

- `source`：消息来源
- `message`：解析后的 `StandardMessage`
- `wants_reply`：调用方是否显式请求回复正文

## 输出契约

provider 必须向 stdout 输出一个 JSON 对象。

最小有效输出：

```json
{
  "decision": "review",
  "summary": "Short analysis"
}
```

可生成回复的输出：

```json
{
  "decision": "draft_reply",
  "summary": "Short analysis",
  "reply_text": "Thanks for your email."
}
```

## 必填字段

- `decision`：非空字符串

它的值必须属于下列稳定 decision 规范中的集合：

- [Agent Decision 规范](agent-decisions.md)

## 可选字段

- `summary`：简短分析摘要
- `reply_text`：回复正文；如果存在，必须是字符串

## 运行时校验

当前示例会校验：

- provider 返回的是 JSON 对象
- `decision` 存在且是非空字符串
- `reply_text` 如果存在，必须是字符串

如果校验失败，示例会直接退出并报告契约错误，而不是产出部分结果。

## Decision 词汇表

示例现在会按照共享的稳定 decision 集合来校验：

- [Agent Decision 规范](agent-decisions.md)
