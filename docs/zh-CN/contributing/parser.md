[English](../../en/contributing/parser.md) | 中文

# Parser 贡献指南

这份文档说明如何为 MailCLI 的共享 parser 做贡献，同时不破坏机器接口契约。

parser 是项目里价值最高的区域之一。

它也是最容易引入“看起来能用、实际上会悄悄回归”的地方之一。

## Parser 的职责边界

parser 接收原始邮件字节，返回 `StandardMessage`。

它负责：

- 解析 MIME 结构和传输编码
- 把不同字符集统一成 UTF-8 文本
- 为 agent 选择最合适的正文表示
- 把噪音 HTML 清洗成适合 Markdown 转换的内容
- 提取 actions、codes、bounce context 等机器可消费结果
- 保守地估算 token 使用量

它不负责：

- 与邮箱 provider 通信
- 写 provider 私有业务策略
- 替某个具体产品决定 inbox workflow 规则
- 吞掉本该属于 driver 的传输错误

## 关键文件

改 parser 行为前，建议先读这些文件：

- `pkg/parser/parser.go`
- `pkg/parser/mime.go`
- `pkg/parser/charset.go`
- `pkg/parser/html.go`
- `pkg/parser/actions.go`
- `pkg/parser/codes.go`
- `pkg/parser/token.go`
- `pkg/parser/parser_test.go`

样本和 golden 数据在：

- `testdata/emails/`
- `testdata/golden/`

保护下游命令输出形状的 CLI 快照测试在：

- `cmd/json_snapshot_test.go`
- `cmd/testdata/snapshots/`

## 推荐工作流

除非有很强的理由，否则建议按这个顺序做：

1. 先在 `pkg/parser/parser_test.go` 写一个聚焦的失败用例。
2. 如果是端到端场景，再在 `testdata/emails/` 增加或调整 fixture。
3. 只有当输出变化是刻意的，才更新 `testdata/golden/` 对应 golden。
4. 先跑定向 parser 测试。
5. 提 PR 前再跑完整测试集。

推荐命令：

```bash
go test ./pkg/parser -run TestParse
go test ./pkg/parser -run TestCleanHTML
go test ./...
```

如果 parser 改动影响了命令 JSON 输出，也要补跑：

```bash
go test ./cmd
```

## Heuristic 边界

parser 可以是 heuristic。

但 parser 不能是“拍脑袋”。

好的 parser heuristic：

- 来自具体 fixture 或失败场景
- 在证据不足时保持保守
- 保留标题、链接、表格、关键图片等有价值结构
- 一定伴随回归测试
- 提升 agent 可读性，但不凭空发明语义

不好的 parser heuristic：

- 为单一 provider 品牌硬编码特殊规则，且无法泛化
- 只因为“像营销内容”就删掉正文
- 对不是明显跳转包装的 URL 做重写
- 仅凭很弱的线索就强行分类 action
- 改了输出结构，却不更新文档和测试

## HTML 清洗规则

在改 `html.go` 时，优先保证这些结果：

- 保留正文主内容
- 去掉明显的 preheader、footer、导航、偏好设置链接等 chrome
- 不要误伤短小的 transactional 正文
- 为 Markdown 转换保留有价值的结构

不要因为容器里出现了 footer-like 词汇，就默认整块内容都能删掉。

优先用评分和窄范围删除，而不是粗暴关键词清洗。

## Action 与 URL 规则

在改 `actions.go` 时：

- 只清洗明显的跳转包装链接
- 保留真实目标 URL
- 保持 label 对 agent 可读
- 以确定性的方式去重
- 不要把普通产品链接轻易判成语义 action

如果一条 URL 清洗规则有可能破坏真实 app 链接，那它通常就太激进了。

## Fixture 建议

优先按行为类型组织 fixture，而不是按品牌组织。

适合的 fixture 类型包括：

