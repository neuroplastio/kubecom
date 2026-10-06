// Package update replaces the running kubecom with a build from its release
// channel.
//
// Every push to v1 is published to the dev channel of pkg.neuroplast.io with
// engram, and every stable tag to the stable channel: one signed manifest per
// build, naming the sha256 of a bare kubecom binary for each platform.
// `kubecom update` asks the build's own channel for its newest build (or a
// named commit), believes only what the release key pinned in internal/channel
// has signed, and swaps the binary it is running from for the one it fetched —
// written beside it and renamed over it, so the file is the old build or the
// new one and never half of either.
//
// A kubecom that kubecom-launcher started is not its own file to replace: the
// launcher owns ~/.local/kubecom, and update installs the build there, with
// enlaunch, for the launcher to run next.
package update

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/neuroplastio/engram"
	"github.com/neuroplastio/engram/enlaunch"
	"github.com/neuroplastio/engram/sshsig"
	"github.com/neuroplastio/kubecom/internal/channel"
	"github.com/neuroplastio/kubecom/internal/version"
)

// Timeout bounds one update: a manifest and a binary of a few tens of
// megabytes.
const Timeout = 5 * time.Minute

// Client is the channel self updates from — its own, or dev for a build on
// none — believing only the pinned key (internal/channel).
func Client(self version.Build) (*engram.Client, error) {
	l := channel.Launcher(self.Channel)
	keys, err := sshsig.ParseAllowedSigners([]byte(l.Signers))
	if err != nil {
		return nil, fmt.Errorf("update: the pinned release key: %w", err)
	}
	return &engram.Client{Base: l.Base, Project: l.Project, Channel: l.Channel, Keys: keys}, nil
}

// Self updates this kubecom: what `kubecom update [commit]` runs.
func Self(ctx context.Context, out io.Writer, commit string) error {
	self := version.Current()
	c, err := Client(self)
	if err != nil {
		return err
	}
	home, _ := enlaunch.Launched()
	return Run(ctx, Options{Client: c, Self: self, Commit: commit, Home: home, Out: out})
}

// Options is one update.
type Options struct {
	Client *engram.Client
	// Self is the build being replaced.
	Self version.Build
	// Exe is the file to replace; empty means this process's executable.
	// Symlinks are followed, so the file replaced is the one they lead to.
	Exe string
	// Home is the home of the launcher that started this kubecom
	// (enlaunch.Launched). When set, the build is installed there and Exe is
	// left alone.
	Home string
	// Commit asks for that build: a full commit, or the start of a live
	// one's. Empty means the channel's newest.
	Commit string
	Out    io.Writer
}

// Run fetches a build, checks it against the pinned key and the signed
// sha256, and puts it in place of o.Exe. When the build is the one already
// running it says so and touches nothing.
func Run(ctx context.Context, o Options) error {
	c := o.Client
	m, err := pick(ctx, c, o.Self, o.Commit)
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}
	// Nothing newer, or the build asked for is this one — a local build of
	// the same commit from a clean tree included: it is the same code.
	if m == nil || (m.Build.Commit == o.Self.Commit && !o.Self.Modified) {
		if o.Commit != "" {
			fmt.Fprintf(o.Out, "kubecom %s is already %s; nothing to do\n", o.Self.Version, version.ShortCommit(m.Build.Commit))
		} else {
			fmt.Fprintf(o.Out, "kubecom %s is already the newest build on %s; nothing to do\n", o.Self.Version, c.Channel)
		}
		return nil
	}
	old := o.Self.Version
	if !o.Self.Published() {
		old += " (local build)"
	}
	if o.Home != "" {
		// The keys o.Client believes, so that one Client is all the trust.
		var keys strings.Builder
		for _, k := range c.Keys {
			keys.WriteString(sshsig.AllowedSigners(channel.Project, engram.Namespace, k))
		}
		cfg := enlaunch.Config{Base: c.Base, Project: channel.Project, Channel: c.Channel, Signers: keys.String(), Home: o.Home}
		if _, err := enlaunch.Install(ctx, cfg, m.Build.Commit); err != nil {
			return fmt.Errorf("update: installing kubecom %s: %w", m.Build.Version, err)
		}
		fmt.Fprintf(o.Out, "kubecom %s → %s\n", old, m.Build.Version)
		fmt.Fprintf(o.Out, "installed in %s; the next kubecom you start runs it\n", o.Home)
		return nil
	}

	exe, err := resolveExe(o.Exe)
	if err != nil {
		return fmt.Errorf("update: finding the kubecom binary: %w", err)
	}
	a, ok := m.Find(channel.Project, runtime.GOOS, runtime.GOARCH)
	if !ok {
		return fmt.Errorf("update: %s on %s has no kubecom for %s/%s", m.Build.Version, c.Channel, runtime.GOOS, runtime.GOARCH)
	}

	// Claim the spot before fetching tens of megabytes: a directory this user
	// cannot write fails now, not after the download.
	tmp, err := os.CreateTemp(filepath.Dir(exe), ".kubecom-update-*")
	if err != nil {
		return writeError(exe, err)
	}
	defer func() {
		if tmp != nil {
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}()
	// Download checks the size and sha256 the signed manifest names: a
	// tampered or truncated binary is refused here and never written.
	data, err := c.Download(ctx, m, a)
	if err != nil {
		return fmt.Errorf("update: fetching kubecom %s: %w", m.Build.Version, err)
	}
	if err := install(tmp, data, exe); err != nil {
		return writeError(exe, err)
	}
	tmp = nil

	fmt.Fprintf(o.Out, "kubecom %s → %s\n", old, m.Build.Version)
	fmt.Fprintf(o.Out, "replaced %s\n", exe)
	return nil
}

