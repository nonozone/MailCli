[English](../../en/contributing/drivers.md) | 中文

# 如何添加 Driver

这份文档说明贡献者应该如何为 MailCLI 添加新的 driver。

## 写代码之前

先阅读：

- [Driver 扩展规范](../spec/driver-extension.md)
- [配置规范](../spec/config.md)
- `pkg/driver/driver.go`
- `pkg/driver/factory.go`
- `pkg/driver/stub.go`

如果你的 driver 需要修改以下公开边界，请先开 issue 讨论：

- `Driver`
- `AccountConfig`
- `SearchQuery`
- `MessageMetaSummary`

## 最小实现步骤

1. 在 `pkg/driver/` 下增加新的 driver 实现
2. 让它满足 `Driver` 接口
3. 在 `NewFromAccount` 中接入
4. 为 list、raw fetch、send 行为补测试
5. 文档化必需配置字段和限制

## 设计规则

- 传输逻辑只放在 driver 里
- `FetchRaw` 必须返回原始消息字节
- driver 不要返回“部分解析后的消息内容”
- 不要在 driver 中重复实现 parser 逻辑
- 不要把 provider 私有业务判断硬编码进共享层
- 建议先从 `stub` driver 这种最小实现形态起步，再逐步引入网络传输复杂度

## 测试要求

至少应补充：

- 配置校验失败用例
- list 行为
- fetch 行为
- send 行为，或明确不支持 send 的行为

## Factory 接入

当前 driver 注册点在：

- `pkg/driver/factory.go`

添加新 driver 时，请保持分发逻辑显式、可读。

## 文档要求

如果你增加了一个 driver，请同步更新：

- 面向用户的配置说明
- 相关 README 示例
- driver 自身的限制说明

目标是让另一个贡献者不用读完内部实现，也能知道这个 driver 怎么用、边界在哪里。
