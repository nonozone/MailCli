[English](README.md) | 中文

# MailCLI

**AI Native 的邮件接口：把混乱的 MIME 转换成适合 agent 使用的结构化上下文。**

MailCLI 是一个面向 **AI agent**、**LLM 工作流** 和 **自动化开发者** 的开源命令行邮件工具。

传统邮件工具主要是为人类阅读收件箱而设计的，而 MailCLI 的目标是让系统能够通过稳定的机器接口去 **读取**、**理解**、**回复** 和 **发送** 邮件。

它不会把原始 MIME、臃肿 HTML 和 provider 私有行为直接推给 prompt，而是把邮件转换成结构化 JSON、干净 Markdown 和可组合的工作流。

## 核心愿景

在 AI 时代，邮件不应该再只是 HTML 模板和传输头的堆叠。

它应该像 API 资源一样被结构化消费。

MailCLI 的存在，就是为了让 agent 处理邮件像读取一份 JSON 文档一样自然。

## 核心特性

- **AI-First Parser**
  将嘈杂的原始邮件转换为适合 agent 推理的标准化 JSON 与 Markdown。
- **协议与内容解耦**
  Driver 负责传输，Parser 负责内容，Composer 负责出站 MIME。
- **动作提取**
  自动提取退订链接、安全入口、账单/付款入口、附件入口、退信/错误上下文和线程相关元数据。
- **开发者友好的 CLI**
  支持 `json`、`yaml`、`table` 输出格式，支持 stdin 管道。
- **双向工作流**
  通过 `list/get/parse` 读邮件，通过 `DraftMessage` 和 `ReplyDraft` 驱动发信与回复。
- **Provider 无关架构**
  核心模型不绑定单一后端，后续可支持 IMAP、SMTP、API 以及其他生态集成。

## 为什么需要 MailCLI

原始邮件不是一个适合 agent 的接口：

- MIME 树太嘈杂
- HTML 模板太耗 token
- 回复线程很容易断
- 各家 provider 的 API 差异太大

MailCLI 提供的是一个稳定边界：

- 入站邮件统一变成 `StandardMessage`
- 出站意图统一变成 `DraftMessage` 或 `ReplyDraft`
- 传输细节隐藏在 driver 后面

## 当前能力

### 读路径

- `mailcli parse --format json|yaml|table <file|->`
- `mailcli list --config ~/.config/mailcli/config.yaml [--account <name>] [--format json|table]`
- `mailcli get --config ~/.config/mailcli/config.yaml [--account <name>] <id>`

### 写路径

- `mailcli send --dry-run <draft.json>`
- `mailcli send --config ~/.config/mailcli/config.yaml <draft.json>`
- `mailcli reply --dry-run <reply.json>`
- `mailcli reply --config ~/.config/mailcli/config.yaml <reply.json>`

### 回复能力

- 支持 `reply_to_message_id`
- 支持 `reply_to_id`
- 当使用 `reply_to_id` 时，MailCLI 可以先抓取原邮件，再自动推导：
  - `In-Reply-To`
  - `References`
  - 默认回复主题

## 架构设计

MailCLI 采用分层架构，方便贡献者在明确边界内工作：

1. **Driver Layer**
   获取原始邮件、发送原始字节。
2. **Parser Engine**
   负责 MIME 解码、字符集归一化、HTML 清洗、Markdown 转换和动作提取。
3. **Composer**
   负责把 `DraftMessage` 和 `ReplyDraft` 编译成标准出站 MIME。
4. **CLI Core**
   负责账户选择、命令编排、输出格式化和流程调度。

核心原则：

**协议归 driver，内容归 parser，编译归 composer，流程调度归 CLI core**

## Agent 协作模型

MailCLI 不只是一个 parser，它是 agent 和邮件系统之间的桥梁。

### 读循环

```text
Agent -> mailcli list/get/parse -> Driver -> Raw Email -> Parser -> StandardMessage -> Agent
```

### 回复循环

```text
Agent -> ReplyDraft -> mailcli reply -> Composer -> Raw MIME -> Driver -> Provider
```

### 新邮件发送循环

```text
Agent -> DraftMessage -> mailcli send -> Composer -> Raw MIME -> Driver -> Provider
```

详细说明见：

- [Agent 协作流程](docs/zh-CN/agent-workflows.md)
- [发送侧消息规范](docs/zh-CN/spec/outbound-message.md)

## 最小配置示例

```yaml
current_account: work
accounts:
  - name: work
    driver: imap
    host: imap.example.com
    port: 993
    username: you@example.com
    password: app-password-or-token
    tls: true
    mailbox: INBOX
    smtp_host: smtp.example.com
    smtp_port: 587
    smtp_username: you@example.com
    smtp_password: app-password-or-token
```

## 快速开始

### 解析本地邮件

```bash
cat test.eml | mailcli parse --format json -
```

### 从已配置账户列出邮件

```bash
mailcli list --config ~/.config/mailcli/config.yaml --format table
```

### 按 id 抓取并解析邮件

```bash
mailcli get --config ~/.config/mailcli/config.yaml "<message-id>"
```

### Dry-run 新邮件

```bash
mailcli send --dry-run draft.json
```

### Dry-run 回复

```bash
mailcli reply --dry-run reply.json
```

## 路线图

### Phase 1: The Brain

- [x] 定义 `StandardMessage`、`DraftMessage`、`ReplyDraft` 和 `SendResult`
- [x] 构建 parser-first MVP，并建立代表性 golden tests
- [x] 建立 `parse`、`list`、`get`、`send`、`reply` 的命令骨架
- [x] 增加本地 MIME composer
- [ ] 强化 HTML 降噪、主体区域提取和 URL 清洗
- [ ] 扩大样本集，覆盖 newsletter、交易邮件、告警和边界场景

### Phase 2: The Hands

- [x] 增加基于配置文件的账户上下文
- [x] 增加基础 IMAP 读路径
- [x] 为 IMAP 风格账户增加 SMTP 发信路径
- [ ] 完成 IMAP `FetchRaw`
- [ ] 将传输错误进一步映射成稳定的结果码

### Phase 3: The Memory

- [ ] 增加本地索引和缓存
- [ ] 增加搜索与线程导航
- [ ] 增加更适合本地 agent 工作流的结构化元数据

### Phase 4: The Ecosystem

- [ ] 支持更多 provider
- [ ] 增加更完整的生态集成与 driver 扩展文档
- [ ] 稳定 RFC 驱动的扩展点

## 示例

- Python 解析示例：`examples/python/parse_email.py`
- Python 回复 dry-run 示例：`examples/python/reply_dry_run.py`
- Shell 解析示例：`examples/shell/parse_email.sh`
- Shell 回复 dry-run 示例：`examples/shell/reply_dry_run.sh`

## 如何贡献

MailCLI 还处在早期阶段，但方向已经明确。

我们重点欢迎这些贡献：

1. **Parser 质量**
   更好的 MIME 处理、HTML 清洗、字符集处理和 Markdown 质量。
2. **语义契约**
   更清晰的 agent-facing 邮件标准。
3. **Driver**
   更多 provider、更稳的传输行为、更好的兼容层。
4. **Agent 工具链**
   更好的 examples、prompt 模式和工作流集成。

重大改动建议先讨论。

项目对社区开放，但方向上会持续围绕以下目标收敛：

- AI-native workflows
- 清晰的关注点分离
- 稳定的机器接口

## 许可证

Apache-2.0

## 相关文档

- [English documentation](README.md)
- [Agent 协作流程](docs/zh-CN/agent-workflows.md)
- [发送侧消息规范](docs/zh-CN/spec/outbound-message.md)
