[English](../../en/spec/agent-decisions.md) | 中文

# Agent Decision 规范

## 目的

agent 示例使用一组很小的 decision 词汇表，让面向 MailCLI 的集成能够交换稳定、机器可读的意图，而不是随意的状态字符串。

## 稳定 decision 集合

当前稳定 decision 有：

- `review`
- `capture_code`
- `draft_reply`
- `escalate_delivery_error`

## 含义

### `review`

这封邮件需要进一步检查或分类，但暂时不需要立刻执行自动动作。

### `capture_code`

这封邮件包含验证码或类似的高价值 code，agent 应优先提取并上报。

### `draft_reply`

provider 希望示例工作流生成回复草稿，并继续进入 `mailcli reply --dry-run`。

### `escalate_delivery_error`

这封邮件代表投递失败或运行错误，应上报给人工或更高层的自动化流程。

## 为什么集合要保持很小

目标是稳定，而不是一开始就追求完整。

较小的词汇表更容易让：

- 示例脚本
- 外部 provider
- 测试
- 未来的 SDK 适配层

保持一致实现。

## 扩展方式

如果确实需要新增 decision：

1. 先更新这份 spec
2. 更新示例里的运行时校验
3. 增加或更新集成测试
4. 在示例文档中补充行为说明

不要在 provider 实现里直接发明新的 decision 字符串而不更新共享契约。

决策样例：

- [Agent 决策样例](../examples/agent-decision-patterns.md)
