[English](../../en/project/next-roadmap.md) | 中文

# 下一阶段开发路线

这份文档定义了 `v0.1.0-rc1` 之后推荐的开发顺序。

目标不是继续随意堆功能。

目标是把当前 RC 打磨成一个稳定、易贡献、真正服务于 agent 开发者的开源项目。

下一阶段还有一个明确的产品技术方向：**核心运行时和官方示例全面 Go 化**。MailCLI 应该交付一个方便 Agent 调用、方便用户安装、方便维护者发布的 Go binary，而不是要求用户理解或部署另一套语言运行时。

## `main` 上最近已经完成的部分

RC 收口阶段最关键的一批工作已经在 `main` 上完成：

- 为 `parse`、`get`、`search`、`threads`、`thread` 增加 CLI JSON 快照覆盖
- 强化 HTML 主体提取与降噪
- 更激进但仍有边界的追踪 URL 归一化
- 更清晰的 `sync` 索引状态输出
- 更丰富的 thread triage 信号，如 `code_count`、`action_count`、`participant_count`
- 一套标准化的 maintainer 入口，用于刷新和校验 local thread demo 固定产物
- 为契约敏感改动增加 RFC issue 模板
- 可复用的 driver 合同测试支架 `pkg/driver/drivertest`
- parser contributor workflow 文档
- 更可靠的本地 search / thread 排序语义
- 零网络优先的 README onboarding，以及展示出站默认推导的最小 reply 样例

如果你需要可直接复制到 GitHub 的 issue 草案，见 [GitHub Backlog 草案](github-backlog.md)。
如果你需要更现实的维护者主导开发顺序，见 [内部主导开发顺序](internal-priority.md)。

## 优先级顺序

1. 先把现有邮箱接入和配置体验收紧为 Go core 能力。
2. 让 inbox / thread 摘要、优先级、待办提取成为 Go 提供的结构化命令。
3. 继续增强附件、发票、验证码、链接动作提取，落在 Go parser / schema。
4. 把草稿、确认、执行、操作日志做成 Go CLI 契约。
5. 暂不优先处理专用 Agent mailbox 或更大 provider 扩展。

## Maintainer 规则

- 稳定的机器接口优先于功能数量
- 官方可执行能力由 Go 实现；不要引入 Go 之外的运行时作为用户安装或 Agent 调用的前置条件
- AI provider 接入保持语言无关的 JSON 契约，同时官方示例和长期维护路径以 Go 为主
- provider 私有业务逻辑不要进入共享 parser / composer 层
- parser heuristic 不是“清理工作”，而是核心产品工作
- 既要优化“好用”，也要优化“好贡献”

## 下一阶段 Go 主线

这四条是当前产品方向认可后的主线：

1. **现有邮箱接入和配置体验：Go core。**
   重点是让普通用户把已有 Gmail、Outlook、QQ、163、企业邮箱或本地 `.eml` 接进 MailCLI，而不是先要求他们拥有专用 Agent 邮箱。
   第一阶段的具体落点是 Go-only 配置路径：`config init`、`config doctor`、`config test` 和 `config capabilities`。
2. **Inbox / thread 摘要、优先级、待办提取：Go 提供结构化数据和命令，AI provider 可外接。**
   Go 负责稳定检索、聚合、字段输出和轻量规则；LLM 负责解释、归纳和生成建议。
3. **附件、发票、验证码、链接动作提取：Go parser / schema。**
   这些是 Agent 处理真实邮箱时最常遇到的高价值信息，应成为核心 parser 质量基线。
4. **草稿、确认、执行、操作日志：Go CLI 契约。**
   自动化必须可控。危险动作应先形成可审计意图，再确认执行，并留下机器可读结果。

## Milestone 1: v0.1 收口

目标：让当前 RC 更可信、更好文档化，也更适合他人基于它集成。

### 完成标准

- README、release notes 和 spec 对稳定边界的描述一致
- 核心命令的 CLI JSON 输出有 snapshot 测试覆盖
- 文档明确标注哪些 parser 行为仍然属于 heuristic
- 贡献者清楚什么时候必须走 RFC，而不是直接提功能 PR

