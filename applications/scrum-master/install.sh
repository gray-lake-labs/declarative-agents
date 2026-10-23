#!/bin/sh
# Installs tpm from a declarative-agents release (eng03): downloads the binary for
# this machine and the release's SHA256SUMS, refuses a binary whose checksum
# differs, and installs it.
#
#   sh install.sh [tag]
#
# With no tag it installs the latest release. TPM_PREFIX is the directory to
# install into, by default $HOME/.local/bin. TPM_RELEASE_URL is where the
# releases are published.
set -eu

tag=${1:-latest}
prefix=${TPM_PREFIX:-$HOME/.local/bin}
base=${TPM_RELEASE_URL:-https://github.com/gray-lake-labs/declarative-agents/releases}

case $(uname -s) in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  *) echo "install.sh: no release binary for $(uname -s); download tpm_windows_<arch>.exe from $base" >&2; exit 1 ;;
esac
case $(uname -m) in
  x86_64 | amd64) arch=amd64 ;;
  arm64 | aarch64) arch=arm64 ;;
  *) echo "install.sh: no release binary for $(uname -m); the releases build amd64 and arm64" >&2; exit 1 ;;
esac
asset=tpm_${os}_${arch}

if [ "$tag" = latest ]; then
  url=$base/latest/download
else
  url=$base/download/$tag
fi

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
curl -fsSL -o "$work/$asset" "$url/$asset"
curl -fsSL -o "$work/SHA256SUMS" "$url/SHA256SUMS"

want=$(awk -v a="$asset" '$2 == a || $2 == "*" a { print $1 }' "$work/SHA256SUMS")
if command -v sha256sum >/dev/null 2>&1; then
  got=$(sha256sum "$work/$asset" | awk '{ print $1 }')
else
  got=$(shasum -a 256 "$work/$asset" | awk '{ print $1 }')
fi
if [ -z "$want" ] || [ "$got" != "$want" ]; then
  echo "install.sh: $asset has checksum $got, but SHA256SUMS lists ${want:-none}; refusing it" >&2
  exit 1
fi

mkdir -p "$prefix"
chmod 0755 "$work/$asset"
mv "$work/$asset" "$prefix/tpm"
echo "installed $("$prefix/tpm" --version) at $prefix/tpm"
case ":$PATH:" in
  *":$prefix:"*) ;;
  *) echo "add $prefix to PATH" ;;
esac
echo "next: tpm install --runtime"
