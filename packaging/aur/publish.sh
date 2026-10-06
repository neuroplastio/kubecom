#!/usr/bin/env bash
#
# Publish kubecom-bin to the AUR: kubecom-launcher, installed as kubecom
# (D293).
#
#   packaging/aur/publish.sh <release> <dir> [--push]
#
# <release> is a kubecom release tag (26.10.06); <dir> holds that release's
# kubecom-launcher_linux_amd64 and kubecom-launcher_linux_arm64, the files on
# its GitHub release. The launcher's bytes are its version, so the package
# moves only when they do:
#
#   • the AUR has no kubecom-bin yet, or other bytes: the package becomes
#     <release>-1 (or the next pkgrel, when <release> was replaced under its
#     name with another launcher);
#   • the same bytes: the package keeps its version, and is published again
#     with the next pkgrel only if packaging/aur/PKGBUILD changed;
#   • otherwise there is nothing to publish.
#
# Either way the package is built as a user's makepkg would build it, from the
# GitHub release, before anything is pushed. Without --push it stops there and
# shows what it would publish. Needs makepkg, so an Arch system; --push needs
# an SSH key the AUR account kubecom-bin belongs to has.
set -euo pipefail

if [ $# -lt 2 ] || { [ $# -eq 3 ] && [ "$3" != --push ]; } || [ $# -gt 3 ]; then
	echo "usage: $0 <release> <dir> [--push]" >&2
	exit 2
fi
release=$1 dir=$2 push=${3:-}
here=$(cd "$(dirname "$0")" && pwd)
pkg=kubecom-bin

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

if [ -n "$push" ]; then
	remote=ssh://aur@aur.archlinux.org/$pkg.git
else
	remote=https://aur.archlinux.org/$pkg.git
fi
# A package nobody has published yet clones as an empty repository, and has
# no .SRCINFO.
git -c init.defaultBranch=master -c advice.defaultBranchName=false \
	clone -q "$remote" "$work/aur" 2>&1 | grep -v 'cloned an empty repository' >&2 || true
test -d "$work/aur/.git"

srcinfo() {
	[ -f "$work/aur/.SRCINFO" ] || return 0
	sed -n "s/^\t$1 = //p" "$work/aur/.SRCINFO" | head -n 1
}
cur_ver=$(srcinfo pkgver)
cur_rel=$(srcinfo pkgrel)
cur_sums="$(srcinfo sha256sums_x86_64) $(srcinfo sha256sums_aarch64)"

sum() { sha256sum "$dir/$1" | cut -d' ' -f1; }
amd64=$(sum kubecom-launcher_linux_amd64)
arm64=$(sum kubecom-launcher_linux_arm64)

render() {
	sed -e "s/@PKGVER@/$1/" -e "s/@PKGREL@/$2/" \
		-e "s/@SHA256_X86_64@/$amd64/" -e "s/@SHA256_AARCH64@/$arm64/" \
		"$here/PKGBUILD"
}

if [ -z "$cur_ver" ]; then
	ver=$release rel=1 why="kubecom-bin's first version"
elif [ "$amd64 $arm64" != "$cur_sums" ]; then
	if [ "$release" = "$cur_ver" ]; then
		ver=$cur_ver rel=$((cur_rel + 1)) why="$release was replaced with another launcher"
	else
		ver=$release rel=1 why="the launcher changed in kubecom $release"
	fi
elif render "$cur_ver" "$cur_rel" | cmp -s - "$work/aur/PKGBUILD"; then
	echo "kubecom-bin $cur_ver-$cur_rel: kubecom $release has the same launcher; nothing to publish"
	exit 0
else
	ver=$cur_ver rel=$((cur_rel + 1)) why="the PKGBUILD changed; the launcher is still $cur_ver's"
fi

render "$ver" "$rel" > "$work/aur/PKGBUILD"
cd "$work/aur"
makepkg --printsrcinfo > .SRCINFO

# Build it as a user's makepkg would: fetched from the GitHub release, its
# sha256 checked, packaged. Everything makepkg writes stays in $work.
PKGDEST="$work/out" SRCDEST="$work/src" BUILDDIR="$work/build" \
	makepkg --force --nodeps --noconfirm >"$work/makepkg.log" 2>&1 ||
	{ cat "$work/makepkg.log" >&2; exit 1; }
built=$(ls "$work/out"/*.pkg.tar.*)
bsdtar -tf "$built" | grep -qx 'usr/bin/kubecom' ||
	{ echo "$built has no usr/bin/kubecom" >&2; exit 1; }

echo "kubecom-bin $ver-$rel: $why"
git add --intent-to-add PKGBUILD .SRCINFO
git --no-pager diff --stat
if [ -z "$push" ]; then
	git --no-pager diff
	echo "(not pushed: run with --push to publish)"
	exit 0
fi
git add PKGBUILD .SRCINFO
git -c user.name="Anatoly Rugalev" -c user.email="anatoly.rugalev@gmail.com" \
	commit -q -m "$ver-$rel: $why"
git push -q origin HEAD:master
echo "published https://aur.archlinux.org/packages/$pkg"
