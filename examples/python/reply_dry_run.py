import subprocess
import sys


def main() -> int:
    if len(sys.argv) != 2:
        print("usage: python reply_dry_run.py <reply.json>", file=sys.stderr)
        return 1

    result = subprocess.run(
        ["mailcli", "reply", "--dry-run", sys.argv[1]],
        capture_output=True,
        text=True,
        check=True,
    )

    print(result.stdout)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
