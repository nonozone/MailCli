# MailCLI Positioning and Project Design

## 1. Summary / 摘要

`MailCLI` is an independent, open-source, AI-native email normalization toolkit.

`MailCLI` 是一个独立开源、面向 AI 的邮件标准化工具集。

It is not designed to compete with existing terminal-first mail clients. Its purpose is to provide AI agents, automation systems, and developers with a stable interface for reading, normalizing, and acting on email.

它的目标不是与现有终端邮件客户端竞争，而是为 AI agent、自动化系统和开发者提供一个稳定的邮件读取、标准化与动作提取接口。

Project direction:

项目方向：

- Open standard first
- AI-first product definition
- Provider-agnostic architecture
- Community-open contribution model
- GitHub-friendly bilingual documentation

- 以开放标准为优先
- 以 AI 为第一目标用户
- 保持 provider 无关架构
- 采用社区可全面参与的治理模型
- 优先建设 GitHub 友好的中英双语文档

## 2. Positioning / 项目定位

### English

`MailCLI` is an open, AI-native email normalization toolkit for agents, automation, and developer workflows.

It turns raw email into structured, provider-agnostic outputs such as normalized JSON, Markdown, actions, and thread context, so AI systems can work with email without dealing with MIME complexity, HTML noise, or provider-specific APIs.

### 中文

`MailCLI` 是一个面向 agent、自动化与开发者工作流的开放式 AI 原生邮件标准化工具集。

它将原始邮件转换为结构化、与 provider 无关的输出，例如标准化 JSON、Markdown、动作信息以及线程上下文，使 AI 系统无需直接处理 MIME 复杂性、HTML 噪音和各家邮件服务的专有接口。

### One-line statement / 一句话表述

`MailCLI aims to become the open standard interface between email and AI.`

`MailCLI` 的目标是成为连接邮件与 AI 的开放标准接口。

## 3. Primary Users / 首要用户

### Primary audience / 第一优先级用户

- AI agent developers
- Automation builders
- Developers integrating email into AI workflows
- Contributors working on parsers, providers, and standards

- AI agent 开发者
- 自动化流程开发者
- 将邮件接入 AI 工作流的开发者
- 参与 parser、provider、标准定义的开源贡献者

### Explicitly not the primary audience / 明确不是优先目标用户

- Traditional terminal mail client power users
- Users looking for a full human-first email UI in the terminal

- 传统终端邮件客户端重度用户
- 需要完整“人类手工收件体验”的终端用户

## 4. Core Value Proposition / 核心价值主张

### English

`MailCLI` creates value by moving email handling up from protocol noise to AI-usable context.

It should make the following workflow simple:

1. Read raw email from a file, stdin, IMAP, or another provider
2. Parse and normalize content into a standard representation
3. Extract actions, reply context, and machine-usable metadata
4. Return outputs that an agent can consume directly

### 中文

`MailCLI` 的核心价值，是把邮件处理从“协议与格式噪音”提升为“AI 可直接消费的上下文”。

它应该把以下流程做得足够简单：

1. 从文件、stdin、IMAP 或其他 provider 读取原始邮件
2. 将内容解析并标准化为统一表示
3. 提取动作、回复上下文和机器可用的元数据
4. 输出 AI agent 可以直接使用的结果

### Core value bullets / 核心价值点

- Parse raw MIME into AI-friendly structured data
- Normalize emails across providers and protocols
- Extract actions such as unsubscribe, reply context, and delivery errors
- Provide a stable interface for automation and agents
- Let the ecosystem evolve without tying the project to a single backend

- 将原始 MIME 解析为 AI 友好的结构化数据
- 在不同 provider 与协议之间统一邮件表示
- 提取退订、回复上下文、投递错误等动作信息
- 为自动化与 agent 提供稳定接口
- 允许生态扩展，而不绑定单一后端

## 5. Non-goals / 不做什么

The project should explicitly avoid scope drift.

项目必须明确避免范围失控。

### Non-goals

