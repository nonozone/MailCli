#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: ./reply_dry_run.sh <reply.json>" >&2
  exit 1
fi

mailcli reply --dry-run "$1"
