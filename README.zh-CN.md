[English](README.md) | 中文

# MailCLI

MailCLI 是一个独立开源的工具集，将原始邮件转换为 AI 可消费的结构化上下文，而不依赖任何特定提供商的接口。

## 项目定位

- 面向 AI agent 和自动化构建者，统一输出标准化的 JSON、Markdown、动作与线程上下文。
- Parser-first 路线优先验证数据质量，而不是重建面向终端的客户端。
- 保持 provider 无关，方便社区后续补充 IMAP、API 或 OhRelay 驱动。

## 非目标

- 不是传统终端邮件客户端或 UI 替代品。
- 不在仓库里绑定具体业务集成。
- 初始阶段不追求每个 provider 私有字段都塞进 schema。
- 当前版本不涵盖发送、规则引擎或完整 UI。

## Parser-first MVP

- `mailcli parse <file>` 与 `cat <file> | mailcli parse -` 接受原始 `.eml`。
- 输出包含标准化头部、Markdown 正文、动作（如退订）、线程元数据和 token 估算。
- 使用代表性 Mercury、bounce、plaintext 样本的 golden tests 确保解析行为一致。
- CLI 保持精简：目前只有 parse 命令，后续再接入格式化与账户上下文。

## 当前命令

- `mailcli parse --format json|yaml|table <file|->`
- `mailcli list --config ~/.config/mailcli/config.yaml [--account <name>]`

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
```

## 架构概要

- `Cmd` 层管理 Cobra 命令、stdin/文件输入和输出格式。
- `Driver` 接口负责传输（获取/发送），不涉足内容解析。
- `Parser` 承担 MIME 遍历、字符集归一、HTML 清洗、Markdown 转换、动作提取与 schema 输出。
- 后续发送/回复链路会引入 `DraftMessage`、`ReplyDraft` 这类意图级 schema，由 `mailcli` 负责把它们编译成带线程头的真实 MIME 邮件。

## 链接

- [English documentation](README.md)
- [发送侧消息规范](docs/zh-CN/spec/outbound-message.md)