- Not a traditional terminal mail client
- Not a replacement for existing power-user mail tools
- Not tied to any specific email provider
- Not a commercial backend disguised as open source
- Not an attempt to support every provider-specific feature in the core schema
- Not an early-stage all-in-one solution for sending, rules, UI, and account orchestration

### 非目标

- 不做传统终端邮件客户端
- 不替代现有面向高级用户的邮件工具
- 不绑定任何单一邮件服务商
- 不把商业后端包装成“伪开源核心”
- 不为了兼容每家 provider 的私有特性而污染核心 schema
- 不在早期追求发信、规则引擎、UI、账号编排等大而全能力

## 6. Product Boundaries / 产品边界

`MailCLI` should remain useful and credible as a standalone open-source project.

`MailCLI` 必须作为独立开源项目具备完整可用性与可信度。

### In scope for the open project / 属于开源核心范围

- Standard JSON schema
- Parser engine
- CLI core
- Provider and driver interfaces
- Baseline IMAP and SMTP support
- Test corpus and golden outputs
- Examples for shell, Python, and agent integrations
- Governance and contribution documentation

- 标准 JSON schema
- Parser engine
- CLI core
- Provider 与 driver 接口
- 基础 IMAP / SMTP 支持
- 测试语料与 golden outputs
- Shell、Python、agent 集成示例
- 治理与贡献文档

### Guiding rule / 边界原则

The open-core project must stand on its own. External providers or service integrations may improve the experience, but they must not define the project.

开源核心必须能够独立成立。外部 provider 或服务集成可以提升体验，但不能反过来定义项目本身。

## 7. Architecture Direction / 架构方向

The architecture should preserve a clean split between transport, parsing, and command execution.

架构上应当保持传输层、解析层与命令执行层的清晰解耦。

### Modules / 模块

1. `Cmd`
   Handles CLI commands, account context, stdin/file input, and output formatting.

   负责命令行入口、账户上下文、stdin / 文件输入以及输出格式化。

2. `Driver`
   Responsible for fetching raw email and sending raw bytes. It should not own content parsing.

   负责获取原始邮件与发送原始字节流，不承担正文语义解析职责。

3. `Parser`
   Responsible for MIME traversal, charset normalization, HTML cleanup, Markdown conversion, action extraction, and normalized structured output.

   负责 MIME 遍历、字符集归一化、HTML 清洗、Markdown 转换、动作提取以及标准结构输出。

4. `Composer`
   Responsible for turning agent-generated drafts into standards-compliant outgoing email, including MIME assembly, thread headers, and message encoding.

   负责将 agent 生成的草稿编译为标准邮件，包括 MIME 组装、线程头补全和消息编码。

### Architectural principle / 架构原则

`protocol belongs to drivers, content belongs to parsers, orchestration belongs to the CLI core`

`协议归驱动，内容归解析器，流程调度归 CLI 核心`

### Read path and write path / 读路径与写路径

Read path:

读路径：

`Agent -> MailCLI -> Driver -> Raw Email -> Parser -> StandardMessage`

Write path:

写路径：

`Agent -> MailCLI -> DraftMessage/ReplyDraft -> Composer -> Raw MIME -> Driver -> Provider`

### Outbound schema principle / 发送侧 schema 原则

Agents should not construct raw MIME directly.

agent 不应直接拼装原始 MIME。

Instead, they should produce intent-level objects:

更合理的做法是生成意图级对象：

- `DraftMessage` for new outgoing email
- `ReplyDraft` for replies tied to an existing message or internal message id
- `SendResult` for transport result reporting

- `DraftMessage` 用于新邮件
- `ReplyDraft` 用于回复既有邮件或内部 message id
- `SendResult` 用于返回发送结果

The CLI core should compile these into standards-compliant outbound email and automatically manage:

CLI 核心应负责把这些对象编译为标准邮件，并自动处理：

- `Message-ID`
- `Date`
- `From`
- `In-Reply-To`
- `References`
- `multipart/alternative`
- text / HTML generation
- attachment packaging
- charset and transfer encoding

## 8. MVP Strategy / MVP 策略

The first release should optimize for proof of value, not feature breadth.

首个版本应优先验证价值，而不是追求功能覆盖面。

### Recommended first slice / 建议的第一刀

