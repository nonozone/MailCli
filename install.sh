#!/bin/sh
set -eu

repo="${MAILCLI_REPO:-nonozone/MailCli}"
version="${MAILCLI_VERSION:-latest}"
install_dir="${MAILCLI_INSTALL_DIR:-$HOME/.local/bin}"
auto_configure="${MAILCLI_AGENT_AUTO_CONFIGURE:-0}"
agents="${MAILCLI_AGENTS:-}"
base_url_override="${MAILCLI_BASE_URL:-}"

usage() {
  cat <<'EOF'
MailCLI installer

Usage:
  install.sh [--version <tag>] [--install-dir <dir>] [--auto-configure] [--agent <codex|claude>]

Environment:
  MAILCLI_REPO=owner/repo
  MAILCLI_VERSION=v0.1.0
  MAILCLI_INSTALL_DIR=$HOME/.local/bin
  MAILCLI_AGENT_AUTO_CONFIGURE=1
  MAILCLI_AGENTS=codex,claude
  MAILCLI_BASE_URL=file:///tmp/mailcli-release
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      version="${2:?missing --version value}"
      shift 2
      ;;
    --install-dir)
      install_dir="${2:?missing --install-dir value}"
      shift 2
      ;;
    --auto-configure)
      auto_configure="1"
      shift
      ;;
    --agent)
      value="${2:?missing --agent value}"
      if [ -n "$agents" ]; then
        agents="$agents,$value"
      else
        agents="$value"
      fi
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "required command not found: $1" >&2
    exit 1
  fi
}

need_cmd curl
need_cmd tar
need_cmd install

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$os" in
  darwin) os="darwin" ;;
  linux) os="linux" ;;
  *)
    echo "unsupported OS: $os" >&2
    exit 1
    ;;
esac

arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *)
    echo "unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

asset="mailcli_${os}_${arch}.tar.gz"
if [ -n "$base_url_override" ]; then
  base_url="$base_url_override"
elif [ "$version" = "latest" ]; then
  base_url="https://github.com/${repo}/releases/latest/download"
else
  base_url="https://github.com/${repo}/releases/download/${version}"
fi

tmpdir="$(mktemp -d)"
cleanup() {
  rm -rf "$tmpdir"
}
trap cleanup EXIT INT TERM

archive="$tmpdir/$asset"
checksum_file="$tmpdir/checksums.txt"

echo "Installing MailCLI from $repo ($version) for $os/$arch"
curl -fsSL "$base_url/$asset" -o "$archive"

if curl -fsSL "$base_url/checksums.txt" -o "$checksum_file"; then
  expected="$(awk -v file="$asset" '$2 == file { print $1 }' "$checksum_file")"
  if [ -z "$expected" ]; then
    echo "checksum entry not found for $asset" >&2
    exit 1
  fi
  if command -v sha256sum >/dev/null 2>&1; then
    actual="$(sha256sum "$archive" | awk '{ print $1 }')"
  elif command -v shasum >/dev/null 2>&1; then
    actual="$(shasum -a 256 "$archive" | awk '{ print $1 }')"
  else
    echo "warning: sha256sum/shasum not found; skipping checksum verification" >&2
    actual="$expected"
  fi
  if [ "$actual" != "$expected" ]; then
    echo "checksum mismatch for $asset" >&2
    exit 1
  fi
else
  echo "warning: checksums.txt not available; skipping checksum verification" >&2
fi

mkdir -p "$install_dir"
tar -xzf "$archive" -C "$tmpdir"
install -m 0755 "$tmpdir/mailcli" "$install_dir/mailcli"

target="$install_dir/mailcli"
echo "Installed: $target"

case ":$PATH:" in
  *":$install_dir:"*) ;;
  *)
    echo "warning: $install_dir is not in PATH; add it to your shell profile or call $target directly" >&2
    ;;
esac

if [ "$auto_configure" = "1" ] || [ "$auto_configure" = "true" ]; then
  set -- agent configure --mailcli-bin "$target"
  if [ -n "$agents" ]; then
    old_ifs="$IFS"
    IFS=,
    for agent in $agents; do
      set -- "$@" --agent "$agent"
    done
    IFS="$old_ifs"
  fi
  "$target" "$@"
else
  "$target" agent doctor --mailcli-bin "$target"
  echo "To register MailCLI with detected agents, rerun with MAILCLI_AGENT_AUTO_CONFIGURE=1 or pass --auto-configure."
fi
