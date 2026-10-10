#!/bin/sh
# kubecom's installer is on its website (D301):
#
#   curl -fsSL https://kubecom.neuroplast.io/install.sh | sh
#
# This file stays so the one-liner that piped it from this repository keeps
# working: it fetches the installer from the website and runs it, with the
# same environment (KUBECOM_CHANNEL, KUBECOM_BINDIR, …). The installer
# itself trusts nothing but the release key pinned in it.
set -eu

url=https://kubecom.neuroplast.io/install.sh
script="$(curl -fsSL "$url")" || { echo "install: cannot fetch $url" >&2; exit 1; }
exec sh -c "$script"
