package tui

import (
	"context"
	"fmt"
	"io"

	tea "charm.land/bubbletea/v2"

	"github.com/neuroplastio/kubecom/internal/kube"
	"github.com/neuroplastio/kubecom/internal/update"
)

// This file is the `:update` palette verb: the one action that hands the
// terminal over to replace the running kubecom with the newest build from its
// release channel (internal/update). It is app-global and has no default key —
// updating is rare, so it is reached by typing `:update` rather than by spending
// a key. Like every suspend it goes through Model.suspend (handover), and like
// the other offline actions it is inert without a channel only in the sense that
// it fails with a clear toast; there is no seam to wire, because the channel is
// compiled into the binary (internal/channel).

// runUpdate suspends the TUI and replaces this kubecom with the newest build of
// its channel, or the named commit. The terminal already shows update's own
// progress during the suspend; the callback reports the outcome on the status
// bar once bubbletea has reclaimed the screen.
func (m *Model) runUpdate() (tea.Model, tea.Cmd) {
	return m, m.suspend(&updateCommand{}, func(err error) tea.Msg { return updateDoneMsg{err: err} })
}

// updateDoneMsg carries the outcome of one update once tea.Exec resumes the TUI.
type updateDoneMsg struct{ err error }

// handleUpdateDone reports an update: a notice asking for a restart — a running
// process keeps the build it started with, so the new one is what the *next*
// launch runs — or the failure as a toast. A "nothing to do" update returns nil
// and reads as the neutral notice too; its own line ("already the newest") was
// printed to the terminal during the suspend.
func (m *Model) handleUpdateDone(msg updateDoneMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		return m, m.surfaceError(ErrorMsg{Context: "update", Err: msg.err, Kind: kube.KindUnknown})
	}
	return m, m.surfaceNotice("updated — restart kubecom to run the new build")
}

// updateCommand adapts internal/update to bubbletea's ExecCommand (tea.Exec), so
// the fetch and the swap run with the terminal released, off the update loop
// (D124's rhythm). It runs in-process, not as a child `kubecom update`: a
// kubecom a launcher started must see ENLAUNCH_HOME, which enlaunch unsets from
// the environment the moment it is read — a grandchild would not, and would
// replace the build binary instead of the launcher's home.
type updateCommand struct {
	out io.Writer
	err io.Writer
}

func (c *updateCommand) SetStdin(io.Reader)    {}
func (c *updateCommand) SetStdout(w io.Writer) { c.out = w }
func (c *updateCommand) SetStderr(w io.Writer) { c.err = w }

func (c *updateCommand) Run() error {
	ctx, cancel := context.WithTimeout(context.Background(), update.Timeout)
	defer cancel()
	err := update.Self(ctx, c.out, "")
	if err != nil && c.err != nil {
		_, _ = fmt.Fprintf(c.err, "kubecom update: %v\n", err)
	}
	return err
}
