import argparse
import filecmp
import json
import os
import subprocess
import sys
import tempfile
from pathlib import Path
from typing import Any

CANONICAL_INDEXED_AT = "2026-03-27T15:11:13Z"
CANONICAL_MIME_DATE = "Fri, 27 Mar 2026 15:11:13 +0000"
CANONICAL_MESSAGE_ID = "<generated@mailcli.local>"
CANONICAL_INDEX_PATH = "/tmp/mailcli-fixtures-index.json"
CANONICAL_CONFIG_PATH = "examples/config/fixtures-dir.yaml"


def main() -> int:
    args = parse_args()
    if args.check:
        return run_check_mode(args)

    output_dir = Path(args.output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)
    generate_artifacts(args, output_dir)
    return 0


def generate_artifacts(args: argparse.Namespace, output_dir: Path) -> None:
    output_dir.mkdir(parents=True, exist_ok=True)

    reset_index_file(args.index)
    sync_result = run_mailcli_json(
        build_sync_command(args),
        cwd=args.workdir,
    )
    write_json(output_dir / "sync.json", normalize_demo_json(sync_result))

    threads = run_mailcli_json(
        build_threads_command(args),
        cwd=args.workdir,
    )
    write_json(output_dir / "threads.json", threads)
    if not isinstance(threads, list) or not threads:
        raise SystemExit("mailcli threads returned no local thread summaries")

    selection = threads[0]
    thread_id = selection.get("thread_id")
    if not isinstance(thread_id, str) or not thread_id:
        raise SystemExit("selected thread is missing thread_id")

    thread_messages = run_mailcli_json(
        build_thread_command(args, thread_id),
        cwd=args.workdir,
    )
    write_json(output_dir / "thread.json", normalize_demo_json(thread_messages))

    reset_index_file(args.index)
    report = run_python_json(
        build_agent_command(args),
        cwd=args.workdir,
    )
    report = normalize_demo_json(report)
    write_json(output_dir / "agent-report.json", report)

    reply = report.get("reply")
    if not isinstance(reply, dict):
        raise SystemExit("agent report is missing reply section")

    draft = reply.get("draft")
    if not isinstance(draft, dict):
        raise SystemExit("agent report reply is missing draft object")
    write_json(output_dir / "reply.draft.json", draft)

    mime = reply.get("mime")
    if not isinstance(mime, str):
        raise SystemExit("agent report reply is missing mime output")
    normalized_mime = normalize_reply_mime(mime)
    reply["mime"] = normalized_mime
    write_json(output_dir / "agent-report.json", report)
    write_text(output_dir / "reply.mime.txt", normalized_mime.rstrip() + "\n")


def run_check_mode(args: argparse.Namespace) -> int:
    with tempfile.TemporaryDirectory(prefix="mailcli-local-thread-demo-") as tmpdir:
        temp_output_dir = Path(tmpdir) / "generated"
        temp_index_path = str(Path(tmpdir) / "index.json")
        temp_args = argparse.Namespace(**vars(args))
        temp_args.output_dir = str(temp_output_dir)
        temp_args.index = temp_index_path
        temp_args.check = False
        generate_artifacts(temp_args, temp_output_dir)

        target_dir = Path(args.output_dir)
        mismatches = compare_artifact_dirs(temp_output_dir, target_dir)
        if mismatches:
            for mismatch in mismatches:
                print(mismatch, file=sys.stderr)
            return 1

    print("local-thread-demo artifacts are up to date")
    return 0


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Refresh the stored local-thread-demo artifacts from the current fixture corpus.",
    )
    parser.add_argument("--mailcli-bin", default="mailcli", help="path to the mailcli binary")
    parser.add_argument("--config", required=True, help="mailcli config path")
    parser.add_argument("--account", required=True, help="mailcli account name")
    parser.add_argument("--index", required=True, help="local index path to build during refresh")
    parser.add_argument("--output-dir", required=True, help="directory where artifacts should be written")
    parser.add_argument("--query", default="invoice", help="thread query used for demo selection")
    parser.add_argument("--mailbox", help="optional mailbox override")
    parser.add_argument("--sync-limit", type=int, default=20, help="sync limit for the demo refresh")
    parser.add_argument("--thread-limit", type=int, default=10, help="thread summary limit")
    parser.add_argument("--thread-message-limit", type=int, default=50, help="thread message limit")
    parser.add_argument("--from-address", default="support@nono.im", help="from address for reply dry-run")
    parser.add_argument(
        "--reply-text",
        default="Thanks, we have received the invoice notification.",
        help="reply body text for the generated reply dry-run",
    )
    parser.add_argument(
        "--workdir",
        help="optional working directory for command execution",
    )
    parser.add_argument(
        "--check",
        action="store_true",
        help="verify that the target artifact directory already matches freshly generated output",
    )
    return parser.parse_args()


