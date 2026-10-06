// Package testchannel is a kubecom release channel for tests: published into a
// directory with engram's own publisher, signed by a throwaway key, and served
// over HTTP — the same files pkg.neuroplast.io serves, minus the CDN.
package testchannel

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/neuroplastio/engram"
	"github.com/neuroplastio/engram/sshsig"
	"github.com/neuroplastio/engram/store"
)

// Channel is kubecom's dev channel in a temporary directory.
type Channel struct {
	Root string // the directory served at URL
	URL  string
	Pub  *engram.Publisher
	key  sshsig.Key
	when time.Time
}

// New publishes nothing yet; the server stops when the test ends.
func New(t testing.TB) *Channel {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	srv := httptest.NewServer(http.FileServer(http.Dir(root)))
	t.Cleanup(srv.Close)
	c := &Channel{Root: root, URL: srv.URL, key: sshsig.Key(priv), when: time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)}
	c.Pub = &engram.Publisher{Store: store.Dir(root), Signer: c.key, Project: "kubecom", Channel: "dev"}
	return c
}

// Commit is a fake full commit made of one repeated hex digit.
func Commit(c byte) string { return strings.Repeat(string(c), 40) }

// Publish puts binary on the channel as this platform's kubecom, the way the
// release workflow does: a bare file named kubecom_<os>_<arch>.
func (c *Channel) Publish(t testing.TB, commit, version string, binary []byte) {
	t.Helper()
	local := filepath.Join(t.TempDir(), "kubecom_"+runtime.GOOS+"_"+runtime.GOARCH)
	if err := os.WriteFile(local, binary, 0o755); err != nil {
		t.Fatal(err)
	}
	c.when = c.when.Add(time.Hour)
	b := engram.Build{Commit: commit, Version: version, Time: c.when, MinEnboot: 1}
	up := []engram.Upload{{Name: "kubecom", OS: runtime.GOOS, Arch: runtime.GOARCH, Local: local}}
	if _, err := c.Pub.Publish(context.Background(), b, up); err != nil {
		t.Fatal(err)
	}
}

// Artifact is where this platform's kubecom of a build lies on disk, for a
// test that tampers with it.
func (c *Channel) Artifact(commit string) string {
	return filepath.Join(c.Root, "kubecom", "dev", "builds", commit, "kubecom_"+runtime.GOOS+"_"+runtime.GOARCH)
}

// Signers is the allowed_signers line of the channel's key.
func (c *Channel) Signers() string {
	return sshsig.AllowedSigners("test@kubecom", engram.Namespace, c.key.Public())
}

// Client reads the channel believing only its key.
func (c *Channel) Client() *engram.Client {
	return &engram.Client{Base: c.URL, Project: "kubecom", Channel: "dev", Keys: []ed25519.PublicKey{c.key.Public()}}
}
