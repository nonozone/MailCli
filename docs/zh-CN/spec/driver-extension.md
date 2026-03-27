[English](../../en/spec/driver-extension.md) | 中文

# Driver 扩展规范

## 目的

MailCLI 把传输能力收敛在 `driver` 包里。

Driver 负责与邮箱协议或 provider API 通信，但不负责 MIME 清洗、语义提取或出站草稿编译。

## 核心接口

当前接口：

```go
type Driver interface {
    List(ctx context.Context, query schema.SearchQuery) ([]schema.MessageMetaSummary, error)
    FetchRaw(ctx context.Context, id string) ([]byte, error)
    SendRaw(ctx context.Context, raw []byte) error
}
```

## 职责

Driver 可以负责：

- 列出邮箱中的消息
- 获取原始 RFC 822 风格字节流
- 发送原始出站 MIME 字节
- 在合适时把 provider 传输错误映射成稳定的操作级错误

Driver 不应该负责：

- 将邮件正文解析成 `StandardMessage`
- 清洗 HTML
- 提取 actions 或 codes
- 从 `DraftMessage` 或 `ReplyDraft` 生成 MIME
- 把 provider 私有业务语义塞进 parser 层

## 当前扩展模型

在 `v0.1 RC` 阶段，扩展模型是 **仓库内代码级扩展**。

也就是说，贡献者添加一个新 driver 的方式是：

1. 实现 `Driver` 接口
2. 接入 driver factory
3. 增加测试
4. 补充账户配置和限制说明

当前契约不包含运行时动态插件。

## 参考实现

当前仓库内已经有两种不同形态的 driver 实现：

- `pkg/driver/imap.go`：真实网络传输，实现 IMAP 读取和 SMTP 发送
- `pkg/driver/stub.go`：确定性的本地 driver，用于开发和扩展示例

如果你要贡献新的 driver，建议先读 `stub.go` 理解最小实现，再对照 `imap.go` 了解有状态的网络型实现该如何组织。

## 稳定边界

当前相对稳定的 driver 边界包括：

- `pkg/driver.Driver`
- `internal/config.AccountConfig`
- `schema.SearchQuery`
- `schema.MessageMetaSummary`

这些边界在 `v1.0` 前仍可能演进，但修改前应先讨论，因为它会影响所有 provider 实现。

如果提案会修改这条边界，请优先使用 `RFC contract change` issue 模板，而不是普通 feature request。

## 错误处理建议

Driver 应优先返回：

- `ErrMessageNotFound`
- `ErrTransportNotConfigured`
- `ErrDriverConfigInvalid`
- `ErrAuthFailed`

其他 provider 私有错误仍然可以继续冒泡，但如果能提升上层行为，优先补充稳定的 typed error。

## 合规测试建议

MailCLI 现在在下面这个文件里提供了一套共享 driver 合同测试：

- `pkg/driver/conformance_test.go`

新增 driver 时，建议先用这套测试证明每个 driver 都应保持的最低行为：

- `List` 返回的消息 id 应该可用于后续抓取
- `FetchRaw` 对已列出的 id 必须成功
- 缺失消息应返回 `ErrMessageNotFound`
- `SendRaw` 要么成功，要么返回稳定的操作级错误

这套测试刻意保持小而稳定。

它不能替代 driver 自己的边界测试。

但它为贡献者和维护者提供了一条共享的最低验收线，让 provider 私有边界问题放到第二层再看。

## 消息标识建议

Driver 应明确说明自己接受哪些消息标识形式。

例如当前 IMAP driver 支持：

- sequence number
- 显式 UID
- `Message-ID` 查找

允许存在 provider 私有的便捷形式，但必须文档化，且不能悄悄改变含义。

## 配置建议

每个 driver 都应明确：

- 必填账户字段
- 可选账户字段
- 传输假设
- 发信限制

如果某个 driver 需要特殊凭据或 provider 私有设置，应写进 driver 贡献文档，而不是污染无关的 parser 或 CLI 代码。
