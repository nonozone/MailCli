import json
import sys
from typing import Any


def main() -> int:
    payload = json.load(sys.stdin)
    result = analyze(payload)
    json.dump(result, sys.stdout, ensure_ascii=False)
    sys.stdout.write("\n")
    return 0


def analyze(payload: dict[str, Any]) -> dict[str, Any]:
    message = payload.get("message") or {}
    if not message:
        latest = payload.get("latest_message") or {}
        if isinstance(latest, dict):
            message = latest.get("message") or {}
    content = message.get("content", {})
    codes = message.get("codes") or []
    error_context = message.get("error_context")
    wants_reply = bool(payload.get("wants_reply"))

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
            "summary": content.get("snippet") or content.get("body_md") or "",
            "reply_text": "Thanks for your email.",
        }

    return {
        "decision": "review",
        "summary": content.get("snippet") or content.get("body_md") or "",
    }


if __name__ == "__main__":
    raise SystemExit(main())
