#!/usr/bin/env bash
#
# Publish kubecom's Homebrew formula to the org tap `neuroplastio/homebrew-tap`
# (installed with `brew install neuroplastio/tap/kubecom`).
#
#   packaging/homebrew/publish.sh <release> [--push]
#
# <release> is a kubecom release tag (26.10.07). The script downloads that
# release's eight artifacts — the launcher and the seed for linux/darwin ×
# amd64/arm64 — renders `Formula/kubecom.rb` from `packaging/homebrew/kubecom.rb`
# with their sha256s, and (with `--push`) commits and pushes it to the tap.
#
# Needs curl, sha256sum (or shasum) and git. On --push it uses HOMEBREW_TAP_TOKEN
# when set (the CI path), and your own git credentials otherwise.
set -euo pipefail

if [ $# -lt 1 ] || [ $# -gt 2 ] || { [ $# -eq 2 ] && [ "$2" != --push ]; }; then
	echo "usage: $0 <release> [--push]" >&2
	exit 2
fi
release=$1 push=${2:-}
here=$(cd "$(dirname "$0")" && pwd)
tap=neuroplastio/homebrew-tap
urlbase="https://github.com/neuroplastio/kubecom/releases/download/$release"

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

# The commit the release names — the seed's build directory in the home. Prefer
# the tag in this checkout; fall back to the GitHub API.
commit="$(git -C "$here" rev-parse --verify -q "${release}^{commit}" 2>/dev/null || true)"
if [ -z "$commit" ]; then
	commit="$(gh api "repos/neuroplastio/kubecom/commits/$release" -q .sha 2>/dev/null || true)"
fi
[ -n "$commit" ] || { echo "publish: cannot resolve $release to a commit" >&2; exit 1; }

sum() { if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | cut -d' ' -f1; else shasum -a 256 "$1" | cut -d' ' -f1; fi; }

assets="kubecom-launcher_darwin_amd64 kubecom-launcher_darwin_arm64 \
	kubecom-launcher_linux_amd64 kubecom-launcher_linux_arm64 \
	kubecom_darwin_amd64 kubecom_darwin_arm64 \
	kubecom_linux_amd64 kubecom_linux_arm64"
for f in $assets; do
	curl -fsSL "$urlbase/$f" -o "$work/$f"
done

platform() { case "$1" in *darwin_amd64) echo DARWIN_AMD64 ;; *darwin_arm64) echo DARWIN_ARM64 ;;
	*linux_amd64) echo LINUX_AMD64 ;; *linux_arm64) echo LINUX_ARM64 ;; esac; }

render() {
	local args=(-e "s/@VERSION@/$release/g" -e "s/@COMMIT@/$commit/") f
	for f in kubecom-launcher_darwin_amd64 kubecom-launcher_darwin_arm64 \
		kubecom-launcher_linux_amd64 kubecom-launcher_linux_arm64; do
		args+=(-e "s/@LAUNCHER_$(platform "$f")@/$(sum "$work/$f")/")
	done
	for f in kubecom_darwin_amd64 kubecom_darwin_arm64 kubecom_linux_amd64 kubecom_linux_arm64; do
		args+=(-e "s/@SEED_$(platform "$f")@/$(sum "$work/$f")/")
	done
	sed "${args[@]}" "$here/kubecom.rb"
}

if [ -n "$push" ] && [ -n "${HOMEBREW_TAP_TOKEN:-}" ]; then
	remote="https://x-access-token:$HOMEBREW_TAP_TOKEN@github.com/$tap.git"
else
	remote="https://github.com/$tap.git"
fi
git clone -q "$remote" "$work/tap"
mkdir -p "$work/tap/Formula"
render > "$work/tap/Formula/kubecom.rb"

cd "$work/tap"
git add Formula/kubecom.rb
if git diff --cached --quiet; then
	echo "kubecom $release: the formula is unchanged; nothing to publish"
	exit 0
fi
git --no-pager diff --cached --stat
if [ -z "$push" ]; then
	echo "(not pushed: run with --push to publish)"
	exit 0
fi
git -c user.name="Kubecom Agent" -c user.email="kubecom@neuroplast.io" \
	commit -q -m "kubecom $release"
git push -q origin HEAD
echo "published https://github.com/$tap"
