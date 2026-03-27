import argparse
import json
import subprocess
import sys
from typing import Any

from provider_contract import parse_external_analysis


def main() -> int:
    args = parse_args()

    sync_result = None
    if not args.skip_sync:
        sync_result = run_sync(args)

    thread_summaries = load_thread_summaries(args)
    selection = select_thread(thread_summaries, args)
    thread_messages = load_thread_messages(selection["thread_id"], args, args.thread_message_limit)
    if not thread_messages:
        raise SystemExit("selected thread did not return any local messages")

    thread_messages, latest_message = ensure_latest_message(selection, thread_messages, args)
    analysis = analyze_with_provider(selection, thread_summaries, thread_messages, latest_message, args)

    report: dict[str, Any] = {
        "tool": "mailcli-thread-agent-example",
        "source": build_source(args),
        "selection": selection,
        "thread_summaries": thread_summaries,
        "thread_messages": thread_messages,
        "latest_message": latest_message,
        "analysis": analysis,
    }
    if sync_result is not None:
        report["sync"] = sync_result

    reply_text = args.reply_text or analysis.get("reply_text")
    if reply_text:
        if not args.from_address:
            print("--from-address is required when --reply-text is used", file=sys.stderr)
            return 2

        draft = build_reply_draft(selection, latest_message, args)
        draft["body_text"] = reply_text
        mime = compile_reply_dry_run(draft, args)
        report["reply"] = {
            "mode": "dry_run",
            "draft": draft,
            "mime": mime,
        }
        report["analysis"]["decision"] = "draft_reply"

    json.dump(report, sys.stdout, indent=2, ensure_ascii=False)
    sys.stdout.write("\n")
    return 0


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Thread-aware MailCLI agent workflow example.",
    )
    parser.add_argument("--mailcli-bin", default="mailcli", help="path to the mailcli binary")
    parser.add_argument("--config", help="mailcli config path")
    parser.add_argument("--account", help="mailcli account override")
    parser.add_argument("--mailbox", help="mailcli mailbox override")
    parser.add_argument("--index", required=True, help="local index path")
    parser.add_argument("--query", help="thread query used for selection")
    parser.add_argument("--thread-id", help="explicit thread id override")
    parser.add_argument("--sync-limit", type=int, default=10, help="maximum messages to sync before thread selection")
    parser.add_argument("--thread-limit", type=int, default=10, help="maximum thread summaries to load")
    parser.add_argument("--thread-message-limit", type=int, default=50, help="maximum local thread messages to load")
    parser.add_argument("--skip-sync", action="store_true", help="skip mailcli sync and use the existing local index")
    parser.add_argument("--reply-text", help="optional reply body text to compile with mailcli reply --dry-run")
    parser.add_argument("--from-address", help="from address to use for reply dry-run")
    parser.add_argument("--from-name", help="optional from display name for reply dry-run")
    parser.add_argument("--agent-provider", choices=["builtin", "external"], default="builtin", help="analysis provider")
    parser.add_argument("--provider-command", help="external provider command")
    parser.add_argument("--provider-arg", action="append", default=[], help="repeatable argument for the external provider command")

    args = parser.parse_args()
    if not args.skip_sync and not args.config:
        parser.error("--config is required unless --skip-sync is used")
    if args.agent_provider == "external" and not args.provider_command:
        parser.error("--provider-command is required when --agent-provider external is used")
    return args


def build_source(args: argparse.Namespace) -> dict[str, Any]:
    return {
        "mode": "local_thread",
        "config": args.config,
        "account": args.account,
        "mailbox": args.mailbox,
        "index": args.index,
        "query": args.query,
        "thread_id": args.thread_id,
        "skip_sync": args.skip_sync,
    }


def run_sync(args: argparse.Namespace) -> dict[str, Any]:
    command = [
        args.mailcli_bin,
        "sync",
        "--format",
        "json",
        "--config",
        args.config,
        "--index",
        args.index,
        "--limit",
        str(args.sync_limit),
    ]
    if args.account:
        command.extend(["--account", args.account])
    if args.mailbox:
        command.extend(["--mailbox", args.mailbox])
    return run_mailcli_json(command)


def load_thread_summaries(args: argparse.Namespace) -> list[dict[str, Any]]:
    command = [
        args.mailcli_bin,
        "threads",
        "--format",
        "json",
        "--index",
        args.index,
        "--limit",
        str(args.thread_limit),
    ]
    if args.account:
        command.extend(["--account", args.account])
    if args.mailbox:
        command.extend(["--mailbox", args.mailbox])
    if args.query:
        command.append(args.query)

    output = run_mailcli_json(command)
    if not isinstance(output, list):
        raise SystemExit("mailcli threads must return a JSON array")
    return output


def select_thread(thread_summaries: list[dict[str, Any]], args: argparse.Namespace) -> dict[str, Any]:
    explicit_thread_id = (args.thread_id or "").strip()
    if explicit_thread_id:
        for item in thread_summaries:
            if item.get("thread_id") == explicit_thread_id:
                result = dict(item)
                result["selection_strategy"] = "explicit_thread_id"
                return result
        return {
            "thread_id": explicit_thread_id,
            "selection_strategy": "explicit_thread_id",
        }

    if not thread_summaries:
        raise SystemExit("no local thread matched the current query")

    result = dict(thread_summaries[0])
    result["selection_strategy"] = "top_thread"
    return result


