[English](../../en/release/announcement-kit.md) | 中文

# MailCLI Announcement Kit

这个页面汇总了一组可复用的对外发布素材，用来介绍 MailCLI。

适用场景：

- GitHub release 描述
- X / Twitter 帖子
- Telegram 或 Discord 公告
- Hacker News / Reddit / 论坛介绍
- 个人博客或 changelog 开头

## 定位

MailCLI 是一个面向 agent 的开源邮件接口。

它并不打算成为一个传统 mail client。

它要提供的是 AI 系统和邮件系统之间的稳定边界：

- 把邮件读取为结构化 JSON 与干净 Markdown
- 使用本地消息与 thread 上下文工作
- 产出 `DraftMessage` 与 `ReplyDraft`，而不是原始 MIME
- 把邮箱接入与传输细节隐藏在 driver 和 CLI 契约后面

## GitHub 简短描述

MailCLI 是一个 AI Native 的邮件接口，面向 agent。它把原始 MIME 转换成结构化上下文，支持本地 thread-aware 检索，并让 agent 通过稳定的机器接口生成回复和发信 draft。

## GitHub Release 文案草稿

标题：

`MailCLI v0.1.0-rc1`

正文：

```md
MailCLI 是一个面向 agent 的开源邮件接口。

它不会把原始 MIME、臃肿 HTML 和 provider 私有行为直接推给 prompt，而是为 AI 系统提供一个稳定的机器边界，用来读取、理解、回复和发送邮件。

当前 RC 的亮点：

- 将原始邮件解析为 `StandardMessage` JSON 和 Markdown
- 通过基础 IMAP 能力支持 `list` 与 `get`
- 将近期邮件同步到本地索引
- 在不重复远端抓取的前提下检索本地消息
- 查看本地 thread 摘要和完整本地 thread
- 支持带有 `thread_id`、labels、categories、actions、codes、participants 以及紧凑计数字段的 thread-aware 本地分拣
- 通过 `listed_count`、`fetched_count`、`indexed_count`、`skipped_count`、`refreshed_count` 暴露更清晰的 sync / index 状态
- 通过更强的 HTML 主体提取和追踪链接清洗提升 parser 输出质量
- 通过 `DraftMessage` 和 `ReplyDraft` 编译新邮件与回复
- 同时支持单封邮件和 thread 场景的 external provider 工作流
- 提供 Python、shell、template provider 和可选 OpenAI provider 示例

建议集成方视为稳定边界的部分：

- `mailcli parse`
- `mailcli list`
- `mailcli get`
- `mailcli sync`
- `mailcli search`
- `mailcli threads`
- `mailcli thread`
- `mailcli reply`
- `mailcli send`
- `StandardMessage`
- `DraftMessage`
- `ReplyDraft`
- `SendResult`

推荐文档：

- `README.md`
- `docs/zh-CN/agent-workflows.md`
- `docs/zh-CN/examples/README.md`
- `docs/zh-CN/spec/agent-provider.md`
```

## 短公告版本

MailCLI 现在已经可以作为一个面向 agent 的开源邮件接口来使用。

它能把原始邮件解析成结构化 JSON，支持本地 thread-aware 检索与分拣，并让 agent 通过稳定 CLI 契约生成回复草稿，而不是手写 MIME。

仓库：`github.com/nonozone/MailCli`

## X / Twitter 版本

MailCLI 是一个面向 agent 的开源邮件接口。

它把原始 MIME 转成结构化 JSON + Markdown，支持本地 thread-aware 检索，并让 agent 通过稳定 CLI 契约生成 reply/send draft。

它不是给人类看的 mail client，而是给 AI 系统用的邮件边界。

## Hacker News / 论坛介绍

我最近在做 MailCLI，一个面向 agent 的开源邮件接口。

目标不是再做一个终端邮件客户端，而是给 AI 系统一个稳定的方法去读取、理解、回复和发送邮件，避免把原始 MIME 和各家 provider 的私有细节直接塞进 prompt。

当前已经覆盖：

- 将原始邮件解析成结构化 JSON 和 Markdown
- 基础 IMAP 读路径
- SMTP 发信路径
- 本地 sync、search、thread 摘要和完整 thread 读取
- thread-aware 回复流程
- 面向模型分析的 external provider handoff

我最在意的是：让 agent 开发者把邮件看成一个接口，而不是一个 scraping 问题。

## 对外沟通时建议强调的点

- “面向 agent 的邮件接口” 是第一定位。
- “不是传统 mail client” 有助于减少错误预期。
- “稳定的机器接口” 是核心价值。
- “thread-aware 的本地工作流” 是当前仓库状态里很强的一个差异点。
- “sync 状态可见性和 thread triage 信号” 让本地 memory 闭环更容易被 agent 稳定消费。
- “external provider handoff” 让项目可以和不同模型栈搭配，而不把核心仓库依赖搞重。

## 推荐链接

- [README](../../../README.zh-CN.md)
- [Agent 协作流程](../agent-workflows.md)
- [Examples 索引](../examples/README.md)
- [Agent Provider 契约](../spec/agent-provider.md)
- [本地索引规范](../spec/local-index.md)
- [Parser 贡献指南](../contributing/parser.md)
