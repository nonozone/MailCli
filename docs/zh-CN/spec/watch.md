[English](../../en/spec/watch.md) | 中文

# Watch 规范

## 目的

`mailcli watch` 将邮箱转变为实时 JSONL 事件流，是邮件系统与 AI Agent 工作流之间的实时数据接口。

Agent 不需要反复轮询 CLI，只需启动一个 `watch` 进程，逐行读取新邮件事件。

## 用法

```bash
mailcli watch [flags]
```

### 参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--config` | — | 配置文件路径 |
| `--account` | — | 账户名称覆盖 |
| `--mailbox` | 账户默认值 | 监控的邮箱，可重复指定（多邮箱） |
| `--poll` | `30s` | 不支持 IMAP IDLE 时的轮询间隔 |
| `--since` | — | 只处理该 RFC3339 时间之后的邮件 |
| `--auto-sync` | `false` | 将新邮件同时写入本地索引（需要 `--index`） |
| `--index` | — | 本地索引路径（启用持久化 seen 状态，`--auto-sync` 必须） |
| `--heartbeat` | `0`（禁用）| 按此间隔发送心跳事件，例如 `5m` |

## 事件结构

stdout 的每一行都是完整的 JSON 对象。

### `watching`

启动时每个邮箱发送一次。

```json
{
  "event": "watching",
  "account": "work",
  "mailbox": "INBOX",
  "ts": "2026-03-31T00:00:00Z"
}
```

### `new_message`

有新邮件到达时发送。包含完整解析后的 `StandardMessage`。

```json
{
  "event": "new_message",
  "account": "work",
  "mailbox": "INBOX",
  "id": "<msg-id@example.com>",
  "message": { "...": "StandardMessage" },
  "ts": "2026-03-31T00:01:00Z"
}
```

### `heartbeat`

按 `--heartbeat` 间隔发送，用于保活检测。

```json
{
  "event": "heartbeat",
  "account": "work",
  "mailbox": "INBOX",
  "ts": "2026-03-31T00:05:00Z"
}
```

### `error`

非致命错误（抓取失败、解析失败、连接错误）时发送。发送后事件流继续。

```json
{
  "event": "error",
  "account": "work",
  "mailbox": "INBOX",
  "id": "<msg-id@example.com>",
  "error": "fetch: dial tcp: connection refused",
  "ts": "2026-03-31T00:01:00Z"
}
```

## 传输策略

MailCLI 根据 driver 类型自动选择最优策略：

| Driver | 策略 |
|--------|------|
| `imap`（服务器支持 IDLE）| **IMAP IDLE**（推送模式） |
| `imap`（不支持 IDLE）| 按 `--poll` 间隔轮询 |
| `dir` | 按 `--poll` 间隔轮询 |
| `stub` | 按 `--poll` 间隔轮询 |

IMAP IDLE 保持持久连接，零延迟接收服务器推送通知，不产生多余流量。

## 持久化 Watch 状态

提供 `--index` 时，`mailcli watch` 在 SQLite 索引数据库中维护一张持久化 **seen set** 表（`watch_seen`）。

```
watch_seen(account, mailbox, msg_id, seen_at)
```

**目的**：即使进程重启，也不会重复发送已处理过的邮件事件。

**行为**：

1. **启动时**：从 SQLite 加载 seen set，表中已有的 ID 会被静默跳过。
2. **首次运行（seen set 为空）**：当前邮箱中的所有邮件 ID 会被写入 seen set，但不发送事件。防止初次启动时大量旧邮件涌入。
3. **新邮件到达**：消息 ID 先写入 `watch_seen`，再发送 `new_message` 事件。保守策略：宁可丢失一次事件通知，也不重复触发 Agent。

**不提供 `--index` 时**：seen set 仅存在于内存中，进程重启后丢失。

## 结合 --auto-sync 使用

同时提供 `--index` 和 `--auto-sync` 时，`mailcli watch` 的处理流程：

1. 收到新消息 ID
2. 抓取原始邮件
3. 解析为 `StandardMessage`
4. 通过 `Upsert` 写入本地 SQLite 索引
5. 记录消息 ID 到 `watch_seen`
6. 向 stdout 发送 `new_message` 事件

这让 Agent 同时拥有实时事件流和可搜索的持久化本地索引，无需再单独运行 `mailcli sync`。

```bash
# 监控 INBOX，将新邮件写入索引，并将事件流传给 Agent：
mailcli watch \
  --account work \
  --index ~/.config/mailcli/index.db \
  --auto-sync \
  | python3 my_agent.py
```

## Agent 集成模式

最简单的集成方式：

```bash
mailcli watch --account work --index ~/.config/mailcli/index.db \
  | while IFS= read -r line; do
      echo "$line" | python3 handle_event.py
    done
```

或直接管道到长运行的 Agent 进程：

```bash
mailcli watch --account work --index ~/.config/mailcli/index.db \
  | python3 tools/agent_example.py
```

Agent 从 stdin 逐行读取，每行是完整 JSON 事件。通过 `event` 字段过滤，只处理 `new_message`。

参考实现见 [tools/agent_example.py](../../../tools/agent_example.py)。

## 多邮箱模式

同时监控多个邮箱：

```bash
mailcli watch \
  --account work \
  --mailbox INBOX \
  --mailbox "Sent" \
  --mailbox "Support"
```

每个邮箱在独立的 goroutine 中运行，所有事件在同一个 stdout 流中混合输出。事件中的 `mailbox` 字段标识来源。

## 优雅关闭

发送 `SIGINT`（Ctrl+C）或 `SIGTERM` 即可干净地停止 watch 进程。关闭期间不会发送不完整的事件。

## 非目标

- `watch` 不发送回复或对邮件采取操作——这是 Agent 的职责。
- `watch` 不保证单进程会话内的精确一次投递。IMAP IDLE 重连时可能重复推送同一 UID。持久化 seen set 保证跨重启不重复，但不保证单次会话内重连窗口期。
