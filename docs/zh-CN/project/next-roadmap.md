[English](../../en/project/next-roadmap.md) | 中文

# 下一阶段开发路线

这份文档定义了 `v0.1.0-rc1` 之后推荐的开发顺序。

目标不是继续随意堆功能。

目标是把当前 RC 打磨成一个稳定、易贡献、真正服务于 agent 开发者的开源项目。

如果你需要可直接复制到 GitHub 的 issue 草案，见 [GitHub Backlog 草案](github-backlog.md)。

## 优先级顺序

1. 先把当前 RC 的稳定边界收紧。
2. 优先提升 agent 当前最有痛感的 parser 质量。
3. 继续增强本地 memory 和 thread 工作流的可靠性。
4. 降低新 driver / parser 贡献者的接入门槛。
5. 在共享契约更清晰之后，再扩 provider 生态。

## Maintainer 规则

- 稳定的机器接口优先于功能数量
- provider 私有业务逻辑不要进入共享 parser / composer 层
- parser heuristic 不是“清理工作”，而是核心产品工作
- 既要优化“好用”，也要优化“好贡献”

## Milestone 1: v0.1 收口

目标：让当前 RC 更可信、更好文档化，也更适合他人基于它集成。

### 完成标准

- README、release notes 和 spec 对稳定边界的描述一致
- 核心命令的 CLI JSON 输出有 snapshot 测试覆盖
- 文档明确标注哪些 parser 行为仍然属于 heuristic
- 贡献者清楚什么时候必须走 RFC，而不是直接提功能 PR

### 建议 GitHub issues

#### Issue: 对齐 RC 文档与实际能力

- Area: docs
- Problem: roadmap 和状态说明容易与实际命令 / schema 支持脱节
- Scope:
  - 对齐 README 路线图勾选状态与当前实现
  - 校验 release notes 与 specs 指向同一组稳定契约
  - 增加一份 maintainer 视角的下一阶段路线文档
- Deliverable: 纯文档 PR

#### Issue: 为核心 CLI 命令增加 JSON 契约快照测试

- Area: cmd, schema, tests
- Problem: agent 集成高度依赖稳定输出结构
- Scope:
  - 为 `parse`、`get`、`search`、`threads`、`thread` 增加 golden / snapshot 覆盖
  - 固定 `v0.1` 计划承诺稳定的字段
  - 文档化哪些字段仍属于 heuristic
- Deliverable: 测试 + 小范围 spec / docs 更新

#### Issue: 增加契约变更 RFC 模板

- Area: docs, governance
- Problem: schema 和 CLI 的变更需要可预期的讨论路径
- Scope:
  - 增加 GitHub issue template，用于 RFC 风格提案
  - 说明哪些变更必须走 RFC，而不是普通 feature request
  - 在 driver / schema 文档中指向这条路径
- Deliverable: `.github` 模板 + 文档更新

## Milestone 2: Parser 质量

目标：优先打磨最直接影响 agent 使用价值的部分。

### 完成标准

- 对噪音 HTML 模板的主体提取更稳定
- 对追踪跳转链接的清洗更激进但仍可控
- fixture 覆盖更贴近真实 agent 工作流
- parser 回归更容易在发版前被发现

### 建议 GitHub issues

#### Issue: 强化 HTML 主体提取与降噪

- Area: parser
- Problem: 某些模板仍会把导航、页脚或布局噪音带进 `body_md`
- Scope:
  - 在 HTML-to-Markdown 之前提升主内容识别能力
  - 保留链接、标题、表格、关键图片等有价值结构
  - 为当前容易退化的 newsletter、alert、transactional 布局增加 fixture
- Deliverable: parser 改动 + golden tests

#### Issue: 提升 agent-facing action URL 的归一化能力

- Area: parser
- Problem: 动作链接里仍可能带有 provider 追踪包装，影响 agent 理解
- Scope:
  - 清洗常见跳转模式，同时避免破坏真实目标链接
  - 在有需要时保留原始 URL 供调试
  - 覆盖 unsubscribe、invoice、reset、attachment 等入口
- Deliverable: parser 改动、测试和 spec 注释

#### Issue: 扩大真实世界边界场景的 parser 样本集

