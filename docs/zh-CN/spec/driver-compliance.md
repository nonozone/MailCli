[English](../../en/spec/driver-compliance.md) | 中文

# Driver 合规清单

这份文档定义了 MailCLI driver 的最低验收标准。

它刻意比“完整 provider 正确性”更窄。

目标是让新 driver 的 review 不再依赖维护者脑内知识。

## 契约边界

一个 driver 应当遵守的共享边界包括：

- `pkg/driver.Driver`
- `internal/config.AccountConfig`
- `schema.SearchQuery`
- `schema.MessageMetaSummary`

如果某个新 driver 需要改动这条边界，请先开 RFC issue。

## 必备行为

每个 driver 都应该满足这些基线行为：

1. `List` 返回稳定、可抓取的消息 id。
2. `FetchRaw` 返回 RFC 822 风格的原始字节，而不是部分解析内容。
3. 在可行时，缺失消息的 `FetchRaw` 应返回 `ErrMessageNotFound`。
4. `SendRaw` 要么成功，要么返回稳定的操作级错误。
5. driver 代码应把传输细节限制在 driver 层内部。

## 必备错误形状

在适用时，优先使用这些稳定错误：

- `ErrMessageNotFound`
- `ErrTransportNotConfigured`
- `ErrDriverConfigInvalid`
- `ErrAuthFailed`

provider 私有错误仍然可以继续冒泡，但只要能改善调用方行为，就应优先使用稳定 typed error。

## 必备测试

一个新的 driver 至少应包含：

- 配置校验覆盖
- list 行为覆盖
- fetch 行为覆盖
- send 行为覆盖，或明确不支持 send 的覆盖
- `pkg/driver/conformance_test.go` 这套共享合同测试

共享合同测试只是最低门槛，不是完整测试计划。

## 建议补充的测试

还应针对 provider 私有边界补这些用例：

- mailbox 选择语义
- 支持的消息标识形式
- auth 失败映射
- 传输配置缺失
- 分页或最近消息假设
- send 时的 envelope 提取边界

## Review 时建议追问的问题

在 review driver PR 时，建议重点看：

- 另一个贡献者能否看懂消息 id 该怎么用？
- `FetchRaw` 是否真的返回原始字节？
- 缺失消息和认证失败是否清晰映射？
- provider 私有解析逻辑有没有泄漏到共享层？
- 配置要求和限制是否写清楚了？

## 参考

- [Driver 扩展规范](driver-extension.md)
- [如何添加 Driver](../contributing/drivers.md)
- `pkg/driver/conformance_test.go`
