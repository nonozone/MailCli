import json
from typing import Any

ALLOWED_DECISIONS = {
    "review",
    "capture_code",
    "draft_reply",
    "escalate_delivery_error",
}


def parse_external_analysis(output: str) -> dict[str, Any]:
    try:
        analysis = json.loads(output)
    except json.JSONDecodeError as exc:
        raise SystemExit("external provider returned invalid JSON") from exc

    if not isinstance(analysis, dict):
        raise SystemExit("external provider must return a JSON object")

    decision = analysis.get("decision")
    if not isinstance(decision, str) or not decision.strip():
        raise SystemExit("external provider response must include a non-empty decision")
    if decision not in ALLOWED_DECISIONS:
        allowed = ", ".join(sorted(ALLOWED_DECISIONS))
        raise SystemExit(f"external provider decision must be one of: {allowed}")

    reply_text = analysis.get("reply_text")
    if reply_text is not None and not isinstance(reply_text, str):
        raise SystemExit("external provider reply_text must be a string when present")

    return analysis