Parser-first MVP:

- `mailcli parse file.eml`
- `cat file.eml | mailcli parse`
- output normalized JSON / Markdown
- extract unsubscribe and delivery-error actions
- extract thread headers and metadata
- validate with a public corpus of representative emails

Parser-first MVP：

- `mailcli parse file.eml`
- `cat file.eml | mailcli parse`
- 输出标准化 JSON / Markdown
- 提取退订与投递错误动作
- 提取线程头部与元数据
- 使用公开的代表性邮件语料进行验证

### Why this slice / 为什么先做这一刀

- It proves the core idea quickly
- It is easy to test offline
- It highlights the parser as the real differentiator
- It is directly useful for AI developers

- 能最快验证项目核心价值
- 可离线测试，降低接入门槛
- 能直接凸显 parser 才是差异化能力
- 对 AI 开发者有立即可见的实用价值

## 9. Governance Model / 治理模型

The project should be community-open, with maintainers protecting compatibility and quality.

项目应保持社区开放，但由维护者承担兼容性与质量把关责任。

### Governance rules / 治理规则

- Community contributions are welcome across all layers
- Maintainers review and protect compatibility-sensitive changes
- Specification changes should follow a lightweight RFC process
- Parser behavior changes that affect normalized outputs need higher scrutiny
- The project should present itself as community-driven and maintainer-guided

- 社区可参与所有层级的贡献
- 维护者重点审查影响兼容性的改动
- 标准层改动应走轻量 RFC 流程
- 会影响标准化输出的 parser 变更需要更严格审查
- 项目对外应呈现“社区驱动、维护者引导”的治理风格

### Sensitive areas / 重点把关区域

- `Standard JSON Spec`
- provider / plugin interfaces
- parser output semantics
- compatibility with existing examples and golden tests

- `Standard JSON Spec`
- provider / plugin 接口
- parser 输出语义
- 与既有示例及 golden tests 的兼容性

## 10. Licensing Recommendation / 授权建议

Recommended license: `Apache-2.0`

建议许可证：`Apache-2.0`

### Reasoning / 原因

- Encourages broad adoption
- Friendly to company and community use
- Supports ecosystem growth
- Fits the goal of becoming an open standard

- 有利于广泛采用
- 对公司与社区使用都更友好
- 有利于生态扩张
- 符合“成为开放标准”的目标

## 11. Documentation Strategy / 文档策略

Documentation should be part of the product, not an afterthought.

文档应该是产品的一部分，而不是事后补充。

### Documentation requirements / 文档要求

- Chinese and English should both exist from the beginning
- Do not use long-form bilingual line-by-line mixed documents as the default format
- Prefer one document per language, with quick links at the top for switching
- Written in plain Markdown first
- Optimized for GitHub browsing, linking, and indexing
- Structured so contributors can extend it gradually

- 从一开始就同时提供中文与英文文档
- 不采用长期的中英逐段对照混排作为默认格式
- 优先按语言拆分文件，并在文档顶部提供快捷切换链接
- 优先使用纯 Markdown
- 针对 GitHub 浏览、链接与索引优化
- 结构上允许贡献者逐步扩展

### Preferred language layout / 推荐语言组织方式

Use separate files per language:

按语言拆分文件：

- `README.md` for English
- `README.zh-CN.md` for Chinese
- topic docs in `docs/en/` and `docs/zh-CN/`
- each document begins with cross-links such as `English | 中文`

- 英文使用 `README.md`
- 中文使用 `README.zh-CN.md`
- 主题文档放在 `docs/en/` 与 `docs/zh-CN/`
- 每篇文档顶部提供 `English | 中文` 这样的互链入口

### Phase 1 / 第一阶段

Use:

- root language entry documents
- `docs/en/` for English topics
- `docs/zh-CN/` for Chinese topics

采用：

- 根目录语言入口文档
- `docs/en/` 存放英文主题文档
- `docs/zh-CN/` 存放中文主题文档

### Phase 2 / 第二阶段

An open-source documentation site can be added later and maintained by the community, but the Markdown source of truth should remain in the repository.

