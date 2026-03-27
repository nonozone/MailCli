[English](../../en/project/github-backlog.md) | 中文

# GitHub Backlog 草案

这个页面把 `v0.1.0-rc1` 之后的路线图，整理成可以直接复制到 GitHub 的 issue 草案。

适合在你准备创建 milestone、labels 和第一批 issues 时直接使用，避免每次重新组织措辞。

## 建议先做的基础设置

### Milestones

- `v0.1 hardening`
- `parser quality`
- `local memory`
- `contributor surface`
- `provider expansion`

### Labels

- `parser`
- `driver`
- `composer`
- `cmd`
- `schema`
- `docs`
- `examples`
- `governance`
- `good first issue`
- `rfc`

## 建议第一批先开的 Issue

优先创建这 8 个：

1. Align docs with actual RC capabilities
2. Add JSON contract snapshot tests for CLI commands
3. Strengthen HTML body extraction and noise filtering
4. Improve URL normalization for agent-facing actions
5. Surface local index and sync state more clearly
6. Enrich thread summaries for triage loops
7. Add a fake-driver test harness for extension contributors
8. Write a parser contributor guide

## Issue 草案

## Issue 1

Title: `Align docs with actual RC capabilities`

Type: Feature request

Milestone: `v0.1 hardening`

Labels: `docs`

Body:

```md
## Problem

README、release notes 和 specs 容易与当前 RC 的真实能力发生漂移。这会让贡献者和 agent 集成方难以判断哪些命令和 schema 边界已经可以被当作稳定接口来依赖。

## Proposed change

对齐当前 `v0.1.0-rc1` 的用户文档与真实实现边界。

## Area

- docs

## Contract impact

这个 issue 不应该修改机器接口契约。它应该明确说明哪些命令、字段和行为当前已经计划视为稳定，哪些区域仍属于 heuristic。

## Alternatives considered

- 保持文档大致一致，只依赖 release notes。
- 等后续功能再多一些时一起整理。

更窄也更合适的做法，是现在先把文档对齐，避免继续漂移。
```

## Issue 2

Title: `Add JSON contract snapshot tests for CLI commands`

Type: Feature request

Milestone: `v0.1 hardening`

Labels: `cmd`, `schema`

Body:

```md
## Problem

Agent 集成依赖稳定的 JSON 输出结构，但当前 CLI 还没有为主要机器接口命令建立明确的 snapshot / golden 覆盖。

## Proposed change

为 agent 工作流会依赖的核心 CLI 命令增加 JSON 输出 snapshot 或 golden tests。

## Area

- cmd
- schema

## Contract impact

这个 issue 的目标是加固现有契约，而不是扩大契约面。

建议优先覆盖：

- `mailcli parse`
- `mailcli get`
- `mailcli search`
- `mailcli threads`
- `mailcli thread`

同时明确文档中哪些字段计划在 `v0.1` 阶段保持稳定，哪些字段仍属于 heuristic。

## Alternatives considered

- 只依赖 package 级测试。
- 只为 `parse` 增加 snapshot，其他先延后。

这两种做法都无法有效避免 CLI 契约被不小心改坏。
```

## Issue 3

Title: `Strengthen HTML body extraction and noise filtering`

Type: Feature request

Milestone: `parser quality`

Labels: `parser`

Body:

```md
## Problem

一些邮件模板仍会把布局噪音、导航区块、页脚杂质或不理想的正文选择带进 `body_md`，影响 agent prompt 的可用性。

## Proposed change

在 Markdown 转换之前，提升 HTML 主体识别与清洗质量。

## Area

- parser

## Contract impact

这个 issue 应该提升输出质量，但不应该重新定义 `StandardMessage` 契约。

目标包括：

- 更准确地选择主消息正文
- 减少模板外壳对 `body_md` 的污染
- 在有价值时保留链接、标题、表格和关键图片

## Alternatives considered

- 保持现有 heuristic，只继续补 fixtures。
- 在共享 parser 中加入 provider 私有清洗规则。

前者力度不够，后者会破坏共享架构边界。
```

## Issue 4

Title: `Improve URL normalization for agent-facing actions`

Type: Feature request

Milestone: `parser quality`

Labels: `parser`

Body:

```md
## Problem

面向 agent 的 action URL 里仍可能带有较重的追踪或跳转包装，这会增加理解成本，也会降低 prompt 的清晰度。

## Proposed change

对提取出来的 action URL 清洗常见跳转模式，同时在需要时保留足够的原始数据用于调试。

## Area

- parser

## Contract impact

这个 issue 的重点是提升 action 质量，而不是引入不必要的契约变化。

建议优先覆盖：

- unsubscribe
- invoice / payment
- password reset
- attachment entry

## Alternatives considered

- 不处理 URL，交给外部工具自行清洗。
- 激进重写 URL，但不保留原始值。

前者对 agent 工作流太弱，后者又可能丢失调试与取证上下文。
```

