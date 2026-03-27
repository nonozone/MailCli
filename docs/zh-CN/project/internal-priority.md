[English](../../en/project/internal-priority.md) | 中文

# 内部主导开发顺序

这份文档定义的是：如果 MailCLI 在当前阶段仍然主要依赖维护者推进，那么最现实的下一步开发顺序应该是什么。

前提很简单：

- 欢迎社区参与
- 但社区参与目前仍然有限
- 核心产品方向仍然需要维护者自己扛住

所以目标不是把 issue 数量堆得很多。

目标是挑出最少、最关键的一组工作，把 MailCLI 从“一个不错的 RC 演示”推进到“agent 可以稳定依赖的接口”。

## 当前工作假设

接下来一个阶段，可以把 MailCLI 看成：

- 核心契约由维护者主导
- parser 质量由维护者主导
- 本地 memory 模型由维护者主导
- 文档、fixtures、examples、部分 contributor surface 由社区辅助

## 核心五项

这五个任务应该被视为接下来内部开发的主线顺序。

### 1. Add JSON contract snapshot tests for CLI commands

为什么排第一：

- 先把机器接口边界钉住，避免输出结构继续漂移
- 在输出 shape 被固定之后，parser 和 thread 改进会更安全
- 这能直接降低 agent 集成被意外改坏的风险

为什么应由维护者主导：

- 它实质上在定义 `v0.1.x` 和 `v0.2` 到底承诺什么
- 这件事非常接近 schema 和命令契约决策

## 2. Strengthen HTML body extraction and noise filtering

为什么排第二：

- parser 质量是目前最直接的产品价值来源
- 如果 `body_md` 质量不够，整个工作流的说服力都会下降
- 这也是 AI-native 邮件接口最强的差异化之一

为什么应由维护者主导：

- 它决定共享 parser 行为的质量基线
- 清洗策略一旦做错，会悄悄影响大量下游输出

## 3. Improve URL normalization for agent-facing actions

为什么排第三：

- action 质量直接决定 agent 能不能稳定使用提取出来的链接
- 更干净的 action URL 能减少 prompt 噪音，提升自动化质量
- 它和正文提取形成互补，会让 action extraction 更可信

为什么应由维护者主导：

- URL 清洗很容易做过头
- 错误的归一化可能破坏真实目标链接，或者丢掉有价值的上下文

## 4. Surface local index and sync state more clearly

为什么排第四：

- 本地 memory 是 MailCLI 在 agent 工作流里最实用的价值之一
- 当前 sync 已经可用，但状态可见性还不足以让人放心依赖
- 当 parser 质量提升后，本地检索的价值会进一步放大

为什么应由维护者主导：

- 这会影响本地 memory 模型如何被解释和消费
- 它也可能影响未来 CLI 输出和本地工作流语义

## 5. Enrich thread summaries for triage loops

为什么排第五：

- thread-aware 本地工作流是当前仓库状态里最强的亮点之一
- 增加紧凑的 triage 信号，可以减少不必要的完整 thread 加载
- 这会让 MailCLI 更像“agent memory interface”，而不是“parser + send 工具”

为什么应由维护者主导：

- 这件事非常接近长期 thread / memory 模型
- 字段一旦加出去，别人开始依赖后就很难再删

## 推荐执行顺序

1. 先用 snapshot tests 锁住 CLI JSON 形状。
2. 提升 `body_md` 质量。
3. 提升 action URL 质量。
4. 让本地 sync / index 状态更容易理解。
5. 让 thread 摘要更适合 triage。

## 可以往后放的事

这些都重要，但不应该抢走核心五项的主线位置：

- 文档对齐收口
- fake-driver contributor harness
- parser contributor guide
- 更多 provider 扩展
- 更广的社区流程建设

它们是围绕产品核心的支撑工作，不是通向更强 `v0.2` 的主路径。

## 适合社区并行辅助的工作

当维护者专注这五项时，社区仍然可以帮助推进：

- fixture 收集和脱敏
- examples
- 文档润色
- 小型 parser regression 报告
- contributor guides
- 不改变契约的测试补充

## 一条简单判断规则

如果一个任务会改变：

- 对外 JSON 形状
- parser 质量基线
- thread 摘要语义
- 本地 memory 语义

那它就应该优先由维护者主导。

如果一个任务主要提升：

- 文档
- examples
- fixtures
- 贡献者 onboarding

那它更适合更早开放给外部参与。

## 相关文档

- [下一阶段开发路线](next-roadmap.md)
- [GitHub Backlog 草案](github-backlog.md)
