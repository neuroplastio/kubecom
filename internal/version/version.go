// Package version exposes build metadata for the kubecom binary. The values are
// overridden at build time via -ldflags; local builds report "dev".
//
// A build has a name and an identity, and they are different things (as in
// margin's internal/version, which this follows). Version is the name, for
// people, never compared or sorted; Commit is the identity, the full commit the
// binary was built from; Channel is where it was published by `make dist`, and
// empty for a binary nobody published — every local build. Date is the build
// stamp the release artifacts carry.
package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

var (
	// Version is the build name ("dev" for local builds; a CalVer name for
	// release artifacts).
	Version = "dev"
	// Commit is the full git commit the binary was built from, or "none".
	Commit = "none"
	// Date is the build timestamp (RFC3339) or "unknown".
	Date = "unknown"
	// Channel is the release channel this build was published to ("dev" or
	// "stable"), empty for a build on none. Set only by `make dist`.
	Channel = ""
	// modified is "true" when the tree had uncommitted changes: the commit then
	// names code this binary does not contain. Unexported, so only the linker
	// sets it (-X reaches an unexported string var).
	modified = ""
)

// Build is one build's metadata: the identity the updater reasons about.
type Build struct {
	Version  string
	Commit   string
	Channel  string
	Date     string
	Modified bool
}

// Current is this binary. A build whose Commit was never stamped ("none", a
// plain `go build` or `go install`) falls back to the commit Go itself
// recorded, so it still names the code it holds.
func Current() Build {
	b := Build{Version: Version, Commit: Commit, Date: Date, Channel: Channel, Modified: modified == "true"}
	if b.Commit == "" || b.Commit == "none" {
		if bi, ok := debug.ReadBuildInfo(); ok {
			for _, s := range bi.Settings {
				switch s.Key {
				case "vcs.revision":
					b.Commit = s.Value
				case "vcs.modified":
					b.Modified = s.Value == "true"
				}
			}
		}
	}
	return b
}

// Published reports whether a channel holds exactly this build: stamped with a
// channel, from a clean tree, at a full commit.
func (b Build) Published() bool {
	return b.Channel != "" && !b.Modified && len(b.Commit) == 40
}

// Short is the first seven characters of the commit, with +dirty for a modified
// tree; "dev" when the commit is unknown.
func (b Build) Short() string {
	if b.Commit == "" || b.Commit == "none" {
		return "dev"
	}
	s := ShortCommit(b.Commit)
	if b.Modified {
		s += "+dirty"
	}
	return s
}

// Info returns a human-readable one-line build summary.
func Info() string {
	return fmt.Sprintf("kubecom %s (commit %s, built %s, %s)",
		Version, Commit, Date, runtime.Version())
}

// ShortCommit is any commit as a person reads it: its first seven characters.
func ShortCommit(commit string) string {
	if len(commit) > 7 {
		return commit[:7]
	}
	return commit
}
