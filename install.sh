#!/bin/sh
# Install kubecom into a directory you own, with no package manager and no root:
# the launcher at <bindir>/kubecom and a complete binary as a "seed" in the
# launcher's home (~/.local/kubecom). The launcher runs the seed, so the first
# run needs no network and kubecom keeps running if the channel is down;
# `kubecom update` then moves the complete binary in the home, outside the
# package manager.
#
#   curl -fsSL https://raw.githubusercontent.com/neuroplastio/kubecom/v1/install.sh | sh
#
# It trusts nothing but the release key pinned below, and uses only tools already
# on the machine: curl, ssh-keygen, sha256sum (or shasum). Read from the
# environment:
#   KUBECOM_URL      where channels are served  (https://pkg.neuroplast.io)
#   KUBECOM_CHANNEL  the channel                (stable)
#   KUBECOM_COMMIT   a specific build           (the channel's newest)
#   KUBECOM_BINDIR   where the launcher goes    (~/.local/bin)
#   KUBECOM_FORCE    reseed the home even if a build is installed (empty)
#   KUBECOM_SIGNERS  override the pinned key    (the release key below)
#   KUBECOM_TOKEN    a private channel's token  (none)
set -eu

url="${KUBECOM_URL:-https://pkg.neuroplast.io}"
project=kubecom
channel="${KUBECOM_CHANNEL:-stable}"
base="$url/$project/$channel"
bindir="${KUBECOM_BINDIR:-$HOME/.local/bin}"
home="${HOME}/.local/kubecom"
# The public half of the key every release is signed with, pinned here rather
# than fetched: a key from the server it vouches for would prove nothing.
signers="${KUBECOM_SIGNERS:-release@neuroplast.io namespaces=\"engram\" ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDCsEZcn1tiubvKQzEVs5pJ4QVXoFmSDtO+k6SdetUdf}"

case "$(uname -s)" in Linux) os=linux ;; Darwin) os=darwin ;; *) echo "install: unsupported OS $(uname -s)" >&2; exit 1 ;; esac
case "$(uname -m)" in x86_64 | amd64) arch=amd64 ;; aarch64 | arm64) arch=arm64 ;; *) echo "install: unsupported machine $(uname -m)" >&2; exit 1 ;; esac

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

# get <curl args…> — curl, sending the token if there is one. The header goes in
# on stdin: on the command line, any user on the machine could read it.
get() {
	if [ -n "${KUBECOM_TOKEN:-}" ]; then
		printf 'Authorization: Bearer %s\n' "$KUBECOM_TOKEN" | curl -fsS -H @- "$@"
	else
		curl -fsS "$@"
	fi
}
# field <key> — the value of key= on the line on stdin (the format is a split on
# spaces and a split on '=').
field() { tr ' ' '\n' | sed -n "s/^$1=//p" | head -n 1; }
sum() { if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | cut -d' ' -f1; else shasum -a 256 "$1" | cut -d' ' -f1; fi; }

commit="${KUBECOM_COMMIT:-}"
if [ -z "$commit" ]; then
	# The head is unsigned: it only says where to look.
	get -o "$tmp/head" "$base/head" ||
		{ echo "install: cannot read the head of $project/$channel" >&2; exit 1; }
	commit="$(grep '^publish ' "$tmp/head" | field commit)"
	[ -n "$commit" ] || { echo "install: $project/$channel has no live build" >&2; exit 1; }
fi

get -o "$tmp/manifest" "$base/builds/$commit/manifest"
get -o "$tmp/manifest.sig" "$base/builds/$commit/manifest.sig"

printf '%s\n' "$signers" >"$tmp/allowed_signers"
identity="${signers%% *}"
ssh-keygen -Y verify -f "$tmp/allowed_signers" -I "$identity" -n engram \
	-s "$tmp/manifest.sig" <"$tmp/manifest" >/dev/null ||
	{ echo "install: the manifest of $commit is not signed by the pinned key" >&2; exit 1; }

# Signed for another project, channel or commit is as bad as not signed.
build="$(grep '^build ' "$tmp/manifest")"
[ "$(echo "$build" | field project)" = "$project" ] &&
	[ "$(echo "$build" | field channel)" = "$channel" ] &&
	[ "$(echo "$build" | field commit)" = "$commit" ] ||
	{ echo "install: signed manifest is not for $project/$channel $commit" >&2; exit 1; }
version="$(echo "$build" | field version)"

# fetch_artifact <name> <dest> — download one artifact and check it against the
# size and sha256 the signed manifest names.
fetch_artifact() {
	line="$(grep '^artifact ' "$tmp/manifest" | grep " name=$1 " | grep " os=$os " | grep " arch=$arch " | head -n 1)"
	[ -n "$line" ] || { echo "install: no $1 for $os/$arch in $commit" >&2; exit 1; }
	path="$(echo "$line" | field path)"
	want="$(echo "$line" | field sha256)"
	get -o "$2" "$base/builds/$commit/$path"
	[ "$(sum "$2")" = "$want" ] || { echo "install: $path does not match the signed manifest" >&2; exit 1; }
}
fetch_artifact kubecom-launcher "$tmp/launcher"
fetch_artifact kubecom "$tmp/kubecom"

mkdir -p "$bindir"
cp "$tmp/launcher" "$bindir/kubecom"
chmod 755 "$bindir/kubecom"

# The seed: a complete binary in the launcher's home, so the first run needs no
# network. An existing home build (a self-updated install) is kept unless
# KUBECOM_FORCE is set — the launcher prefers it anyway.
seeded=""
if [ -n "${KUBECOM_FORCE:-}" ] || [ ! -x "$home/bin/$project" ]; then
	mkdir -p "$home/builds/$commit"
	cp "$tmp/kubecom" "$home/builds/$commit/$project"
	chmod 755 "$home/builds/$commit/$project"
	ln -sfn "builds/$commit" "$home/bin"
	seeded=", seed $home/builds/$commit"
fi

echo "installed kubecom $version ($commit), verified"
echo "  launcher $bindir/kubecom$seeded"
case ":$PATH:" in
*":$bindir:"*) ;;
*) echo "  add it to PATH:  export PATH=\"$bindir:\$PATH\"" ;;
esac
[ -n "$seeded" ] || echo "  a build was already installed in $home; kept it (KUBECOM_FORCE=1 to reseed)"
"$bindir/kubecom" version 2>/dev/null || true
