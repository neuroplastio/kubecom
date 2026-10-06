// Command kubecom-launcher is kubecom's thin launcher, what a package
// (AUR, Homebrew) installs as kubecom. It runs the kubecom installed in
// ~/.local/kubecom, and the first time, with none there, fetches the newest
// from the channel, checked against the release key. It does nothing else:
// `kubecom update` moves what it runs.
//
// It is built from internal/channel, the channel stamped into it, and engram,
// and nothing of kubecom's own: its bytes are its version.
package main

import (
	"github.com/neuroplastio/engram/enlaunch"
	"github.com/neuroplastio/kubecom/internal/channel"
	"github.com/neuroplastio/kubecom/internal/version"
)

func main() {
	enlaunch.Main(channel.Launcher(version.Channel))
}