### 建议 GitHub issues

#### Issue: 对齐 RC 文档与实际能力

状态：当前 `main` 的基线对齐已完成，但后续仍需随着契约演进持续维护。

- Area: docs
- Problem: roadmap 和状态说明容易与实际命令 / schema 支持脱节
- Scope:
  - 对齐 README 路线图勾选状态与当前实现
  - 校验 release notes 与 specs 指向同一组稳定契约
  - 增加一份 maintainer 视角的下一阶段路线文档
- Deliverable: 纯文档 PR

#### Issue: 为核心 CLI 命令增加 JSON 契约快照测试

状态：当前核心命令集已完成。

- Area: cmd, schema, tests
- Problem: agent 集成高度依赖稳定输出结构
- Scope:
  - 为 `parse`、`get`、`search`、`threads`、`thread` 增加 golden / snapshot 覆盖
  - 固定 `v0.1` 计划承诺稳定的字段
  - 文档化哪些字段仍属于 heuristic
- Deliverable: 测试 + 小范围 spec / docs 更新

#### Issue: 增加契约变更 RFC 模板

状态：已完成。

- Area: docs, governance
- Problem: schema 和 CLI 的变更需要可预期的讨论路径
- Scope:
  - 增加 GitHub issue template，用于 RFC 风格提案
  - 说明哪些变更必须走 RFC，而不是普通 feature request
  - 在 driver / schema 文档中指向这条路径
- Deliverable: `.github` 模板 + 文档更新

## Milestone 2: Parser / Schema 质量

目标：优先打磨最直接影响 agent 使用价值的部分。

### 完成标准

- 对噪音 HTML 模板的主体提取更稳定
- 对追踪跳转链接的清洗更激进但仍可控
- 对附件、发票、验证码、链接动作的结构化输出更完整
- fixture 覆盖更贴近真实 agent 工作流
- parser 回归更容易在发版前被发现

### 建议 GitHub issues

#### Issue: 强化 HTML 主体提取与降噪

状态：基线增强已完成，但这仍然是持续演进的 parser 产品工作。

- Area: parser
- Problem: 某些模板仍会把导航、页脚或布局噪音带进 `body_md`
- Scope:
  - 在 HTML-to-Markdown 之前提升主内容识别能力
  - 保留链接、标题、表格、关键图片等有价值结构
  - 为当前容易退化的 newsletter、alert、transactional 布局增加 fixture
- Deliverable: parser 改动 + golden tests

#### Issue: 提升 agent-facing action URL 的归一化能力

状态：基线增强已完成，后续更应优先补 fixture，而不是默认继续放宽规则。

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

#### Issue: 将入站附件和发票入口提升为一等结构化输出

- Area: parser, schema, cmd
- Problem: 普通邮箱中的高价值信息经常藏在附件、发票入口、下载链接或正文 URL 中，agent 不应该只靠全文猜测。
- Scope:
  - 在 `StandardMessage` 中设计稳定的入站附件 / 附件入口表达
  - 区分真实 MIME 附件、正文中的附件下载入口、发票查看 / 下载动作
  - 为发票、附件通知、多语言验证码增加 fixture 和 golden 覆盖
- Deliverable: schema / parser 改动、CLI 输出、测试和 spec 更新

## Milestone 3: Inbox / Thread Intelligence

目标：让 AI 基于用户已有邮箱更快、更准确地获取信息，而不是只做关键词搜索。

### 完成标准

- `sync` 的重复运行语义更容易理解
- 用户和贡献者更容易看清楚本地缓存里到底有什么
- thread 元数据足够减少不必要的完整消息加载
- inbox / thread 摘要能支持优先级、待办、需要回复等常见 triage 判断

### 建议 GitHub issues

#### Issue: 更清晰地暴露本地索引和 sync 状态

状态：sync / index 可见性的基础增强已完成。

