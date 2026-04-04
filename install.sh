#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
binary_name="endgame"
build_path="$repo_root/$binary_name"

if [[ $# -gt 1 ]]; then
  echo "usage: ./install.sh [install-dir]" >&2
  exit 1
fi

if [[ $# -eq 1 ]]; then
  install_dir="$1"
elif [[ -w /usr/local/bin ]]; then
  install_dir="/usr/local/bin"
else
  install_dir="$HOME/.local/bin"
fi

mkdir -p "$install_dir"

echo "Building $binary_name..."
(cd "$repo_root" && go build -o "$build_path" .)

echo "Installing to $install_dir/$binary_name..."
install -m 0755 "$build_path" "$install_dir/$binary_name"

echo "Installed $binary_name to $install_dir/$binary_name"

case ":${PATH:-}:" in
  *:"$install_dir":*)
    ;;
  *)
    echo
    echo "Add $install_dir to your PATH to run '$binary_name' directly."
    ;;
esac
