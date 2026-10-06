package main

import (
	"context"
	"io"

	"github.com/neuroplastio/kubecom/internal/update"
	"github.com/spf13/cobra"
)

// newUpdateCmd builds `kubecom update`. It takes the update as an argument,
// like the root takes its handlers, so a test can drive the command without a
// channel or a binary to replace.
func newUpdateCmd(run func(ctx context.Context, out io.Writer, commit string) error) *cobra.Command {
	return &cobra.Command{
		Use:   "update [commit]",
		Short: "Replace this kubecom with the newest build from its release channel",
		Long: `kubecom update fetches the newest build of kubecom from this kubecom's
channel on pkg.neuroplast.io (stable for a release, dev for a build of v1),
checks it against the release key built into this kubecom (the signature over
the build's manifest, then the binary's sha256), and puts it in place of the
kubecom you ran — following symlinks to the real file, written beside it and
renamed over it.

A commit (full, or its first few characters) installs that build instead,
older ones included, as long as the channel still holds it.

A local build (built from a checkout, on no channel) updates to the dev
channel's newest build; kubecom says what it replaced. If this kubecom's
directory is not writable by you, update refuses: install kubecom somewhere
you own, such as ~/.local/bin.

A kubecom that kubecom-launcher started (a package's kubecom) installs the
build into ~/.local/kubecom instead, and the next kubecom you start runs it.

The server, the key and the channel are built into kubecom; nothing moves
them. To move to another channel, install a kubecom from it.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			commit := ""
			if len(args) == 1 {
				commit = args[0]
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), update.Timeout)
			defer cancel()
			return run(ctx, cmd.OutOrStdout(), commit)
		},
	}
}
