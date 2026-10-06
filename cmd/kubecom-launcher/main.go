// Command kubecom-launcher is kubecom's thin launcher, what a package (AUR,
// Homebrew) installs as kubecom. It runs the kubecom installed in
// ~/.local/kubecom, or the seed the package ships in /usr/lib/kubecom, and the
// first time with neither it fetches the newest from the channel, checked
// against the release key. It does nothing else: `kubecom update` moves what it
// runs.
//
// It is built from internal/channel, the channel stamped into it, and engram,
// and nothing of kubecom's own (D293). It rides the version of the release it
// ships with (D295), so `kubecom --launcher-version` names the package, while
// `kubecom version` — after the handover — names the build actually running.
package main

import (
	"fmt"
	"os"

	"github.com/neuroplastio/engram/enlaunch"
	"github.com/neuroplastio/kubecom/internal/channel"
	"github.com/neuroplastio/kubecom/internal/version"
)

func main() {
	// A long flag, so it never shadows the program's own `version`/`--version`
	// (which the handover answers with the running build, not this shim).
	if len(os.Args) == 2 && os.Args[1] == "--launcher-version" {
		fmt.Printf("kubecom-launcher %s (commit %s, built %s)\n", version.Version, version.Commit, version.Date)
		return
	}
	enlaunch.Main(channel.Launcher(version.Channel))
}
