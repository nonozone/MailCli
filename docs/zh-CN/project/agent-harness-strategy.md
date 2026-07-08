[English](../../en/project/agent-harness-strategy.md) | 中文

# Agent Harness 策略

这份文档定义 MailCLI 后续如何吸收 agent harness 的方法，而不是只把它当成“给 AI 用的命令行工具”。

结论先说清楚：

- Skill 解决的是“agent 需要知道怎么做某类事”。
- Harness 解决的是“一个长期任务如何被推进、暂停、验证、交接、审计和让人介入”。
- MailCLI 的产品机会不是把邮箱协议暴露给 AI，而是把邮件处理流程变成可执行、有证据、有边界的工作流。

## 外部对标

截至 2026-07-08，外部 agent / workflow 系统呈现出几条清晰路线：

| 系统 | 主要启发 | 对 MailCLI 的意义 |
| --- | --- | --- |
| [Specability](https://specability.com/approach/) | 先找不变量，再写 spec，最后让 harness 执行规则、暴露判断点、记录决策 | MailCLI 应把“哪些邮件动作必须确认、哪些字段不能泄漏、哪些输出稳定”写成可执行 contract |
| [Agent Skills](https://agentskills.io/) / [Claude Skills](https://docs.anthropic.com/en/docs/claude-code/skills) | Skill 是按需加载的知识、流程、脚本和参考资料包 | MailCLI 可以提供技能说明，但不能把稳定性寄托在 prompt 纪律上 |
| [LangGraph](https://docs.langchain.com/oss/python/langgraph/overview) | 长任务需要 durable execution、persistence、human-in-the-loop、stateful workflow | 邮件处理经常跨多封邮件、多轮确认和后续动作，不能只靠一次性命令 |
| [Deep Agents](https://docs.langchain.com/oss/python/deepagents/overview) | 现代 agent harness 内建 planning、文件系统、subagent、长期记忆和人工审批 | MailCLI 的未来 CLI 契约应支持计划、候选动作、审批和审计结果 |
| [Temporal](https://docs.temporal.io/temporal) | 长流程的关键是事件历史、可恢复状态和失败恢复，而不是“模型更聪明” | 邮件自动化需要 operation log 和 intent id，避免失败后无法解释 |
| [Microsoft Agent Framework](https://learn.microsoft.com/en-us/agent-framework/overview/) | Agent 适合开放对话，workflow 适合明确步骤；HITL 通过 request / response 暂停和恢复 | MailCLI 应把危险操作变成“请求人确认”的结构化状态，而不是一句自然语言提醒 |
| [OpenAI Agents SDK](https://openai.github.io/openai-agents-python/) | 生产级 agent runtime 需要 handoffs、guardrails、sessions、tracing、human-in-the-loop | MailCLI 不需要内建模型 runtime，但需要提供能被这些 runtime 安全调用的边界 |
| [Google ADK](https://google.github.io/adk-docs/agents/workflow-agents/) | 复杂 agent 系统需要 workflow agents、sessions、memory、artifacts、confirmation | 邮件处理结果应沉淀为 artifact / operation，而不是只存在对话里 |

## Harness 与 Skill 的区别

| 维度 | 普通 Skill | Harness |
| --- | --- | --- |
| 核心形态 | 一个可加载的说明包 | 一个运行时控制层 |
| 主要作用 | 告诉 agent 怎么做 | 决定下一步该做什么、是否允许做、做完如何证明 |
| 状态 | 通常依赖当前上下文 | 有 task / run / artifact / check / log |
| 失败处理 | agent 自己解释和重试 | 有可恢复状态、失败原因和下一步动作 |
| 人的介入 | agent 想起需要问时才问 | 在明确判断点暂停，收集人的回答后继续 |
| 验证 | 多靠说明和测试建议 | 以 gate、criterion、evidence、CI 或 verifier 作为完成条件 |
| 审计 | 通常没有长期记录 | 记录决策、操作、证据和责任边界 |

Skill 的价值仍然很大。它适合承载：

- 某个 provider 的接入指南
- 邮件 parser fixture 的贡献流程
- release checklist
- Claude / Codex / OpenAI Agents 的接入说明

但 Skill 不适合单独承担：

- 多天或多轮开发任务的推进
- 邮箱自动化的安全确认
- 对外 JSON contract 的稳定性
- 用户目标、验收标准和实际执行证据的对齐

## 跟人的交互应该有什么不同

Harness 风格的人机交互，不是“更频繁地问用户”，而是“更少、更准地问用户”。

### 1. 开始时问目标，不问实现细节

错误方式：

- 你要用什么数据库？
- 你要不要 OAuth？
- 你要不要插件系统？

更好的方式：

- 这次目标是“能接入真实邮箱”，还是“能自动处理邮件”？
- 完成后你希望用户能少做哪一步？
- 哪些行为必须由人确认？
- 哪些能力这次明确不做？

### 2. 把模糊目标转成 goal card

每个较大的任务都应该先形成一张 goal card：

```text
Objective: 这次要达成什么
User pain: 减少用户哪种困扰
Non-goals: 明确不做什么
Acceptance: 怎么判断完成
Human decisions: 哪些地方必须问人
Evidence: 完成时要跑哪些验证
```

这能减少用户反复解释，也能避免 agent 做着做着偏到别的方向。

### 3. 过程中只在判断点打断用户

应该问用户的问题：

- 这个方向会改变产品定位，是否接受？
- 这个默认行为可能有安全风险，选保守还是激进？
- 这个字段一旦成为公开 JSON contract，后续会有兼容成本，是否稳定？

不应该频繁问用户的问题：

- 文件放哪里
- 测试叫什么
- 普通实现细节用哪个函数名
- 能从代码和项目规则推断出的选择

### 4. 汇报时给证据，不给情绪

每次阶段性汇报应优先回答：

- 改了什么
- 为什么这解决了用户痛点
- 跑了哪些验证
- 还剩什么风险
- 下一步最小可推进切片是什么

## 如何减轻用户困扰

MailCLI 面向的用户并不想学习邮件协议、OAuth 细节、MIME、IMAP flags 或 agent tool schema。他们真正的负担是：

- 不知道怎么把自己的邮箱接进来
- 不知道 AI 会不会误发、误删、误移动邮件
- 不知道哪些邮件重要
- 不知道自动化执行了什么
- 不知道失败后该怎么修

对应的产品策略：

1. **配置负担下沉到 Go CLI。**
   `account add`、`config doctor`、`config test` 和 `config capabilities` 应继续成为主入口。
2. **普通用户只看下一步。**
   命令输出应该直接告诉用户：设置哪个环境变量、跑哪个检查、哪里需要去邮箱后台开启 IMAP。
3. **Agent 读取机器契约。**
   人看简短文本，agent 看 `--format json`，不要让两者抢同一种输出。
4. **危险动作先生成 intent。**
   发送、删除、移动、标记都应先形成可审计意图，再确认执行。
5. **失败可解释。**
   失败结果要有稳定 error code、account、target、operation id 和下一步建议。

## 如何推进项目开发

后续开发不应只按“功能清单”推进，而要按用户工作流推进。

### 当前主线

1. **接入现有邮箱。**
   用户先把已有邮箱接入进来，这是所有 AI 能力的前提。
2. **理解 inbox / thread。**
   MailCLI 先给出结构化候选信号：priority、needs_reply、todo、codes、actions。
3. **提取高价值信息。**
   附件、发票、验证码、链接动作要进入稳定 schema。
4. **执行前确认，执行后记录。**
   自动化从“直接运行命令”升级为“prepare -> confirm -> execute -> log”。

### Harness 化后的开发节奏

每个开发切片都应至少包含：

- 目标和非目标
- 影响的 CLI / schema / docs
- 是否改变公开 JSON contract
- 是否涉及 secret / destructive action / stored user content
- 验收命令
- 完成后的证据

这和普通开发计划的区别是：它不是为了写得好看，而是为了让 agent、维护者和 CI 都能据此执行。

## 如何确定用户目标和期望值

用户通常不会一开始就给出完整需求。Harness 应该把目标澄清变成稳定流程。

### 目标澄清顺序

1. **先确认用户要减少的摩擦。**
   例如“减少邮箱接入难度”比“支持 Gmail”更接近真实目标。
2. **再确认成功状态。**
   例如“用户运行一条命令后看到下一步配置建议”比“体验更好”可验证。
3. **再确认边界。**
   例如“暂不内建 OAuth”或“危险动作必须人工确认”。
4. **最后才确认实现方案。**
   例如 provider preset、operation log、MCP tool exposure。

### 期望值模板

对较大的请求，agent 应该把用户期望整理成：

```text
I think the goal is:
- ...

I will treat these as non-goals:
- ...

I will verify with:
- ...

I will ask you only if:
- ...
```

这能让用户更少承受“管理 agent”的负担。

## 对 MailCLI 的具体产品补充

### 近期应该补

1. **`mailcli account add` 后续增强。**
   增加 provider-specific next steps、环境变量检查、readiness JSON。
2. **`mailcli inbox triage` 或等价命令。**
   输出 priority、needs_reply、todo_candidates、action_candidates，不绑定特定 LLM。
3. **operation intent。**
   为 `send`、`reply`、`delete`、`move`、`mark` 增加 prepare / confirm 契约。
4. **operation log。**
   本地记录执行动作、目标、结果、错误码和时间，禁止记录 raw secret 和完整敏感正文。
5. **MCP capability 分层。**
   默认 MCP 只暴露读取和 setup；危险工具必须在本地配置中显式启用。

### 暂时不应该补

- 托管式 Agent mailbox
- 重 OAuth provider 平台
- 运行时插件加载
- 让 LLM provider 进入 Go core
- 需要用户理解 Python / Node runtime 的官方路径

## 当前最小开发切片

当前第一阶段已经落地：

**`send` 的 operation intent + log。**

原因：

- 它直接解决用户最担心的“AI 会不会乱发、乱删邮件”
- 它会决定未来 MCP 是否能安全暴露写操作
- 它和外部 harness 的核心思想一致：规则自动执行，判断留给人，证据被记录

第一阶段 CLI 契约：

- `mailcli send prepare [--config] [--operations] draft.json`
- `mailcli send confirm [--config] [--operations] <intent-id>`
- `mailcli operations list [--operations]`
- `mailcli operations show [--operations] <operation-id|intent-id>`

验收标准：

- dry-run / prepare 不发送邮件
- confirm 只能执行同一个 intent
- operation log 不记录 secret
- JSON 输出稳定并有命令级测试覆盖
- 文档说明 agent 何时必须请求人确认

后续扩展：

- 给 `reply`、`delete`、`move`、`mark` 增加同样的 prepare / confirm / log 契约
- 先补 inbox triage 信号，再扩大自动执行范围
- MCP 写工具继续默认关闭，除非本地 policy 明确开启
