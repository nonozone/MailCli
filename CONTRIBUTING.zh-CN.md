[English](CONTRIBUTING.md) | 中文

# 参与贡献 MailCLI

MailCLI 是一个开源的 AI Native 邮件接口项目。

它的目标不是成为传统终端邮件客户端，而是为邮件驱动的 agent 工作流建立稳定的机器接口。

## 开始之前

- 阅读 [README.zh-CN.md](README.zh-CN.md)
- 阅读 [Agent 协作流程](docs/zh-CN/agent-workflows.md)
- 如果你想加 provider，请先阅读 [Driver 扩展规范](docs/zh-CN/spec/driver-extension.md)
- 修改公开契约前，先阅读对应 spec
- 涉及重大行为、schema 或架构调整时，建议先开 issue 讨论

## 我们欢迎的贡献方向

- parser 质量和样本覆盖
- provider / driver 兼容性
- 出站 MIME 正确性
- agent workflow 示例
- 文档与贡献体验

## 贡献规则

- 保持核心 provider 无关
- 不要把 provider 私有的业务逻辑直接塞进 parser，除非它足够通用
- 协议逻辑放在 driver，内容逻辑放在 parser，MIME 生成放在 composer，流程编排放在 `cmd`
- 契约变更必须同时更新文档和测试
- 优先提交范围清晰的小型 PR

## 基础开发命令

```bash
go test ./...
go build ./cmd/mailcli
python3 -m py_compile examples/python/*.py examples/providers/*.py
```

## Pull Request 请包含

- 改了什么
- 为什么改
- 如何验证
- 如果契约或工作流变了，文档是否同步更新

## 对公开契约敏感的变更

修改以下内容前，请先开 issue 讨论：

- `StandardMessage`
- `DraftMessage`
- `ReplyDraft`
- `SendResult`
- 命令输出结构
- driver 扩展边界

如果你要贡献 driver，也建议先看：

- [Parser 贡献指南](docs/zh-CN/contributing/parser.md)，如果你要改共享 parser 行为
- [如何添加 Driver](docs/zh-CN/contributing/drivers.md)

## 语言与文档

- 主文档采用中英文分文件维护
- 优先用互相跳转链接，而不是中英对照混排
- 如果更新用户可见文档，原则上同步更新两种语言版本，除非内容本身只适用于某一种语言环境

## 评审标准

一个改动更容易被接受，当它能提升以下至少一项，同时不明显破坏其他项：

- AI workflow 可用性
- 关注点分离
- 机器接口稳定性
- 贡献者理解成本
