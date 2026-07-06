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

这意味着下一阶段不应该继续停留在“边界是否成立”，而应该转向 Go-only 核心、现有邮箱体验、inbox intelligence 和安全自动化闭环。

## 当前工作假设

接下来一个阶段，可以把 MailCLI 看成：

- 官方可执行能力由 Go core 承担
- 核心契约、parser 质量、本地 memory 模型由维护者主导
- AI provider 通过语言无关 JSON 契约外接，不进入 core
- Go 示例是官方主路径；不再维护第二条官方运行时路径
- 文档、fixtures、examples、部分 contributor surface 由社区辅助

## 下一阶段核心四项

这四个任务应该被视为接下来内部开发的主线顺序。第五项“专用 Agent mailbox / 更大 provider 扩展”暂时不处理。

### 1. 现有邮箱接入和配置体验：Go core

为什么排第一：

- 大多数用户已经有 Gmail、Outlook、QQ、163 或企业邮箱，不会先拥有专用 Agent 邮箱
- 用户真正需要的是让 AI 安全理解和处理已有邮箱
- 安装、配置、连接测试、账户能力说明都应该由 Go binary 提供，不依赖另一套语言运行时
- 第一阶段的实现切片应让 `config init`、`config doctor`、`config test`、`config capabilities` 分别覆盖配置创建、静态诊断、联网检查和机器可读能力发现

为什么应由维护者主导：

- 这会定义 MailCLI 面向普通用户的第一体验
- 配置契约和账户能力输出会直接影响后续所有 Agent 工作流

## 2. Inbox / thread 摘要、优先级、待办提取

为什么排第二：

- 用户想要的是“哪些邮件重要、哪些需要回复、下一步是什么”
- 本地 thread / search 已经可用，下一步应该把它提升为可解释的 triage 信号
- Go 层应提供结构化候选数据，LLM provider 只负责外接解释和生成建议

为什么应由维护者主导：

- 摘要和优先级信号会影响对外 JSON 形状和 prompt 使用方式
- 本地 memory 语义一旦做差，很快会污染示例和用户工作流

## 3. 附件、发票、验证码、链接动作提取

为什么排第三：

- 这些是普通邮箱中最常见、最有自动化价值的信息
- 现有 action / code 基线已经成立，后续应扩大到入站附件和发票入口
- 这比新增 provider 更直接提升用户已有邮箱的 AI 可用性

为什么应由维护者主导：

- parser / schema 的字段设计就是产品质量基线
- fixture 选择和 golden 输出需要体现维护者对真实邮件场景的判断

## 4. 草稿、确认、执行、操作日志

为什么排第四：

- 自动化必须可控，尤其是发送、删除、移动、标记这类高影响动作
- 现有 `send` / `reply` / mailbox mutation 已经能用，下一步应补 prepare / confirm / log
- 这能让 Agent 从“会调用命令”升级为“有审计边界地执行”

为什么应由维护者主导：

- 它接近公开稳定 CLI 契约和用户信任边界
- 确认 token、intent id、操作日志字段需要一开始就足够稳定

## 推荐执行顺序

1. 收紧现有邮箱接入、配置、账户能力和 Go-first 示例。
2. 增强 inbox / thread 摘要、优先级、待办提取。
3. 增强附件、发票、验证码、链接动作提取。
4. 增加草稿 / 确认 / 执行 / 操作日志闭环。

## 可以往后放的事

这些都重要，但不应该抢走下一阶段核心四项的主线位置：

- 文档对齐收口
- parser contributor guide
- 更大范围的 provider 扩展
- 更广的社区流程建设
- 专用 Agent mailbox / 托管邮箱身份
- 在 Go 之外继续维护第二条官方运行时路径

它们是围绕产品核心的支撑工作，不是通向更强 `v0.2` 的主路径。

## 适合社区并行辅助的工作

当维护者专注这四项时，社区仍然可以帮助推进：

- fixture 收集和脱敏
- examples
- 文档润色
- 小型 parser regression 报告
- contributor guides
- 不改变契约的测试补充
- 非官方语言示例，但不能让它们成为安装或使用前置条件

## 一条简单判断规则

如果一个任务会改变：

- 对外 JSON 形状
- parser 质量基线
- thread 摘要语义
- 本地 memory 语义
- Go CLI 的执行 / 确认 / 审计契约

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
