[English](../../en/project/internal-priority.md) | 中文

# 内部主导开发顺序

这份文档定义的是：如果 MailCLI 在当前阶段仍然主要依赖维护者推进，那么最现实的下一步开发顺序应该是什么。

前提很简单：

- 欢迎社区参与
- 但社区参与目前仍然有限
- 核心产品方向仍然需要维护者自己扛住

所以目标不是把 issue 数量堆得很多。

目标是挑出最少、最关键的一组工作，把 MailCLI 从“一个不错的 RC 演示”推进到“agent 可以稳定依赖的接口”。

## 已完成的基线主线

最初那组由维护者主导的“核心五项”已经在 `main` 上完成：

1. CLI JSON 快照测试
2. 更强的 HTML 主体提取
3. 更干净的追踪 URL 归一化
4. 更清晰的本地 sync / index 状态
5. 更丰富的 thread triage 摘要信号

另外，local thread demo 固定产物的维护闭环也已经接入仓库级命令和 CI 校验。

这意味着下一阶段不应该继续停留在“边界是否成立”，而应该转向语料、贡献杠杆和共享质量基线。

## 当前工作假设

接下来一个阶段，可以把 MailCLI 看成：

- 核心契约由维护者主导
- parser 质量由维护者主导
- 本地 memory 模型由维护者主导
- 文档、fixtures、examples、部分 contributor surface 由社区辅助

## 下一阶段核心五项

这五个任务应该被视为接下来内部开发的主线顺序。

### 1. 扩大 parser 样本集，覆盖真实失败模式

为什么排第一：

- 当前 parser 质量更受语料覆盖约束，而不是缺少基础设施
- 更多 fixture 会让现有 heuristic 的演进更安全
- 这是在不重定义契约的前提下，最快提升 agent 可用性的方式

为什么应由维护者主导：

- fixture 选择本身就在定义 parser 的质量基线
- 第一批语料整理应体现维护者对产品方向的判断

## 2. 为社区 driver 建立 fake-driver / 合规测试支架

为什么排第二：

- driver 贡献门槛仍然偏高
- 共享验收标准能降低维护者 review 成本
- 这会为后续社区 provider 扩展打开更安全的路径

为什么应由维护者主导：

- 它在定义整个生态会继承的最小传输契约
- 支架如果太弱，后面 provider PR 会变得非常嘈杂

## 3. 收紧本地 search 语义与排序

为什么排第三：

- 本地 memory 已经是一个真实可用的工作流
- 更好的排序和过滤能减少不必要的完整消息加载
- 这可以提升 agent 价值，同时不引入新的传输表面积

为什么应由维护者主导：

- search 语义非常接近长期 memory 模型
- 排序策略一旦做差，会很快污染 prompt 和 examples

## 4. 在不扩张核心契约的前提下改善出站体验

为什么排第四：

- send / reply 已经能用，但贡献者和使用者体验还有提升空间
- 这能在更大 provider 工作延后时，继续增强读写闭环
- 也能让 agent 的收发链路更完整

为什么应由维护者主导：

- send / reply 体验很接近公开稳定契约
- 需要由维护者决定便利性增强和 schema 膨胀之间的边界

## 5. 在前四项收口后，再增加一个新的内置 provider

为什么排第五：

- 生态广度很重要，但前提是共享层已经更容易安全扩展
- 新增一个 provider 能加强架构故事，但前提是不能扰动核心
- 当 parser、本地 memory 和 driver 指引更稳后，这件事更容易 review

为什么应由维护者主导：

- 第二个真实 provider 会强烈影响社区对扩展风格的预期
- 目前仍应由维护者先定义传输隔离和文档质量基线

## 推荐执行顺序

1. 继续扩大 parser fixture 覆盖。
2. 建立 driver 合规测试支架。
3. 提升本地 search 语义。
4. 改善出站体验。
5. 增加一个新的 provider。

## 可以往后放的事

这些都重要，但不应该抢走下一阶段核心五项的主线位置：

- 文档对齐收口
- parser contributor guide
- 更大范围的 provider 扩展
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
- [Parser 贡献指南](../contributing/parser.md)