def build_sync_command(args: argparse.Namespace) -> list[str]:
    command = [
        args.mailcli_bin,
        "sync",
        "--format",
        "json",
        "--config",
        args.config,
        "--account",
        args.account,
        "--index",
        args.index,
        "--limit",
        str(args.sync_limit),
    ]
    if args.mailbox:
        command.extend(["--mailbox", args.mailbox])
    return command


def build_threads_command(args: argparse.Namespace) -> list[str]:
    command = [
        args.mailcli_bin,
        "threads",
        "--format",
        "json",
        "--index",
        args.index,
        "--account",
        args.account,
        "--limit",
        str(args.thread_limit),
        args.query,
    ]
    if args.mailbox:
        command[6:6] = ["--mailbox", args.mailbox]
    return command


def build_thread_command(args: argparse.Namespace, thread_id: str) -> list[str]:
    command = [
        args.mailcli_bin,
        "thread",
        "--format",
        "json",
        "--index",
        args.index,
        "--account",
        args.account,
        "--limit",
        str(args.thread_message_limit),
        thread_id,
    ]
    if args.mailbox:
        command[6:6] = ["--mailbox", args.mailbox]
    return command


def build_agent_command(args: argparse.Namespace) -> list[str]:
    command = [
        sys.executable,
        str(Path(__file__).with_name("agent_thread_assistant.py")),
        "--mailcli-bin",
        args.mailcli_bin,
        "--config",
        args.config,
        "--account",
        args.account,
        "--index",
        args.index,
        "--sync-limit",
        str(args.sync_limit),
        "--thread-limit",
        str(args.thread_limit),
        "--thread-message-limit",
        str(args.thread_message_limit),
        "--query",
        args.query,
        "--from-address",
        args.from_address,
        "--reply-text",
        args.reply_text,
    ]
    if args.mailbox:
        command.extend(["--mailbox", args.mailbox])
    return command


def run_mailcli_json(command: list[str], cwd: str | None) -> Any:
    return run_json_command(command, cwd=cwd)


def run_python_json(command: list[str], cwd: str | None) -> Any:
    return run_json_command(command, cwd=cwd)


def run_json_command(command: list[str], cwd: str | None) -> Any:
    completed = subprocess.run(
        command,
        cwd=cwd,
        capture_output=True,
        text=True,
        check=False,
    )
    if completed.returncode != 0:
        raise SystemExit(completed.stderr or completed.stdout)
    try:
        return json.loads(completed.stdout)
    except json.JSONDecodeError as exc:
        raise SystemExit(f"expected JSON output from {' '.join(command)}: {exc}") from exc


def write_json(path: Path, value: Any) -> None:
    path.write_text(json.dumps(value, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")


def write_text(path: Path, value: str) -> None:
    path.write_text(value, encoding="utf-8")


def reset_index_file(path: str) -> None:
    target = Path(path)
    if target.exists():
        target.unlink()


def normalize_demo_json(value: Any) -> Any:
    if isinstance(value, dict):
        out: dict[str, Any] = {}
        for key, item in value.items():
            if key == "indexed_at" and isinstance(item, str):
                out[key] = CANONICAL_INDEXED_AT
                continue
            if key in {"index", "index_path"} and isinstance(item, str):
                out[key] = CANONICAL_INDEX_PATH
                continue
            if key == "config" and isinstance(item, str):
                out[key] = CANONICAL_CONFIG_PATH
                continue
            out[key] = normalize_demo_json(item)
        return out
    if isinstance(value, list):
        return [normalize_demo_json(item) for item in value]
    return value


def normalize_reply_mime(mime: str) -> str:
    lines = []
    for line in mime.splitlines():
        if line.startswith("Message-ID: "):
            lines.append(f"Message-ID: {CANONICAL_MESSAGE_ID}")
            continue
        if line.startswith("Date: "):
            lines.append(f"Date: {CANONICAL_MIME_DATE}")
            continue
        lines.append(line)
    return "\n".join(lines)


def compare_artifact_dirs(generated: Path, target: Path) -> list[str]:
    expected_files = sorted(path.name for path in generated.iterdir() if path.is_file())
    mismatches: list[str] = []

    for name in expected_files:
        generated_path = generated / name
        target_path = target / name
        if not target_path.exists():
            mismatches.append(f"missing artifact: {target_path}")
            continue
        if not filecmp.cmp(generated_path, target_path, shallow=False):
            mismatches.append(f"artifact drift: {target_path}")

    target_files = sorted(path.name for path in target.iterdir() if path.is_file())
    for name in target_files:
        if name not in expected_files:
            mismatches.append(f"unexpected artifact: {target / name}")

    return mismatches


if __name__ == "__main__":
    raise SystemExit(main())