## Issue 5

Title: `Surface local index and sync state more clearly`

Type: Feature request

Milestone: `local memory`

Labels: `cmd`, `docs`

Body:

```md
## Problem

当前用户和贡献者对 `mailcli sync` 到底缓存了什么、跳过了什么、刷新了什么，缺少足够清晰的可见性。

## Proposed change

通过 CLI 输出和文档，更清晰地暴露 sync 与本地索引状态。

## Area

- cmd
- docs

## Contract impact

这个 issue 可能会影响 sync 相关命令的 CLI 输出结构，因此任何 JSON 变更都应该谨慎处理并明确文档化。

可考虑补充：

- fetched、skipped、refreshed、failed 的计数
- 对增量 sync 行为的更清晰解释
- 当前索引的 account / mailbox 可见性

## Alternatives considered

- 保持行为不变，只补文档。
- 增加 verbose 日志，但不做结构化输出。

只补文档不够，而非结构化日志对 agent 也不够友好。
```

## Issue 6

Title: `Enrich thread summaries for triage loops`

Type: Feature request

Milestone: `local memory`

Labels: `cmd`, `schema`

Body:

```md
## Problem

agent 在决定某个会话是否值得打开、回复或降优先级之前，仍需要读取过多完整 thread。

## Proposed change

在保持输出紧凑的前提下，为 thread 摘要增加少量更适合 triage 的信号。

## Area

- cmd
- schema

## Contract impact

这个 issue 可能会影响 `mailcli threads` 和 `mailcli search` 的输出结构。如果增加字段，应尽量保持 additive，并明确写入文档。

可考虑的紧凑字段：

- 更清晰的 latest-message 时间语义
- participant summary
- action counts
- code presence 或 code counts

## Alternatives considered

- 保持当前摘要最小化，几乎所有场景都要求继续 `mailcli thread`
- 一次性增加过多字段，让紧凑输出也变得 prompt-heavy

更合理的方向是增加少量、明确、面向 triage 的 additive 字段。
```

## Issue 7

Title: `Add a fake-driver test harness for extension contributors`

Type: Feature request

Milestone: `contributor surface`

Labels: `driver`, `docs`, `good first issue`

Body:

```md
## Problem

外部贡献者目前还没有一个清晰且可复用的方式，去验证新 driver 是否满足 MailCLI 预期行为。

## Proposed change

增加 fake-driver 或 conformance 风格的测试支架，让 driver 预期行为显式且可复用。

## Area

- driver
- docs

## Contract impact

这个 issue 不应该扩大 driver 契约，而应该让现有契约更容易理解和测试。

最低要求应覆盖：

- config validation failures
- list behavior
- raw fetch behavior
- send behavior，或明确不支持 send 的行为

## Alternatives considered

- 依赖贡献者手动复制现有测试。
- 只写文档说明，不提供复用型测试辅助。

这两种方式都会让贡献门槛继续偏高。
```

## Issue 8

Title: `Write a parser contributor guide`

Type: Feature request

Milestone: `contributor surface`

Labels: `docs`, `parser`, `good first issue`

Body:

```md
## Problem

Parser 是 MailCLI 最有价值的贡献区域之一，但当前安全修改 parser 的路径仍然太隐性。

## Proposed change

编写一份专门的 parser contributor guide，明确 parser 修改应如何组织和验证。

## Area

- docs
- parser

## Contract impact

这个 issue 不应该修改契约。它的目标是减少触碰 parser 时的贡献歧义。

建议覆盖：

- parser fixtures 的存放位置
- golden tests 的组织方式
- 哪些 heuristic 漂移是可接受的
- 哪类 provider 私有逻辑不应该进入共享 parser 代码

## Alternatives considered

- 把 parser 贡献说明继续分散在代码和 PR review 里
- 只增加零散的代码注释

单独的指南会给外部贡献者一个更清晰的入口。
```

## RFC 候选场景

出现以下情况时，应该走 RFC 模板，而不是普通 feature request：

- 修改 `StandardMessage`
- 修改 `DraftMessage`
- 修改 `ReplyDraft`
- 修改 `SendResult`
- 修改稳定的 CLI JSON 输出
- 修改 `Driver` 契约预期

参考：

- [RFC issue template](../../../.github/ISSUE_TEMPLATE/rfc_contract_change.md)
- [下一阶段开发路线](next-roadmap.md)
