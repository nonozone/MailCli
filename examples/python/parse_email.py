import json
import subprocess
import sys


def main() -> int:
    if len(sys.argv) != 2:
        print("usage: python parse_email.py <email.eml>", file=sys.stderr)
        return 1

    result = subprocess.run(
        ["mailcli", "parse", sys.argv[1]],
        capture_output=True,
        text=True,
        check=True,
    )

    message = json.loads(result.stdout)
    print(message["content"]["body_md"])
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