- newsletter / 推广
- transactional 状态更新
- 验证 / 登录
- 账单 / 支付
- 退信 / 投递失败
- 安全重置
- 附件入口

仓库里已经有一些可以直接参考的代表性样本：

- `mercury.eml`：大型 HTML newsletter / 推广邮件
- `confirm_subscription.eml`：验证邮箱地址以确认订阅，而不是确认登录的场景
- `verification.eml`：标准验证码 / 验证流程
- `reply_quoted_verification.eml`：验证码内容被引用回复噪音包裹的场景
- `invoice.eml`：账单 / 发票类 transactional 邮件
- `attachment_notice.eml`：主要动作是查看附件或附件入口的邮件
- `bounce.eml` 与 `postfix_bounce.eml`：投递失败和机器生成退信上下文
- `unsubscribe_mixed.eml`：正文和退订动作并存，且退订不能压过主内容的场景

增加 fixture 时：

- 尽量脱敏地址、id 和 token
- 让原始邮件足够真实，保留真正的问题形态
- 用最小但足够保护该行为的样本

## 多语言 Action 样本

MailCLI 现在已经有一批多语言 action 覆盖，尤其包括中文 transactional 与偏好设置类邮件。

可直接参考的多语言样本：

- `unsubscribe_cn.eml`
- `confirm_subscription_cn.eml`
- `invoice_cn.eml`
- `security_reset_cn.eml`
- `security_verify_cn.eml`
- `verification_cn_fullwidth.eml`
- `view_online_abuse_cn.eml`
- `attachment_notice_cn.eml`

扩展多语言覆盖时，建议遵守这些原则：

- 优先补真实 action 模式，不要只补一段翻译文本
- action label 和它所在的正文上下文要一起覆盖
- 如果新关键词可能误伤，至少补一个 negative case
- heuristic 可以语言感知，但仍应保持“模式导向”，不要按发件人品牌硬编码

如果一条新规则只因为某个 provider 固定用了某一句话才成立，那它通常应该更窄，或者暂时不合并。

## Golden 更新规则

golden 文件的作用是保护契约，不是帮你“顺手接受输出变化”。

更新 golden 之前，请先确认：

- 你看过完整 diff，而不只是改动行
- 新输出确实更利于 agent 消费
- token 或结构变化是有意的
- 是否同时影响了 `cmd/json_snapshot_test.go`

不要把无关的 parser 清洗顺手塞进同一次 golden 更新里。小而能解释清楚的 diff 更容易 review，也更安全。

## 推荐验证闭环

parser 相关改动，推荐按这个顺序验证：

```bash
go test ./pkg/parser -run TestParse
go test ./pkg/parser -run TestExtractActions
go test ./pkg/parser -run TestExtractCodes
go test ./cmd
go test ./...
go build -o /tmp/mailcli ./cmd/mailcli
```

如果改动影响 local demo 产物或 thread-facing JSON 输出，再补跑：

```bash
make demo-local-thread-refresh
make demo-local-thread-check
```

第一条命令会重建仓库中提交的 demo 产物。

第二条命令会校验这些产物与仓库预期仍然一致。

## 契约变更

如果你的 parser 改动会影响：

- `StandardMessage`
- action 或 code 的 shape
- 命令 JSON 输出
- 任何已文档化的稳定字段

那么请先开 RFC 风格 issue，而不是只发代码 PR。

参考：

- [RFC 契约变更模板](../../../.github/ISSUE_TEMPLATE/rfc_contract_change.md)
- [Agent Decision 规范](../spec/agent-decisions.md)

## 评审标准

一个 parser 贡献更容易被接受，当它同时满足：

- agent 更容易消费输出
- 回归风险有明确测试覆盖
- 规则能够泛化，而不是只服务某一家 provider
- 输出契约更清晰，而不是更嘈杂

如果你不确定某个行为是否应该放进 parser，优先做更小的改动，并在 PR 中明确写出不确定点。
