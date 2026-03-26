#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: ./parse_email.sh <email.eml>" >&2
  exit 1
fi

mailcli parse --format json "$1"