- Area: cmd, internal/index, docs
- Problem: agent 和贡献者需要更明确地知道本地缓存状态
- Scope:
  - 通过 CLI 输出暴露基本 sync / index 统计信息
  - 文档化 `sync` 会跳过、覆盖和刷新哪些内容
  - 让本地检索语义更容易解释给开发者
- Deliverable: 命令输出增强 + 文档

#### Issue: 为 triage 循环增强 thread 摘要

状态：thread 摘要的基线增强和本地排序语义收紧已完成。

- Area: internal/index, cmd, schema
- Problem: agent 在常见分拣场景里仍需要加载过多完整 thread
- Scope:
  - 评估增加更清晰的 latest timestamp、participant summary、action/code counts 等紧凑字段
  - 保持输出体积足够小，适合 prompt 使用
  - 对最终 shape 做 snapshot 测试
- Deliverable: schema / 输出增强 + 文档

#### Issue: 增加 inbox / thread 的优先级与待办提取信号

- Area: internal/index, cmd, schema
- Problem: 用户最常见的问题不是“搜到邮件”，而是“哪些邮件重要、哪些需要处理、下一步做什么”。
- Scope:
  - 设计轻量、可解释的 priority / needs_reply / todo-like 信号
  - 保持 Go 层输出为结构化候选信号，不在 core 中绑定特定 LLM
  - 为本地 fixture 和 thread demo 增加固定输出覆盖
- Deliverable: Go 命令输出、schema / spec 更新、snapshot 测试

## Milestone 4: 安全出站闭环与操作日志

目标：让 AI 自动化从“直接执行命令”升级为“生成意图、确认执行、记录结果”的可控闭环。

### 完成标准

- `send` / `reply` / `delete` / `move` / `mark` 等高影响动作可以先生成 dry-run / intent
- 确认执行有稳定 token 或 intent id，避免 agent 误操作
- 执行结果和失败原因进入机器可读操作日志
- 操作日志不依赖 provider 私有行为

### 建议 GitHub issues

#### Issue: 为危险动作增加 prepare / confirm 流程

- Area: cmd, schema, docs
- Problem: Agent 可以生成发送、删除、移动等动作，但直接执行会放大误操作风险。
- Scope:
  - 为发送和 mailbox mutation 设计 `prepare` 输出
  - 使用 intent id 或确认 token 执行同一意图
  - 保持 dry-run、prepare、confirm 的输出都适合 Agent 读取
- Deliverable: Go CLI 契约、schema、测试和文档

#### Issue: 增加本地操作日志

- Area: cmd, internal, docs
- Problem: Agent 自动化需要事后审计：执行了什么、为什么失败、对应哪个 message / thread。
- Scope:
  - 记录操作类型、账户、目标 ID、intent id、结果、错误码和时间
  - 提供 `mailcli operations list/show` 或等价查询入口
  - 避免记录秘密字段和完整敏感正文
- Deliverable: Go 存储 / CLI、测试和安全说明

## 暂缓：Provider 扩展与专用 Agent Mailbox

腾讯 Agent Mail 证明了“专用 Agent 邮箱身份”有价值，但 MailCLI 当前主线不是托管邮箱服务。下一阶段应优先帮助用户用 AI 处理已有邮箱。

因此暂缓：

- 内建重 OAuth 流程
- 提供托管式 `@agent` 邮箱身份
- 大范围 provider 扩展
- 运行时插件加载

后续只有在 Go core 的接入、检索、提取、确认执行闭环稳定之后，才重新评估专用 Agent mailbox 或新增 provider。

## 建议在 GitHub 中建立的 Milestones

- `v0.1 hardening`
- `go-only core`
- `existing mailbox setup`
- `inbox intelligence`
- `parser actions and attachments`
- `safe outbound automation`

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
- 提供托管式专用 Agent mailbox
- 运行时插件加载
- 在共享层中加入 provider 私有业务策略
- 试图一次性解决所有邮箱厂商
- 在 Go 之外再引入第二条官方运行时路径

最强的下一步，不是把边界做得更宽，而是用 Go 把“AI 安全处理用户已有邮箱”的边界做得更锋利。
