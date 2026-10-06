// Package channel is where kubecom is published and how a build of it is
// believed: the server, kubecom's name on it, the release key, and which
// channel a build follows (mirrors margin's internal/channel, and plx's; the
// organisations' channels are one design).
//
// It is everything kubecom-launcher is made of besides engram, and it is kept
// apart from kubecom's updater on purpose. The launcher is versioned by its
// bytes: a binary keeps the line of every function it links, so an edit to a
// file the launcher links — a comment that moves a line included — is a new
// launcher. Nothing belongs here that the launcher does not need.
package channel

import "github.com/neuroplastio/engram/enlaunch"

const (
	// URL is where channels are served, pinned like the key: no release can
	// be pointed at another server.
	URL = "https://pkg.neuroplast.io"

	// Project is kubecom's name on the channel server.
	Project = "kubecom"

	// Default is the channel a build on none — any local build — follows.
	Default = "dev"
)

// releaseSigners is the public half of the key every published kubecom is
// signed with, pinned here: a key fetched from the server it vouches for would
// prove nothing. The same key signs engram, plx and margin. Rotation is a
// list — a release adds the next key before the old one retires.
const releaseSigners = `release@neuroplast.io namespaces="engram" ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDCsEZcn1tiubvKQzEVs5pJ4QVXoFmSDtO+k6SdetUdf`

// signers is the allowed_signers text a channel is checked against, and
// pkgURL the server it is read from. Always the pinned key and URL, except in
// a binary built with -tags updatetest, which no release is (see
// seams_updatetest.go).
var (
	signers = func() string { return releaseSigners }
	pkgURL  = func() string { return URL }
)

// Launcher is the channel name as enlaunch reads it, the default for "":
// what kubecom-launcher runs with, and what kubecom's update reads and
// installs into the launcher's home with.
func Launcher(name string) enlaunch.Config {
	if name == "" {
		name = Default
	}
	return enlaunch.Config{Base: pkgURL(), Project: Project, Channel: name, Signers: signers()}
}
