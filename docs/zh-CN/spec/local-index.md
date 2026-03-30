[English](../../en/spec/local-index.md) | 中文

# 本地索引规范

## 目的

MailCLI 提供一个本地 SQLite 索引，让 Agent 可以在不反复访问远端邮箱的情况下检索最近同步过的邮件。

索引支持：

- 通过 FTS5 进行快速全文检索
- 线程聚合与过滤
- 进程重启后的 watch 去重持久化

## 命令

- `mailcli sync` — 从远端抓取邮件并写入索引
- `mailcli search` — 全文检索本地索引
- `mailcli threads` — 查看会话摘要
- `mailcli thread` — 阅读特定线程的所有邮件
- `mailcli export` — 将索引导出为 JSON、JSONL 或 CSV
- `mailcli watch --index` — 实时流式接收新邮件，附带持久化 seen 状态

## 存储模型

索引是一个 SQLite 数据库文件。

默认路径：

```text
$CACHE_DIR/mailcli/index.db
```

如果无法获取平台 cache 目录，回退到：

```text
.mailcli-index.db
```

### 向后兼容

以 `.json` 结尾的路径会自动映射到 `.db`。现有的 `--index` 参数值无需修改即可继续使用。

```bash
# 以下两条命令打开同一个数据库：
mailcli sync --index /tmp/mailcli.json
mailcli sync --index /tmp/mailcli.db
```

### Schema

维护三张表：

**`messages`** — 每个 `(account, message-id)` 一行：

- `account`, `mailbox`, `id`, `thread_id`, `date`, `indexed_at`
- `from_addr`, `subject`, `category`, `labels`, `action_types`
- `has_codes`, `code_count`, `snippet`, `body_text`, `body_json`

**`messages_fts`** — FTS5 虚拟表：

- 列：`subject`, `from_addr`, `body_text`
- 分词器：`unicode61 remove_diacritics 2`
- 在每次写事务内与 `messages` 保持同步

**`watch_seen`** — 持久化 watch 去重：

- `(account, mailbox, msg_id, seen_at)`
- 被 `mailcli watch` 用于防止进程重启后重复发送事件

### WAL 模式

数据库以 WAL（Write-Ahead Logging）模式运行。写入期间允许并发读取，这对于同时运行 `watch --auto-sync`（写）和搜索（读）的场景非常重要。

## 索引内容

每条索引记录存储：

- 账户名、邮箱名
- driver 侧的消息 ID
- `thread_id`（从 References/In-Reply-To/Message-ID 派生）
- 索引时间戳
- 完整解析后的 `StandardMessage`（以 JSON 形式存储在 `body_json`）
- 预提取的搜索字段（subject、from、snippet、body text）

## 检索语义

### 全文检索（FTS5）

`mailcli search <query>` 使用 FTS5 进行召回：

- 查询中的每个词变成前缀词项
- 所有词项必须同时匹配（隐式 AND）
- FTS5 自动处理 Unicode、变音符号和前缀扩展

FTS5 召回后，结果由 Go 层的 `scoreMatch()` 函数按字段权重重新排名：

| 字段 | 权重 |
|------|------|
| 主题匹配 | 最高 |
| 摘要匹配 | 高 |
| 正文匹配 | 中 |
| 发件人匹配 | 中 |
| 操作/验证码文本 | 中 |

结果先按分数排序，再按消息日期（最新在前）排序。

### 仅过滤检索

不提供查询文本时，`mailcli search` 支持以下过滤：

- `--account`
- `--mailbox`
- `--thread`
- `--since` / `--before`

结果按日期倒序排列。

### 紧凑结果 vs 完整结果

- 默认：紧凑格式 `SearchResult`（account、mailbox、id、thread_id、subject、from、date、snippet、category、labels、score）
- `--full`：包含完整 `StandardMessage` JSON 的 `IndexedMessage`

## 线程语义

`mailcli threads` 将本地索引消息聚合成会话摘要。

线程分组规则（确定性）：

1. 优先取 `references` 中的第一个条目
2. 其次取 `in_reply_to`
3. 其次取 `message_id`
4. 最后取本地索引的消息 ID

线程摘要包含：

- `thread_id`、主题、最新日期、最新发件人、最新预览
- 聚合：categories、action types、labels
- `has_codes`、`code_count`、`action_count`、`participant_count`、参与者列表
- 本地消息 ID 列表、消息数量、分数

`mailcli threads` 过滤：

```bash
mailcli threads --has-codes
mailcli threads --category verification
mailcli threads --action verify_sign_in
```

`mailcli thread <thread_id>` 返回某个线程的完整消息，按消息日期排序。

## 同步语义

`mailcli sync` 默认以增量方式运行：

1. 从配置账户拉取近期消息列表
2. **`BulkHas`**：一次 SQL 查询检查哪些 ID 已存在
3. 仅抓取缺失的消息（或使用 `--refresh` 强制全量）
4. 解析所有抓取到的消息
5. **`BulkUpsert`**：在单个 SQLite 事务中批量写入所有消息

使用单事务批量写入意味着无论批次大小只需一次 `fsync`，大型邮箱的性能远优于逐条提交。

`mailcli sync` 输出：

- `listed_count`、`fetched_count`、`indexed_count`、`skipped_count`
- `refreshed_count`（使用 `--refresh` 时）
- `index_path`

## Watch 状态语义

使用 `--index` 运行 `mailcli watch` 时，`watch_seen` 表会追踪哪些消息已经作为事件发送过。

详见 [Watch 规范](watch.md)。

## 为什么需要这一层

这一层给 MailCLI 提供了实用的 Agent 检索闭环：

1. `mailcli sync` — 一次性填充索引（或通过 `watch --auto-sync` 持续写入）
2. `mailcli search` — 本地多次检索，无需远端请求
3. `mailcli thread` — 按需加载完整会话上下文
4. `mailcli watch` — 实时接收新邮件，附带去重保护