- Area: parser, tests
- Problem: parser 的上限取决于 fixture 语料的覆盖度
- Scope:
  - 增加更多脱敏后的多语言、thread-heavy、bounce、layout-heavy 邮件样本
  - 按行为类型而不是 provider 品牌组织 fixture
  - 说明每个 fixture 保护的行为是什么
- Deliverable: 新 fixture + 定向回归测试

## Milestone 3: Local Memory 与 Threads

目标：让本地 agent 检索体验从“能用”变成“可靠”。

### 完成标准

- `sync` 的重复运行语义更容易理解
- 用户和贡献者更容易看清楚本地缓存里到底有什么
- thread 元数据足够减少不必要的完整消息加载

### 建议 GitHub issues

#### Issue: 更清晰地暴露本地索引和 sync 状态

- Area: cmd, internal/index, docs
- Problem: agent 和贡献者需要更明确地知道本地缓存状态
- Scope:
  - 通过 CLI 输出暴露基本 sync / index 统计信息
  - 文档化 `sync` 会跳过、覆盖和刷新哪些内容
  - 让本地检索语义更容易解释给开发者
- Deliverable: 命令输出增强 + 文档

#### Issue: 为 triage 循环增强 thread 摘要

- Area: internal/index, cmd, schema
- Problem: agent 在常见分拣场景里仍需要加载过多完整 thread
- Scope:
  - 评估增加更清晰的 latest timestamp、participant summary、action/code counts 等紧凑字段
  - 保持输出体积足够小，适合 prompt 使用
  - 对最终 shape 做 snapshot 测试
- Deliverable: schema / 输出增强 + 文档

## Milestone 4: Contributor Surface

目标：降低外部开发者做出有效 PR 的成本。

### 完成标准

- 新增 driver 的方式容易理解，也容易验证
- parser 贡献者有清晰的测试和入口说明
- 契约变更有一条不依赖“口口相传”的讨论路径

### 建议 GitHub issues

#### Issue: 为 driver 扩展贡献者增加 fake-driver 测试支架

- Area: driver, tests, docs
- Problem: 外部 driver 的验证门槛仍偏高
- Scope:
  - 增加可复用的 driver 合规测试辅助或 fixture
  - 明确 list / fetch / send 的最小行为要求
  - 文档化最低接受门槛
- Deliverable: 测试支架 + 贡献文档

#### Issue: 编写 parser contributor guide

- Area: docs
- Problem: parser 是高价值区域，但当前进入路径仍然偏隐性
- Scope:
  - 说明 fixture 布局、golden tests 和 parser 设计约束
  - 说明哪些 heuristic 是允许的，哪些地方不允许随意漂移
  - 指向最相关的 parser 包与测试文件
- Deliverable: 新贡献文档

## Milestone 5: Provider 扩展

目标：在不破坏共享模型的前提下扩展生态。

这个里程碑应该建立在前几个阶段已经基本收口之后。

### 建议 GitHub issues

#### Issue: 增加一个新的内置 provider，并配齐测试与文档

- Area: driver, docs
- Problem: 当仓库里不止一种真实接入路径时，生态故事会更有说服力
- Scope:
  - 实现一个额外的 provider 或 provider 风格
  - 保持传输逻辑与 parser / composer 逻辑分离
  - 补齐配置、限制与测试策略文档
- Deliverable: 新 driver + 测试 + 文档

#### Issue: 定义 driver 合规性检查清单

- Area: docs, tests
- Problem: 社区 driver 需要共享质量门槛
- Scope:
  - 定义 list、fetch、send、config validation 的必备行为
  - 把这些期望映射到测试和贡献说明
  - 保持清单足够稳定，便于 PR review 直接引用
- Deliverable: spec 或 contributing 文档更新

## 建议在 GitHub 中建立的 Milestones

- `v0.1 hardening`
- `parser quality`
- `local memory`
- `contributor surface`
- `provider expansion`

## 建议标签

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

## 暂时不要优先做的事

- 完整终端 mail client 体验
- 在 core 中引入重 OAuth 认证流
- 运行时插件加载
- 在共享层中加入 provider 私有业务策略
- 试图一次性解决所有邮箱厂商

最强的下一步，不是把边界做得更宽，而是把当前 agent 边界做得更锋利。
