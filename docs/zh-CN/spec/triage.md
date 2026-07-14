[English](../../en/spec/triage.md) | 中文

# Triage Evidence 与 Enrichment 规范

## 目的

MailCLI 把“可以确定性得到的事实”和“必须依赖 heuristic 或外部 AI provider
的判断”分开。

这样 `priority`、`needs_reply` 等字段不会伪装成 parser 事实。Go 核心不调用
LLM，也不自行生成这些判断。

## 命令

```bash
mailcli triage message message.eml
mailcli triage thread --index ~/.cache/mailcli/index.db "<thread-id>"
```

两个命令都会返回带版本号的 `TriageRecord`，其中 `evidence` 完全由确定性
逻辑产生。thread 命令会读取匹配线程的全部本地消息，不使用 `mailcli thread`
展示命令默认的 50 封限制。

每条记录还包含一个 SHA-256 `evidence_id`。外部 enrichment 必须原样回传该值，
这样线程新增消息后，基于旧上下文生成的判断不会被静默合并到新 evidence。

如果多个账户存在相同 thread id，应传入 `--account`。MailCLI 会拒绝把不同
账户的邮件静默合并成同一条 triage 记录。

## 确定性 Evidence

Evidence 包含：

- 消息数量、按时间排列的 message id、参与者、category 和 label
- 最新时间、最后一封消息 id 与最后发件人
- action、code、附件、auto-submitted 与 error 数量
- 每封消息对应一个紧凑 `messages` fact，包含发件人、收件人、主题、时间、
  snippet 和该消息的计数

保留逐封消息 fact 是有意设计。一个需要处理的请求可能出现在较早的邮件里，
后面又经过数次回复仍未完成。只看最后一封邮件，无法可靠判断 `needs_reply`。

Evidence 是紧凑上下文，不代替完整对话。外部 provider 判断 `needs_reply`、
deadline 或 todo 时，如果正文上下文会影响结论，还应读取完整的
`mailcli thread` 输出。

## 外部 Enrichment

外部 provider 可以返回 `TriageEnrichment` JSON：

```json
{
  "version": "v1",
  "scope": "thread",
  "subject_id": "<root@example.com>",
  "evidence_id": "sha256:...",
  "source": {
    "kind": "external",
    "provider": "example-provider",
    "model": "example-model"
  },
  "generated_at": "2026-07-14T09:00:00Z",
  "summary": "客户正在等待修改后的报价。",
  "priority": {
    "level": "high",
    "confidence": 0.9,
    "reasons": ["客户要求明天之前收到报价。"]
  },
  "needs_reply": {
    "value": true,
    "confidence": 0.85,
    "reasons": ["第一封邮件里的请求仍未完成。"]
  },
  "todos": [
    {
      "text": "发送修改后的报价",
      "source_message_id": "msg-root",
      "confidence": 0.9
    }
  ]
}
```

通过同一个命令合并并校验 enrichment：

```bash
mailcli triage thread \
  --index ~/.cache/mailcli/index.db \
  --enrichment enrichment.json \
  "<root@example.com>"
```

`--enrichment -` 从 stdin 读取 enrichment JSON。`triage message` 不能让邮件
正文与 enrichment 同时占用 stdin。

## 校验规则

以下情况会被拒绝：

- 契约版本、scope、subject id 或 evidence id 与当前 evidence 不匹配
- source kind 没有明确标注为 `external` 或 `heuristic`
- 缺少 provider 或 RFC3339 `generated_at`
- priority 不属于 `low`、`normal`、`high`、`urgent`
- confidence 不在 0 到 1 之间
- priority 或 needs-reply 没有给出 reasons
- todo 或 deadline 没有指向 evidence 中真实存在的 message id
- JSON 带有未知字段或多个顶层值

合并结果不会写回本地索引。调用方负责在自己的工作流状态中保存结果。

## 测试与评估

Snapshot test 保护确定性 evidence 的结构和 enrichment 校验契约，但不能证明
AI provider 的 triage 判断质量。

Provider 评估应使用单独的人工标注线程集，至少覆盖：

- 请求埋在前面数封邮件中的线程
- 最后一封由用户发送或由对方发送的不同情况
- 自动回执与 no-reply 发件人
- 看似结束后又重新打开的对话
- 引用内容里的旧请求，不应被当成新请求
- 明确和含糊的 deadline 表达

应分别评估 `priority`、`needs_reply`、deadline 和 todo。Parser fixtures 全绿，
不能作为 enrichment 判断质量良好的证据。

## 安全与隐私

- Triage evidence 不包含附件字节。
- 附件文件名始终是不可信 metadata。
- MailCLI 核心不会自行把数据发送给外部 provider。
- 调用方决定向外部进程提供紧凑 evidence、完整消息或两者。

安全 MCP surface 提供确定性的 `mailcli_triage_message` 与
`mailcli_triage_thread` 工具；enrichment 生成仍留在核心之外。