def load_thread_messages(thread_id: str, args: argparse.Namespace, limit: int | None) -> list[dict[str, Any]]:
    command = [
        args.mailcli_bin,
        "thread",
        "--format",
        "json",
        "--index",
        args.index,
    ]
    if limit is not None:
        command.extend(["--limit", str(limit)])
    if args.account:
        command.extend(["--account", args.account])
    if args.mailbox:
        command.extend(["--mailbox", args.mailbox])
    command.append(thread_id)

    output = run_mailcli_json(command)
    if not isinstance(output, list):
        raise SystemExit("mailcli thread must return a JSON array")
    return output


def ensure_latest_message(
    selection: dict[str, Any],
    thread_messages: list[dict[str, Any]],
    args: argparse.Namespace,
) -> tuple[list[dict[str, Any]], dict[str, Any]]:
    expected_latest_id = (selection.get("last_message_id") or "").strip()
    if not expected_latest_id:
        return thread_messages, thread_messages[-1]

    for item in thread_messages:
        if item.get("id") == expected_latest_id:
            return thread_messages, item

    reloaded_messages = load_thread_messages(selection["thread_id"], args, 0)
    if not reloaded_messages:
        raise SystemExit("selected thread did not return any local messages after reload")

    for item in reloaded_messages:
        if item.get("id") == expected_latest_id:
            return reloaded_messages, item

    return reloaded_messages, reloaded_messages[-1]


def analyze_thread(selection: dict[str, Any], thread_messages: list[dict[str, Any]], wants_reply: bool) -> dict[str, Any]:
    latest = thread_messages[-1]
    message = latest.get("message", {})
    content = message.get("content", {})
    codes = message.get("codes") or []
    error_context = message.get("error_context")

    if codes:
        return {
            "decision": "capture_code",
            "summary": f"Latest thread message contains {len(codes)} extracted code(s).",
        }

    if error_context:
        return {
            "decision": "escalate_delivery_error",
            "summary": error_context.get("diagnostic_code") or error_context.get("status_code") or "Delivery error detected.",
        }

    if wants_reply:
        return {
            "decision": "draft_reply",
            "summary": content.get("snippet") or content.get("body_md") or selection.get("last_message_preview") or "",
        }

    return {
        "decision": "review",
        "summary": content.get("snippet") or content.get("body_md") or selection.get("last_message_preview") or "",
    }


def analyze_with_provider(
    selection: dict[str, Any],
    thread_summaries: list[dict[str, Any]],
    thread_messages: list[dict[str, Any]],
    latest_message: dict[str, Any],
    args: argparse.Namespace,
) -> dict[str, Any]:
    if args.agent_provider == "external":
        payload = {
            "source": build_source(args),
            "selection": selection,
            "thread_summaries": thread_summaries,
            "thread_messages": thread_messages,
            "latest_message": latest_message,
            "wants_reply": bool(args.reply_text),
        }
        output = run_mailcli(
            [args.provider_command, *args.provider_arg],
            stdin=json.dumps(payload),
        )
        analysis = parse_external_analysis(output)
        analysis.setdefault("provider", "external")
        return analysis

    analysis = analyze_thread(selection, thread_messages, wants_reply=bool(args.reply_text))
    analysis.setdefault("provider", "builtin")
    return analysis


def build_reply_draft(selection: dict[str, Any], latest_message: dict[str, Any], args: argparse.Namespace) -> dict[str, Any]:
    message = latest_message.get("message", {})
    meta = message.get("meta", {})
    sender = meta.get("from") or {}
    sender_address = sender.get("address")
    if not sender_address:
        raise SystemExit("latest thread message does not contain a sender address for reply drafting")

    draft: dict[str, Any] = {
        "from": {"address": args.from_address},
        "to": [{"address": sender_address}],
        "body_text": "",
    }
    if args.from_name:
        draft["from"]["name"] = args.from_name
    if sender.get("name"):
        draft["to"][0]["name"] = sender["name"]

    account = (args.account or latest_message.get("account") or "").strip()
    if account:
        draft["account"] = account

    if args.config:
        draft["reply_to_id"] = latest_message.get("id")
    else:
        references = list(meta.get("references") or [])
        message_id = meta.get("message_id")
        if message_id and message_id not in references:
            references.append(message_id)
        draft["reply_to_message_id"] = message_id
        draft["references"] = references
        draft["subject"] = meta.get("subject") or selection.get("subject") or ""

    return draft


def compile_reply_dry_run(draft: dict[str, Any], args: argparse.Namespace) -> str:
    command = [args.mailcli_bin, "reply", "--dry-run"]
    if args.config:
        command.extend(["--config", args.config])
    if args.account:
        command.extend(["--account", args.account])
    command.append("-")
    return run_mailcli(command, stdin=json.dumps(draft))


def run_mailcli_json(command: list[str], stdin: str | None = None) -> Any:
    output = run_mailcli(command, stdin=stdin)
    return json.loads(output)


def run_mailcli(command: list[str], stdin: str | None = None) -> str:
    result = subprocess.run(
        command,
        input=stdin,
        capture_output=True,
        text=True,
        check=False,
    )
    if result.returncode != 0:
        message = result.stderr.strip() or result.stdout.strip() or "mailcli command failed"
        raise SystemExit(message)
    return result.stdout


if __name__ == "__main__":
    raise SystemExit(main())
