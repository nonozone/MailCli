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

## 稳定边界

当前相对稳定的 driver 边界包括：

- `pkg/driver.Driver`
- `internal/config.AccountConfig`
- `schema.SearchQuery`
- `schema.MessageMetaSummary`

这些边界在 `v1.0` 前仍可能演进，但修改前应先讨论，因为它会影响所有 provider 实现。

## 错误处理建议

Driver 应优先返回：

- `ErrMessageNotFound`
- `ErrTransportNotConfigured`
- `ErrDriverConfigInvalid`

其他 provider 私有错误仍然可以继续冒泡，但如果能提升上层行为，优先补充稳定的 typed error。

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
