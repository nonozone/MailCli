[English](../../en/examples/local-thread-demo.md) | 中文

# 本地 Thread Demo

这个页面展示一条完整的本地 thread agent 往返链路，不需要先去读 Python 示例源码。

它使用：

- `examples/config/fixtures-dir.yaml`
- 内置 `dir` driver
- 仓库里的 `testdata/emails` fixture 语料

目标很直接：

1. 同步本地状态
2. 对候选 thread 排序
3. 读取选中的 thread
4. 让 agent 选择回复动作
5. 把 `ReplyDraft` 编译成 MIME

## 为什么需要这个 Demo

常规示例页面主要告诉你命令怎么跑。

这个页面则直接解释整条链路里，机器侧输入输出到底长什么样。

如果 fixture corpus 发生变化，可以用下面这条命令重新生成这组固定 demo 产物：

```bash
python3 examples/python/refresh_local_thread_demo.py \
  --mailcli-bin ./mailcli \
  --config examples/config/fixtures-dir.yaml \
  --account fixtures \
  --index /tmp/mailcli-fixtures-index.json \
  --output-dir examples/artifacts/local-thread-demo
```

## 命令序列

```bash
mailcli sync --config examples/config/fixtures-dir.yaml --account fixtures --index /tmp/mailcli-fixtures-index.json --limit 20
mailcli threads --index /tmp/mailcli-fixtures-index.json invoice
mailcli thread --index /tmp/mailcli-fixtures-index.json "<invoice-123@example.com>"
mailcli reply --config examples/config/fixtures-dir.yaml --account fixtures --dry-run examples/artifacts/local-thread-demo/reply.draft.json
```

## 已保存的样例资产

- [sync.json](../../../examples/artifacts/local-thread-demo/sync.json)
- [threads.json](../../../examples/artifacts/local-thread-demo/threads.json)
- [thread.json](../../../examples/artifacts/local-thread-demo/thread.json)
- [reply.draft.json](../../../examples/artifacts/local-thread-demo/reply.draft.json)
- [reply.mime.txt](../../../examples/artifacts/local-thread-demo/reply.mime.txt)
- [agent-report.json](../../../examples/artifacts/local-thread-demo/agent-report.json)

## 第一步：Sync

第一份机器输出是 sync 结果：

```json
{
  "account": "fixtures",
  "mailbox": "INBOX",
  "listed_count": 13,
  "fetched_count": 13,
  "indexed_count": 13,
  "skipped_count": 0,
  "refreshed_count": 0,
  "index_path": "/tmp/mailcli-fixtures-index.json"
}
```

agent 在这里会知道：

- 当前索引的是哪个账户和 mailbox
- 本地状态是不是足够新
- 是否需要再做一次 refresh

## 第二步：Thread 排序

接着 agent 看紧凑 thread 摘要：

```json
{
  "thread_id": "<invoice-123@example.com>",
  "subject": "Your April invoice is ready",
  "last_message_id": "invoice.eml",
  "action_types": ["pay_invoice", "view_invoice"],
  "score": 85
}
```

这一步的价值在于：

- agent 不用先加载整封正文，就能挑出相关会话
- `action_types` 和 `score` 减少了 prompt 里的分拣工作

## 第三步：读取 Thread

然后 agent 读取选中的 thread：

```json
{
  "id": "invoice.eml",
  "thread_id": "<invoice-123@example.com>",
  "message": {
    "meta": {
      "from": { "name": "Billing Team", "address": "billing@example.com" },
      "subject": "Your April invoice is ready",
      "message_id": "<invoice-123@example.com>"
    },
    "content": {
      "body_md": "## Your invoice is ready\n\nInvoice #123 is now available in your billing portal.\n\n[View invoice](https://billing.example.com/invoices/123)\n\n[Pay invoice](https://billing.example.com/pay/123)"
    },
    "actions": [
      { "type": "view_invoice", "url": "https://billing.example.com/invoices/123" },
      { "type": "pay_invoice", "url": "https://billing.example.com/pay/123" }
    ]
  }
}
```

agent 在这里会拿到：

- 发件人是谁
- 归一化后的正文长什么样
- 可以执行哪些回复或后续动作

## 第四步：Agent 决策与 Reply Draft

到这一步，agent 可以产出一个稳定的 `ReplyDraft` 边界：

```json
{
  "from": { "address": "support@nono.im" },
  "to": [{ "address": "billing@example.com", "name": "Billing Team" }],
  "body_text": "Thanks, we have received the invoice notification.",
  "account": "fixtures",
  "reply_to_id": "invoice.eml"
}
```

`reply_to_id` 的价值是：

- agent 不必自己重建 `In-Reply-To`
- agent 不必自己重建 `References`
- MailCLI 可以重新抓原始邮件，自己推导线程头

## 第五步：编译后的 MIME

`mailcli reply --dry-run` 会把这个 draft 编译成可发送的 MIME：

```text
From: support@nono.im
To: Billing Team <billing@example.com>
Subject: Re: Your April invoice is ready
Message-ID: <generated@mailcli.local>
In-Reply-To: <invoice-123@example.com>
References: <invoice-123@example.com>
Date: Fri, 27 Mar 2026 15:11:13 +0000
MIME-Version: 1.0
Content-Type: text/plain; charset=UTF-8

Thanks, we have received the invoice notification.
```

`Message-ID` 和 `Date` 在运行时会动态生成。

稳定点在于：agent 自始至终都不需要自己手写 raw MIME。

## 完整报告

如果你想一次看完整条链路，可以直接看：

- [agent-report.json](../../../examples/artifacts/local-thread-demo/agent-report.json)

这个文件对应的就是 `examples/python/agent_thread_assistant.py` 在这条本地 fixture 流程上的完整输出。

## 相关文档

- [Agent Thread 示例](agent-thread-assistant.md)
- [Examples 索引](README.md)
- [Agent 协作流程](../agent-workflows.md)
