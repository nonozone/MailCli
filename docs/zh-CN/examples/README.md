[English](../../en/examples/README.md) | 中文

# Examples

这个页面是 MailCLI 可运行示例的统一入口。

当你想快速判断“我现在应该看哪个示例”时，从这里开始最合适。

维护者可以用下面这条命令校验仓库里提交的 local thread demo 产物是否仍然最新：

```bash
make demo-local-thread-check
```

## 从这里开始

- 如果你想走最快的零网络路径，直接使用内置 `dir` driver 的 `examples/config/fixtures-dir.yaml`。
- [Agent Inbox Assistant](agent-inbox-assistant.md)
  适合单封邮件解析、分类和 reply dry-run。
- [Agent Thread Assistant](agent-thread-assistant.md)
  适合本地索引、thread 选择、thread 上下文和回复工作流。
- [Local Thread Demo](local-thread-demo.md)
  适合直接理解本地 thread 链路里的 JSON 和 MIME 边界。
- [Agent 决策样例](agent-decision-patterns.md)
  适合查看 verification、bounce 和 unsubscribe 的决策样例。
- [Outbound Draft Patterns](outbound-draft-patterns.md)
  适合查看具体的 `ReplyDraft` 和 `DraftMessage` 交接模式。
- [OpenAI External Provider](openai-external-provider.md)
  适合把 OpenAI 驱动的分析器接到任一 agent 示例上。

## Python 脚本

### Agent 工作流

- `examples/python/agent_inbox_assistant.py`
  最小单封邮件 agent 边界。
- `examples/python/agent_thread_assistant.py`
  带 thread 感知的本地检索与回复边界。

### 原始工具

- `examples/python/parse_email.py`
  通过 `mailcli parse` 解析单个本地 `.eml` 文件。
- `examples/python/reply_dry_run.py`
  通过 `mailcli reply --dry-run` 编译单个 `ReplyDraft` JSON 文件。
- `examples/python/refresh_local_thread_demo.py`
  根据当前 fixture corpus 重新生成 local-thread-demo 的固定产物。

### Provider 适配器

- `examples/providers/template_external_provider.py`
  同时适用于 inbox 和 thread payload 的最小 external provider 模板。
- `examples/providers/openai_external_provider.py`
  使用 Responses API 的可选 OpenAI provider 示例。

## Shell 脚本

- `examples/shell/parse_email.sh`
- `examples/shell/reply_dry_run.sh`

## 建议阅读顺序

1. 如果你想先看最小 agent 闭环，从 [Agent Inbox Assistant](agent-inbox-assistant.md) 开始。
2. 如果你需要本地记忆和 thread 上下文，再看 [Agent Thread Assistant](agent-thread-assistant.md)。
3. 如果你需要稳定的 reply / 新邮件 JSON 边界，再看 [Outbound Draft Patterns](outbound-draft-patterns.md)。
4. 如果你想把规则分析替换成模型子进程，再接 [OpenAI External Provider](openai-external-provider.md)。

## 相关文档

- [Agent 协作流程](../agent-workflows.md)
- [Agent Provider 契约](../spec/agent-provider.md)
- [发送侧消息规范](../spec/outbound-message.md)
