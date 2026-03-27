import argparse
import json
import subprocess
import sys
from typing import Any

from provider_contract import parse_external_analysis


def main() -> int:
    args = parse_args()

    message = load_message(args)
    analysis = analyze_with_provider(message, args)
    report: dict[str, Any] = {
        "tool": "mailcli-agent-example",
        "source": build_source(args),
        "message": message,
        "analysis": analysis,
    }

    reply_text = args.reply_text or analysis.get("reply_text")
    if reply_text:
        if not args.from_address:
            print("--from-address is required when --reply-text is used", file=sys.stderr)
            return 2

        draft = build_reply_draft(message, args)
        draft["body_text"] = reply_text
        mime = run_mailcli(
            [args.mailcli_bin, "reply", "--dry-run", "-"],
            stdin=json.dumps(draft),
        )
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
        description="Minimal agent-style MailCLI workflow example.",
    )
    parser.add_argument("--mailcli-bin", default="mailcli", help="path to the mailcli binary")
    parser.add_argument("--email", help="local .eml file to parse")
    parser.add_argument("--message-id", help="message id to fetch through mailcli get")
    parser.add_argument("--config", help="mailcli config path for inbox-backed commands")
    parser.add_argument("--account", help="mailcli account override")
    parser.add_argument("--reply-text", help="optional reply body text to compile with mailcli reply --dry-run")
    parser.add_argument("--from-address", help="from address to use for reply dry-run")
    parser.add_argument("--from-name", help="optional from display name for reply dry-run")
    parser.add_argument("--agent-provider", choices=["builtin", "external"], default="builtin", help="analysis provider")
    parser.add_argument("--provider-command", help="external provider command")
    parser.add_argument("--provider-arg", action="append", default=[], help="repeatable argument for the external provider command")

    args = parser.parse_args()
    if bool(args.email) == bool(args.message_id):
        parser.error("exactly one of --email or --message-id is required")
    if args.agent_provider == "external" and not args.provider_command:
        parser.error("--provider-command is required when --agent-provider external is used")
    return args


def build_source(args: argparse.Namespace) -> dict[str, Any]:
    if args.email:
        return {"mode": "email", "value": args.email}
    return {
        "mode": "message_id",
        "value": args.message_id,
        "config": args.config,
        "account": args.account,
    }


def load_message(args: argparse.Namespace) -> dict[str, Any]:
    if args.email:
        output = run_mailcli([args.mailcli_bin, "parse", "--format", "json", args.email])
    else:
        command = [args.mailcli_bin, "get", "--format", "json"]
        if args.config:
            command.extend(["--config", args.config])
        if args.account:
            command.extend(["--account", args.account])
        command.append(args.message_id)
        output = run_mailcli(command)
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


def analyze_message(message: dict[str, Any], wants_reply: bool) -> dict[str, Any]:
    content = message.get("content", {})
    snippet = content.get("snippet") or content.get("body_md") or ""
    codes = message.get("codes") or []
    error_context = message.get("error_context")

    if codes:
        first_code = codes[0]
        summary = f"Verification email with {len(codes)} code(s)."
        if first_code.get("expires_in_seconds"):
            summary += f" First code expires in {first_code['expires_in_seconds']} seconds."
        return {
            "decision": "capture_code",
            "summary": summary,
        }

    if error_context:
        return {
            "decision": "escalate_delivery_error",
            "summary": error_context.get("diagnostic_code") or error_context.get("status_code") or "Delivery error detected.",
        }

    if wants_reply:
        return {
            "decision": "draft_reply",
            "summary": snippet,
        }

    return {
        "decision": "review",
        "summary": snippet,
    }


def analyze_with_provider(message: dict[str, Any], args: argparse.Namespace) -> dict[str, Any]:
    if args.agent_provider == "external":
        payload = {
            "source": build_source(args),
            "message": message,
            "wants_reply": bool(args.reply_text),
        }
        output = run_mailcli(
            [args.provider_command, *args.provider_arg],
            stdin=json.dumps(payload),
        )
        analysis = parse_external_analysis(output)
        analysis.setdefault("provider", "external")
        return analysis

    analysis = analyze_message(message, wants_reply=bool(args.reply_text))
    analysis.setdefault("provider", "builtin")
    return analysis


def build_reply_draft(message: dict[str, Any], args: argparse.Namespace) -> dict[str, Any]:
    meta = message.get("meta", {})
    sender = meta.get("from") or {}
    sender_address = sender.get("address")
    if not sender_address:
        raise SystemExit("message does not contain a sender address for reply drafting")

    references = list(meta.get("references") or [])
    message_id = meta.get("message_id")
    if message_id and message_id not in references:
        references.append(message_id)

    draft: dict[str, Any] = {
        "from": {"address": args.from_address},
        "to": [{"address": sender_address}],
        "body_text": args.reply_text or "",
        "reply_to_message_id": message_id,
        "references": references,
        "subject": meta.get("subject") or "",
    }
    if sender.get("name"):
        draft["to"][0]["name"] = sender["name"]
    if args.from_name:
        draft["from"]["name"] = args.from_name
    if args.account:
        draft["account"] = args.account

    return draft


if __name__ == "__main__":
    raise SystemExit(main())
