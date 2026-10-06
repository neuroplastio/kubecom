//go:build !updatetest

package channel

import "testing"

// Only a binary built with -tags updatetest can be pointed at another key or
// server. This test is compiled the way every release is, without the tag,
// and proves the variables that would move them do nothing there — in
// kubecom's update and in kubecom-launcher alike, which both read them here.
func TestTheKeyAndTheServerCannotBeMovedInARealBuild(t *testing.T) {
	t.Setenv("KUBECOM_UPDATE_SIGNERS", `test@test namespaces="engram" ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdead`)
	t.Setenv("KUBECOM_PKG_URL", "http://127.0.0.1:1")
	if l := Launcher(""); l.Base != URL || l.Signers != releaseSigners {
		t.Errorf("the channel is read from %s with %q", l.Base, l.Signers)
	}
	if l := Launcher(""); l.Channel != Default || l.Project != Project {
		t.Errorf("the channel is %q/%q", l.Project, l.Channel)
	}
}
