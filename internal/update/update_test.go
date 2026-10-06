package update

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neuroplastio/engram"
	"github.com/neuroplastio/kubecom/internal/channel"
	"github.com/neuroplastio/kubecom/internal/update/testchannel"
	"github.com/neuroplastio/kubecom/internal/version"
)

var (
	commitA = testchannel.Commit('a')
	commitB = testchannel.Commit('b')
)

// twoBuilds is a channel holding a, then b, newest last.
func twoBuilds(t *testing.T) *testchannel.Channel {
	t.Helper()
	ch := testchannel.New(t)
	ch.Publish(t, commitA, "26.10.01-dev.aaaaaaa", []byte("kubecom build a"))
	ch.Publish(t, commitB, "26.10.02-dev.bbbbbbb", []byte("kubecom build b"))
	return ch
}

// onDev is a build of commit as the dev channel published it.
func onDev(commit, v string) version.Build {
	return version.Build{Version: v, Commit: commit, Channel: "dev"}
}

// installed is a kubecom binary in a directory of its own.
func installed(t *testing.T, content string) string {
	t.Helper()
	exe := filepath.Join(t.TempDir(), "kubecom")
	if err := os.WriteFile(exe, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return exe
}

func run(t *testing.T, ch *testchannel.Channel, self version.Build, exe, commit string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := Run(context.Background(), Options{Client: ch.Client(), Self: self, Exe: exe, Commit: commit, Out: &out})
	return out.String(), err
}

func content(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// leftovers fails the test if an update left a temporary file beside exe.
func leftovers(t *testing.T, exe string) {
	t.Helper()
	entries, _ := os.ReadDir(filepath.Dir(exe))
	for _, e := range entries {
		if e.Name() != filepath.Base(exe) {
			t.Errorf("left %s beside the binary", e.Name())
		}
	}
}

func TestAStaleBuildReplacesItselfWithTheNewest(t *testing.T) {
	ch := twoBuilds(t)
	exe := installed(t, "kubecom build a")
	out, err := run(t, ch, onDev(commitA, "26.10.01-dev.aaaaaaa"), exe, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := content(t, exe); got != "kubecom build b" {
		t.Errorf("binary is %q, want build b", got)
	}
	if fi, _ := os.Stat(exe); fi.Mode().Perm() != 0o755 {
		t.Errorf("mode %v, want 0755", fi.Mode().Perm())
	}
	if !strings.Contains(out, "kubecom 26.10.01-dev.aaaaaaa → 26.10.02-dev.bbbbbbb\n") {
		t.Errorf("output:\n%s", out)
	}
	leftovers(t, exe)
}

// The binary is checked against the sha256 the signed manifest names; one that
// does not match is never written.
func TestATamperedBinaryIsRefused(t *testing.T) {
	ch := twoBuilds(t)
	if err := os.WriteFile(ch.Artifact(commitB), []byte("kubecom build X"), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := installed(t, "kubecom build a")
	_, err := run(t, ch, onDev(commitA, "26.10.01-dev.aaaaaaa"), exe, "")
	if err == nil || !strings.Contains(err.Error(), "sha256") {
		t.Fatalf("a tampered binary: err = %v", err)
	}
	if got := content(t, exe); got != "kubecom build a" {
		t.Errorf("binary was replaced by %q", got)
	}
	leftovers(t, exe)
}

// A manifest signed by any key but the pinned one is not believed.
func TestABuildSignedByAnotherKeyIsRefused(t *testing.T) {
	ch := twoBuilds(t)
	other := testchannel.New(t)
	exe := installed(t, "kubecom build a")
	var out bytes.Buffer
	c := ch.Client()
	c.Keys = other.Client().Keys
	err := Run(context.Background(), Options{Client: c, Self: version.Build{Version: "local"}, Exe: exe, Out: &out})
	if err == nil {
		t.Fatal("a build signed by an unpinned key was installed")
	}
	if got := content(t, exe); got != "kubecom build a" {
		t.Errorf("binary was replaced by %q", got)
	}
}

func TestTheNewestBuildDoesNothing(t *testing.T) {
	ch := twoBuilds(t)
	exe := installed(t, "kubecom build b")
	out, err := run(t, ch, onDev(commitB, "26.10.02-dev.bbbbbbb"), exe, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "already the newest") {
		t.Errorf("output:\n%s", out)
	}
	// A local build of the same commit is the same code.
	out, err = run(t, ch, version.Build{Version: "bbbbbbb", Commit: commitB}, exe, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "kubecom bbbbbbb is already the newest build on dev") {
		t.Errorf("clean local build of the newest commit:\n%s", out)
	}
	if got := content(t, exe); got != "kubecom build b" {
		t.Errorf("binary changed to %q", got)
	}
}

// A local build — even a modified tree of the newest commit — takes the
// channel's newest, and says it replaced a local build.
func TestALocalBuildUpdatesToTheNewest(t *testing.T) {
	ch := twoBuilds(t)
	exe := installed(t, "my own kubecom")
	out, err := run(t, ch, version.Build{Version: "bbbbbbb+dirty", Commit: commitB, Modified: true}, exe, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := content(t, exe); got != "kubecom build b" {
		t.Errorf("binary is %q, want build b", got)
	}
	if !strings.Contains(out, "kubecom bbbbbbb+dirty (local build) → 26.10.02-dev.bbbbbbb") || !strings.Contains(out, "replaced "+exe) {
		t.Errorf("output:\n%s", out)
	}
}

// A commit by name installs that build, older ones included: going back is
// allowed when it is asked for.
func TestANamedCommit(t *testing.T) {
	ch := twoBuilds(t)
	exe := installed(t, "kubecom build b")
	if _, err := run(t, ch, onDev(commitB, "26.10.02-dev.bbbbbbb"), exe, "aaaa"); err != nil {
		t.Fatal(err)
	}
	if got := content(t, exe); got != "kubecom build a" {
		t.Errorf("binary is %q, want build a", got)
	}
	out, err := run(t, ch, onDev(commitA, "26.10.01-dev.aaaaaaa"), exe, commitA)
	if err != nil || !strings.Contains(out, "already aaaaaaa") {
		t.Errorf("asked for the commit it is: %q %v", out, err)
	}
	if _, err := run(t, ch, onDev(commitA, "x"), exe, "ffff"); err == nil || !strings.Contains(err.Error(), "no live build") {
		t.Errorf("unknown prefix: %v", err)
	}
	if _, err := run(t, ch, onDev(commitA, "x"), exe, testchannel.Commit('f')); err == nil || !strings.Contains(err.Error(), "not a live build") {
		t.Errorf("unknown commit: %v", err)
	}
	if _, err := run(t, ch, onDev(commitA, "x"), exe, "main"); err == nil || !strings.Contains(err.Error(), "not a commit") {
		t.Errorf("not a commit: %v", err)
	}
}

// A channel build knows its place in the channel, and refuses a head that
// points further back — a withdrawn build, or a channel rolled back by
// whoever controls the storage.
func TestNeverBackwardsUnlessAsked(t *testing.T) {
	ch := twoBuilds(t)
	if err := ch.Pub.Scrap(context.Background(), commitB, engram.ReasonBroken); err != nil {
		t.Fatal(err)
	}
	ch.Publish(t, testchannel.Commit('c'), "26.10.03-dev.ccccccc", []byte("kubecom build c"))
	// Rewind the head to a by hand, as a hostile store could: its publish
	// record becomes a's, byte for byte as the journal has it.
	dir := filepath.Join(ch.Root, "kubecom", "dev")
	var publishA string
	for _, l := range strings.Split(string(mustRead(t, filepath.Join(dir, "journal"))), "\n") {
		if strings.HasPrefix(l, "publish ") && strings.Contains(l, "commit="+commitA) {
			publishA = l
		}
	}
	head := strings.Split(string(mustRead(t, filepath.Join(dir, "head"))), "\n")
	for i, l := range head {
		if strings.HasPrefix(l, "publish ") {
			head[i] = publishA
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "head"), []byte(strings.Join(head, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	exe := installed(t, "kubecom build c")
	_, err := run(t, ch, onDev(testchannel.Commit('c'), "26.10.03-dev.ccccccc"), exe, "")
	if err == nil || !strings.Contains(err.Error(), "older build") {
		t.Fatalf("rolled-back head: %v", err)
	}
	if got := content(t, exe); got != "kubecom build c" {
		t.Errorf("binary went back to %q", got)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// The link stays a link; the binary it names is the one replaced.
func TestASymlinkIsFollowed(t *testing.T) {
	ch := twoBuilds(t)
	exe := installed(t, "kubecom build a")
	link := filepath.Join(t.TempDir(), "kubecom")
	if err := os.Symlink(exe, link); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, ch, onDev(commitA, "26.10.01-dev.aaaaaaa"), link, ""); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Lstat(link); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("the link was replaced: %v %v", fi, err)
	}
	if got := content(t, exe); got != "kubecom build b" {
		t.Errorf("the binary behind the link is %q", got)
	}
}

func TestAnUnwritableDirectoryIsAClearError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes anywhere")
	}
	ch := twoBuilds(t)
	exe := installed(t, "kubecom build a")
	dir := filepath.Dir(exe)
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })
	_, err := run(t, ch, onDev(commitA, "26.10.01-dev.aaaaaaa"), exe, "")
	if err == nil || !strings.Contains(err.Error(), "is not writable by this user") {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(err.Error(), "sudo") {
		t.Errorf("the error suggests sudo: %v", err)
	}
	if got := content(t, exe); got != "kubecom build a" {
		t.Errorf("binary is %q", got)
	}
}

func TestClientFollowsTheBuildsChannel(t *testing.T) {
	c, err := Client(version.Build{})
	if err != nil {
		t.Fatal(err)
	}
	if c.Base != channel.URL || c.Project != "kubecom" || c.Channel != "dev" || len(c.Keys) != 1 {
		t.Errorf("a local build's client: %+v", c)
	}
	if c, _ := Client(version.Build{Channel: "stable"}); c.Channel != "stable" {
		t.Errorf("a stable build's client: %+v", c)
	}
}

// A kubecom a launcher started leaves its own file alone: the build goes into
// the launcher's home, and bin moves to it for the next launch.
func TestUnderALauncherTheBuildGoesIntoItsHome(t *testing.T) {
	ch := twoBuilds(t)
	exe := installed(t, "kubecom build a")
	home := filepath.Join(t.TempDir(), "kubecom")
	var out bytes.Buffer
	err := Run(context.Background(), Options{Client: ch.Client(), Self: onDev(commitA, "26.10.01-dev.aaaaaaa"), Exe: exe, Home: home, Out: &out})
	if err != nil {
		t.Fatal(err)
	}
	if got := content(t, exe); got != "kubecom build a" {
		t.Errorf("the running file was replaced: %q", got)
	}
	if got := content(t, filepath.Join(home, "bin", "kubecom")); got != "kubecom build b" {
		t.Errorf("the home's kubecom is %q", got)
	}
	if !strings.Contains(out.String(), "kubecom 26.10.01-dev.aaaaaaa → 26.10.02-dev.bbbbbbb\ninstalled in "+home) {
		t.Errorf("said:\n%s", out.String())
	}
}