后续可以增加独立的开源文档站，并允许社区共同维护，但 Markdown 仍应是仓库中的事实来源。

### Recommended documentation tree / 建议文档结构

```text
/
  README.md
  README.zh-CN.md
  CONTRIBUTING.md
  CONTRIBUTING.zh-CN.md
  GOVERNANCE.md
  GOVERNANCE.zh-CN.md
  LICENSE
  docs/
    en/
      overview.md
      architecture.md
      spec/
        standard-message.md
        provider-interface.md
      parser/
        mime-handling.md
        html-cleaning.md
        actions.md
      examples/
        python.md
        shell.md
        agent.md
      roadmap.md
      faq.md
    zh-CN/
      overview.md
      architecture.md
      spec/
        standard-message.md
        provider-interface.md
      parser/
        mime-handling.md
        html-cleaning.md
        actions.md
      examples/
        python.md
        shell.md
        agent.md
      roadmap.md
      faq.md
  rfcs/
```

## 12. Repository Structure Direction / 仓库结构建议

Recommended initial structure:

建议的初始结构：

```text
/mailcli
  /cmd
  /pkg
    /driver
    /parser
    /schema
  /internal
    /config
  /examples
  /testdata
  /docs
  /rfcs
```

### Notes / 说明

- `testdata/` should contain representative email samples and golden outputs
- `examples/` should show practical AI and automation usage
- `rfcs/` should hold schema and interface evolution proposals

- `testdata/` 应存放代表性邮件样本与 golden outputs
- `examples/` 应提供真实的 AI 与自动化使用案例
- `rfcs/` 应用于存放 schema 与接口演进提案

## 13. Messaging Guidance / 对外表述建议

### Preferred language / 推荐说法

- `MailCLI is an open, AI-native email normalization toolkit.`
- `MailCLI helps agents and developers work with email as structured context instead of raw MIME.`
- `MailCLI aims to become the open standard interface between email and AI.`

- `MailCLI 是一个开放的、面向 AI 的邮件标准化工具集。`
- `MailCLI 帮助 agent 与开发者以结构化上下文而不是原始 MIME 的方式处理邮件。`
- `MailCLI 的目标是成为连接邮件与 AI 的开放标准接口。`

### Language to avoid / 避免的说法

- claiming it is a terminal mail client replacement
- presenting it as tied to a single provider
- mixing open project identity with business integrations

- 宣称它是终端邮件客户端替代品
- 将其描述为绑定某个单一 provider
- 在项目定义中混入具体业务集成叙事

## 14. Contributor Guidance / 贡献者机制说明

Contributors should understand that `MailCLI` is not only a parser project.

贡献者需要明确：`MailCLI` 不只是一个 parser 项目。

It has two equally important contracts:

它有两类同等重要的协议边界：

1. inbound normalization
2. outbound composition

1. 入站标准化
2. 出站编译

Inbound work should improve:

入站方向应重点改进：

- MIME traversal
- content extraction
- Markdown quality
- action extraction
- thread and metadata normalization

Outbound work should improve:

出站方向应重点改进：

- draft schemas
- reply threading behavior
- MIME generation
- attachment support
- provider-safe send behavior

When contributors touch parser semantics or outbound schemas, they should also update the public spec docs so the project remains understandable to new collaborators.

当贡献者修改 parser 语义或发送侧 schema 时，也应同步更新公开规范文档，确保新加入的开发者能快速理解项目机制。

## 15. Next Step / 下一步

The next planning step should focus on implementation sequencing, starting with a parser-first MVP.

下一步应进入实施规划，优先围绕 parser-first MVP 制定实现路径。

Recommended plan entry:

建议的实施入口：

1. Define the standard message schema
2. Build the parser for local `.eml` and stdin input
3. Add golden tests using representative samples
4. Expose the parser through a minimal Cobra CLI
5. Add baseline configuration and provider interfaces

1. 定义标准消息 schema
2. 完成面向本地 `.eml` 与 stdin 的 parser
3. 用代表性样本建立 golden tests
4. 通过最小化 Cobra CLI 暴露 parser 能力
5. 加入基础配置系统与 provider 接口
