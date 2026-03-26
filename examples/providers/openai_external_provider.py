import json
import os
import sys
from typing import Any

ALLOWED_DECISIONS = {
    "review",
    "capture_code",
    "draft_reply",
    "escalate_delivery_error",
}

SYSTEM_PROMPT = """You are an email analysis provider for MailCLI.

Return JSON that matches the required schema.
Choose exactly one decision from:
- review
- capture_code
- draft_reply
- escalate_delivery_error

Use:
- capture_code when the message includes verification codes in message.codes
- escalate_delivery_error when the message includes error_context
- draft_reply only when wants_reply is true and a short safe reply is appropriate
- review otherwise

Keep summary short. Only include reply_text when decision is draft_reply.
"""

OUTPUT_SCHEMA: dict[str, Any] = {
    "type": "object",
    "additionalProperties": False,
    "properties": {
        "decision": {
            "type": "string",
            "enum": sorted(ALLOWED_DECISIONS),
        },
        "summary": {
            "type": "string",
        },
        "reply_text": {
            "type": "string",
        },
    },
    "required": ["decision", "summary"],
}


def main() -> int:
    api_key = os.environ.get("OPENAI_API_KEY", "").strip()
    if not api_key:
        print("OPENAI_API_KEY is required", file=sys.stderr)
        return 2

    try:
        from openai import OpenAI
    except ImportError:
        print("openai package is required. Install it with: pip install openai", file=sys.stderr)
        return 2

    payload = json.load(sys.stdin)
    model = os.environ.get("OPENAI_MODEL", "gpt-5-mini").strip() or "gpt-5-mini"

    client = OpenAI()
    response = client.responses.create(
        model=model,
        input=[
            {
                "role": "system",
                "content": [
                    {
                        "type": "input_text",
                        "text": SYSTEM_PROMPT,
                    }
                ],
            },
            {
                "role": "user",
                "content": [
                    {
                        "type": "input_text",
                        "text": json.dumps(payload, ensure_ascii=False),
                    }
                ],
            },
        ],
        text={
            "format": {
                "type": "json_schema",
                "name": "mailcli_agent_decision",
                "schema": OUTPUT_SCHEMA,
                "strict": True,
            }
        },
    )

    try:
        result = json.loads(response.output_text)
    except json.JSONDecodeError as exc:
        print(f"OpenAI provider returned invalid JSON: {exc}", file=sys.stderr)
        return 1

    validation_error = validate_result(result)
    if validation_error:
        print(validation_error, file=sys.stderr)
        return 1

    json.dump(result, sys.stdout, ensure_ascii=False)
    sys.stdout.write("\n")
    return 0


def validate_result(result: Any) -> str | None:
    if not isinstance(result, dict):
        return "OpenAI provider must return a JSON object"

    decision = result.get("decision")
    if not isinstance(decision, str) or not decision.strip():
        return "OpenAI provider response must include a non-empty decision"
    if decision not in ALLOWED_DECISIONS:
        allowed = ", ".join(sorted(ALLOWED_DECISIONS))
        return f"OpenAI provider decision must be one of: {allowed}"

    summary = result.get("summary")
    if not isinstance(summary, str) or not summary.strip():
        return "OpenAI provider response must include a non-empty summary"

    reply_text = result.get("reply_text")
    if reply_text is not None and not isinstance(reply_text, str):
        return "OpenAI provider reply_text must be a string when present"

    return None


if __name__ == "__main__":
    raise SystemExit(main())