// pick is the verified manifest of the build to install, or nil when the
// channel has nothing newer than self.
func pick(ctx context.Context, c *engram.Client, self version.Build, commit string) (*engram.Manifest, error) {
	if commit != "" {
		// Asked for by name: any live build, older ones included. An explicit
		// choice is the one case where going back is allowed (SPEC, Trust).
		full, err := resolve(ctx, c, commit)
		if err != nil {
			return nil, err
		}
		m, err := c.Manifest(ctx, full)
		if errors.Is(err, engram.ErrNotLive) {
			return nil, fmt.Errorf("%s is not a live build on the %s channel", version.ShortCommit(full), c.Channel)
		}
		return m, err
	}
	// The newest build — and never one older than self. A build of this
	// channel knows its own place in it: its manifest's sequence is the one
	// the channel must beat. A local build has no place, so anything goes.
	accepted := 0
	if self.Published() && self.Channel == c.Channel {
		switch own, err := c.Manifest(ctx, self.Commit); {
		case err == nil:
			accepted = own.Build.Seq
		case errors.Is(err, engram.ErrNotLive):
			// Expired or scrapped: whatever is live is newer.
		default:
			return nil, fmt.Errorf("reading the %s channel: %w", c.Channel, err)
		}
	}
	m, err := c.Latest(ctx, accepted)
	if err != nil {
		return nil, fmt.Errorf("reading the %s channel: %w", c.Channel, err)
	}
	if m == nil && accepted == 0 {
		return nil, fmt.Errorf("the %s channel holds no build", c.Channel)
	}
	return m, nil
}

// resolve turns a commit, or the start of one, into the full commit of a live
// build. The journal it reads is unsigned; it only names which manifest to
// ask for, and that manifest is checked like any other.
func resolve(ctx context.Context, c *engram.Client, commit string) (string, error) {
	commit = strings.ToLower(commit)
	if len(commit) < 4 || len(commit) > 40 || strings.Trim(commit, "0123456789abcdef") != "" {
		return "", fmt.Errorf("%q is not a commit (give at least its first four characters)", commit)
	}
	if len(commit) == 40 {
		return commit, nil
	}
	j, err := c.Journal(ctx)
	if err != nil {
		return "", fmt.Errorf("reading the %s channel: %w", c.Channel, err)
	}
	var found []string
	for _, b := range j.Live() {
		if strings.HasPrefix(b.Commit, commit) {
			found = append(found, b.Commit)
		}
	}
	switch len(found) {
	case 0:
		return "", fmt.Errorf("no live build on the %s channel starts with %s", c.Channel, commit)
	case 1:
		return found[0], nil
	}
	return "", fmt.Errorf("%s names %d builds on the %s channel; give more of the commit", commit, len(found), c.Channel)
}

// resolveExe is the file to replace: path, or this process's executable, with
// symlinks followed — replacing a link would leave the binary it names alone.
func resolveExe(path string) (string, error) {
	if path == "" {
		var err error
		if path, err = os.Executable(); err != nil {
			return "", err
		}
	}
	return filepath.EvalSymlinks(path)
}

// install writes data to tmp, makes it executable, flushes it to disk and
// renames it over exe. A rename within one directory is atomic: anyone
// starting kubecom meanwhile gets the old build or the new one, never half of
// either, and a process still running the old one keeps its file.
func install(tmp *os.File, data []byte, exe string) error {
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Chmod(0o755); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), exe); err != nil {
		return err
	}
	// The rename itself survives a crash only once the directory is flushed.
	if d, err := os.Open(filepath.Dir(exe)); err == nil {
		d.Sync()
		d.Close()
	}
	return nil
}

// writeError explains a failure to put the new binary in place. A directory
// this user cannot write is the common case — kubecom installed by a package
// manager or into a system directory — and the way out is a copy the user
// owns, not elevated rights for a tool that rewrites itself.
func writeError(exe string, err error) error {
	dir := filepath.Dir(exe)
	if errors.Is(err, fs.ErrPermission) || errors.Is(err, syscall.EROFS) {
		return fmt.Errorf("update: cannot replace %s: %s is not writable by this user. "+
			"Install kubecom into a directory you own (such as ~/.local/bin) and update that copy", exe, dir)
	}
	return fmt.Errorf("update: replacing %s: %w", exe, err)
}
