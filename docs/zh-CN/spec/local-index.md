[English](../../en/spec/local-index.md) | 中文

# 本地索引规范

## 目的

MailCLI 提供一个本地文件索引，让 agent 可以在不反复访问远端邮箱的情况下检索最近同步过的邮件。

这是当前本地检索闭环的基础实现。它刻意保持简单，后续可以平滑替换成更强的存储后端。

## 命令

当前命令：

- `mailcli sync`
- `mailcli search`
- `mailcli threads`
- `mailcli thread`

当前搜索模式：

- 默认返回紧凑的搜索结果
- 使用 `mailcli search --full` 返回完整索引消息

当前线程模式：

- 使用 `mailcli threads` 返回会话摘要

## 存储模型

当前实现使用本地磁盘上的 JSON 文件。

默认路径：

```text
$CACHE_DIR/mailcli/index.json
```

如果当前平台无法获取 cache 目录，MailCLI 会回退到：

```text
.mailcli-index.json
```

## 索引内容

当前每条索引记录会保存：

- account 名称
- mailbox 名称
- 面向 driver 的 message id
- indexed timestamp
- 解析后的 `StandardMessage`

这样可以保持检索边界简单清晰：

- 传输仍归 driver
- 结构仍归 `StandardMessage`
- 检索留在本地

## 检索语义

当前本地搜索是大小写不敏感的子串匹配，匹配面包括：

- account
- mailbox
- message id
- subject
- snippet
- body markdown
- category
- labels
- 归一化后的发件人

这还不是全文检索，只是一个确定性的基础版本，优先服务 agent 工作流以及未来存储后端的演进。

当前紧凑搜索结果还会返回一个确定性的 `score` 字段。

当前排序权重大致偏向于：

- subject 命中
- snippet 命中
- body markdown 命中
- 发件人命中

结果会先按 score 排序，再按本地索引时间排序。

`mailcli search` 也支持按以下维度做本地过滤：

- account
- mailbox
- `thread_id`

紧凑版搜索结果会直接暴露 `thread_id`，因此 agent 命中一条消息后，可以直接继续调用 `mailcli thread <thread_id>` 或 `mailcli search --thread <thread_id> ...`，无需在本地再次推导线程归属。

当使用 `--full` 时，命令会返回完整的索引记录，而不是紧凑摘要。完整索引记录同样会暴露 `thread_id`。

## 线程语义

`mailcli threads` 会把本地索引消息聚合成会话摘要。

当前线程归并规则是确定性的，优先顺序为：

- `references` 中的第一条可用引用
- 否则 `in_reply_to`
- 否则 `message_id`
- 再否则回退到本地索引消息 id

当前线程摘要会暴露：

- `thread_id`
- subject
- latest date
- latest message id
- latest message sender
- latest message preview
- aggregated categories
- aggregated action types
- aggregated labels
- `has_codes`
- message count
- participants
- 本地 message ids
- score

`mailcli thread <thread_id>` 会返回选中本地线程中的完整索引消息，并按消息时间排序。这些完整记录同样会暴露 `thread_id`。

`mailcli threads` 当前支持以下确定性的 thread 级过滤：

- `--has-codes`
- `--category <value>`
- `--action <type>`

## 同步语义

当前 `mailcli sync` 默认采用增量策略：

- 先从当前账户列出近期邮件
- 对于本地索引中已经存在相同 `(account, id)` 的消息直接跳过
- 只抓取、解析并 upsert 缺失的消息

如果你希望重新抓取并覆盖已有记录，可以使用 `--refresh`。

## 当前阶段明确不做的事

- 暂不要求 SQLite
- 暂不要求 FTS
- 暂不支持远端增量变更跟踪
- 暂不处理复杂的 partial-sync 冲突

## 为什么先做这个

这一层让 MailCLI 立刻具备了可用的 agent 检索闭环：

1. 先同步一次近期邮件
2. 再在本地进行多次搜索
3. 只有在需要时才重新抓取完整邮件
