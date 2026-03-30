#!/usr/bin/env python3
"""
mailcli AI Reply Agent — minimal example of a watch → LLM → reply pipeline.

Usage:
    mailcli watch --account work | python3 tools/agent_example.py

Environment variables:
    ANTHROPIC_API_KEY   or   OPENAI_API_KEY
    MAILCLI_AUTO_SEND   set to "1" to send replies automatically (default: draft only)
    MAILCLI_ACCOUNT     account name passed to mailcli reply
    MAILCLI_DRY_RUN     set to "1" to print the MIME instead of sending

Requirements:
    pip install anthropic   # or openai
"""

import json
import os
import subprocess
import sys
from datetime import datetime, timezone

# ── Configuration ─────────────────────────────────────────────────────────────

ACCOUNT     = os.environ.get("MAILCLI_ACCOUNT", "")
AUTO_SEND   = os.environ.get("MAILCLI_AUTO_SEND", "0") == "1"
DRY_RUN     = os.environ.get("MAILCLI_DRY_RUN", "0") == "1"

SYSTEM_PROMPT = """You are a helpful email assistant. When given an incoming email,
draft a concise, professional reply. Output ONLY a JSON object with these fields:
{
  "should_reply": true/false,
  "reply_body_md": "Your reply in Markdown",
  "reasoning": "one-sentence reason"
}
If the email is a newsletter, automated notification, or doesn't warrant a reply,
set should_reply to false."""


# ── LLM call (Anthropic by default, falls back to OpenAI) ────────────────────

def ask_llm(email_json: dict) -> dict:
    prompt = f"Incoming email:\n```json\n{json.dumps(email_json, ensure_ascii=False, indent=2)}\n```"

    if os.environ.get("ANTHROPIC_API_KEY"):
        import anthropic
        client = anthropic.Anthropic()
        msg = client.messages.create(
            model="claude-3-5-sonnet-20241022",
            max_tokens=1024,
            system=SYSTEM_PROMPT,
            messages=[{"role": "user", "content": prompt}],
        )
        raw = msg.content[0].text
    elif os.environ.get("OPENAI_API_KEY"):
        from openai import OpenAI
        client = OpenAI()
        resp = client.chat.completions.create(
            model="gpt-4o-mini",
            messages=[
                {"role": "system", "content": SYSTEM_PROMPT},
                {"role": "user",   "content": prompt},
            ],
            response_format={"type": "json_object"},
        )
        raw = resp.choices[0].message.content
    else:
        print("[agent] No LLM API key found — skipping", file=sys.stderr)
        return {"should_reply": False}

    try:
        return json.loads(raw)
    except json.JSONDecodeError:
        print(f"[agent] LLM returned non-JSON: {raw}", file=sys.stderr)
        return {"should_reply": False}


# ── mailcli helper ────────────────────────────────────────────────────────────

def mailcli_reply(original_msg: dict, reply_body_md: str) -> None:
    meta = original_msg.get("meta", {})
    frm  = meta.get("from") or {}

    # Build ReplyDraft JSON matching mailcli's schema.
    reply_draft = {
        "reply_to_message_id": original_msg.get("id", ""),
        "references": original_msg.get("references", [original_msg.get("id", "")]),
        "to": [{"name": frm.get("name", ""), "address": frm.get("address", "")}],
        "subject": meta.get("subject", ""),
        "body_md": reply_body_md,
    }

    cmd = ["mailcli", "reply", json.dumps(reply_draft)]
    if ACCOUNT:
        cmd += ["--account", ACCOUNT]
    if DRY_RUN:
        cmd += ["--dry-run"]

    result = subprocess.run(cmd, capture_output=True, text=True)
    if result.returncode != 0:
        print(f"[agent] reply failed: {result.stderr}", file=sys.stderr)
    else:
        print(f"[agent] {'sent' if not DRY_RUN else 'dry-run'}: {json.loads(result.stdout)}", file=sys.stderr)


# ── Main event loop ───────────────────────────────────────────────────────────

def main():
    print("[agent] starting — reading from stdin (mailcli watch output)", file=sys.stderr)

    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue

        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            continue

        etype = event.get("event")

        if etype == "watching":
            print(f"[agent] monitoring {event.get('account')}/{event.get('mailbox')}", file=sys.stderr)

        elif etype == "new_message":
            msg = event.get("message") or {}
            subject = (msg.get("meta") or {}).get("subject", "(no subject)")
            print(f"[agent] new message: {subject}", file=sys.stderr)

            decision = ask_llm(msg)

            if not decision.get("should_reply"):
                print(f"[agent] skip ({decision.get('reasoning', 'no reason')})", file=sys.stderr)
                continue

            reply_body = decision.get("reply_body_md", "")
            print(f"[agent] replying — {decision.get('reasoning', '')}", file=sys.stderr)

            if AUTO_SEND:
                mailcli_reply(msg, reply_body)
            else:
                # Human-in-the-loop: print draft to stdout for review.
                draft = {
                    "ts": datetime.now(timezone.utc).isoformat(),
                    "in_reply_to": msg.get("id"),
                    "subject": (msg.get("meta") or {}).get("subject"),
                    "draft_body_md": reply_body,
                    "reasoning": decision.get("reasoning"),
                }
                print(json.dumps(draft, ensure_ascii=False))

        elif etype == "error":
            print(f"[agent] error: {event.get('error')}", file=sys.stderr)

        elif etype == "heartbeat":
            pass  # ignore


if __name__ == "__main__":
    main()
