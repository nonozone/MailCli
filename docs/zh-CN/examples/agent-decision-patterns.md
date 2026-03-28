[English](../../en/examples/agent-decision-patterns.md) | 中文

# Agent 决策样例

这个页面展示三种基于当前 MailCLI agent 契约的具体决策样例。

它面向的是这类开发者：

- 想知道 agent provider 实际会收到什么
- 想知道 provider 应该返回哪种稳定 decision
- 想知道到底是哪些消息字段驱动了这个 decision

## 当前稳定 decision 范围

当前稳定 decision 定义见：

- [Agent Decision 规范](../spec/agent-decisions.md)

这个页面聚焦三种高频场景：

1. 验证码邮件
2. 退信 / 投递失败邮件
3. 退订 / 订阅邮件

## 已保存的样例资产

- [verification-input.json](../../../examples/artifacts/agent-decisions/verification-input.json)
- [verification-output.json](../../../examples/artifacts/agent-decisions/verification-output.json)
- [bounce-input.json](../../../examples/artifacts/agent-decisions/bounce-input.json)
- [bounce-output.json](../../../examples/artifacts/agent-decisions/bounce-output.json)
- [unsubscribe-input.json](../../../examples/artifacts/agent-decisions/unsubscribe-input.json)
- [unsubscribe-output.json](../../../examples/artifacts/agent-decisions/unsubscribe-output.json)

## 样例 1：验证码邮件

Decision：

```json
{
  "decision": "capture_code",
  "summary": "Verification email with 1 code(s). First code expires in 600 seconds."
}
```

原因：

- 解析结果里已经直接暴露了 `codes`
- 当前 agent 契约把验证码视为高优先级结果
- 这类邮件的价值在于提取 code，而不是立即生成回复

关键输入字段：

- `message.codes`
- 可选的 `expires_in_seconds`

## 样例 2：退信 / 投递失败

Decision：

```json
{
  "decision": "escalate_delivery_error",
  "summary": "535 Invalid login: 535 Authentication credentials invalid"
}
```

原因：

- 解析结果已经结构化暴露了 `error_context`
- 这类邮件是运行状态，不是普通 inbox 内容
- agent 更适合把它上抛给人工或上层自动化流程

关键输入字段：

- `message.error_context`
- `message.labels`
- `message.content.category`

## 样例 3：退订 / 订阅邮件

Decision：

```json
{
  "decision": "review",
  "summary": "Subscription email with 2 unsubscribe action(s)."
}
```

原因：

- 当前稳定 decision 集里还没有专门的 `unsubscribe`
- 但真正可执行的动作已经通过 `message.actions` 暴露出来了
- 用 `review` 可以保持契约足够小，同时让后续自动化自己检查退订 URL

关键输入字段：

- `message.actions`
- `message.meta.list_unsubscribe`

## 实践建议

- 当关键价值是 code 本身时，用 `capture_code`
- 当邮件在报告系统失败时，用 `escalate_delivery_error`
- 当下一步依赖下游策略判断时，即使 parser 已暴露动作，也继续使用 `review`

## 相关文档

- [Agent Provider 契约](../spec/agent-provider.md)
- [Agent Decision 规范](../spec/agent-decisions.md)
- [Examples 索引](README.md)
