# MailCLI v0.1.0-rc1

这是 MailCLI 作为开放式 AI Native 邮件接口的第一个 release candidate。

## 亮点

- 将原始邮件解析成 `StandardMessage` JSON 和 Markdown
- 通过基础 IMAP driver 提供 `list` / `get` 收件箱读取能力
- 通过面向 IMAP 风格账户的 SMTP 路径发送新邮件和回复
- 支持 `reply_to_message_id` 和 `reply_to_id` 回复流程
- 出站 MIME 已支持 `multipart/alternative` 和附件打包
- 为 agent 暴露稳定的机器接口：
  - `StandardMessage`
  - `DraftMessage`
  - `ReplyDraft`
  - `SendResult`
- 提供 Python / shell 示例，以及 external provider 契约和可选 OpenAI provider 示例

## 这个 RC 已包含

- `mailcli parse`
- `mailcli list`
- `mailcli get`
- `mailcli send`
- `mailcli reply`
- 基于序号、UID 和 `Message-ID` 的 IMAP 原始邮件抓取
- 稳定的出站结果码，例如 `auth_failed`、`message_not_found`、`transport_not_configured`
- 面向密码字段的环境变量秘密注入
- driver 扩展说明与贡献文档
- 已扩充的 parser 样本集，覆盖：
  - 纯文本邮件
  - 推广 / newsletter 邮件
  - 退信邮件
  - 验证码邮件
  - 账单 / 支付邮件
  - 安全重置邮件
  - 附件入口邮件

## 仍在演进中的部分

- HTML 清洗与 URL 归一化 heuristic
- action 提取覆盖率
- 常见布局之外的验证码提取
- 更丰富的搜索与本地索引
- 基础 IMAP/SMTP 之外的更多 provider

## 建议视为稳定的集成边界

在这个 RC 阶段，建议将以下部分视为稳定边界：

- `mailcli parse`
- `mailcli list`
- `mailcli get`
- `mailcli send`
- `mailcli reply`
- `StandardMessage`
- `DraftMessage`
- `ReplyDraft`
- `SendResult`

## 验证

- `go test ./...`
- `go build ./cmd/mailcli`
- `python3 -m py_compile examples/python/*.py examples/providers/*.py`

## 文档

- 发布说明：`docs/zh-CN/release/v0.1-rc.md`
- Agent 协作流程：`docs/zh-CN/agent-workflows.md`
- 发送侧规范：`docs/zh-CN/spec/outbound-message.md`
- Driver 扩展规范：`docs/zh-CN/spec/driver-extension.md`
