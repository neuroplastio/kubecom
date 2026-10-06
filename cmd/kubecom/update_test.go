package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestUpdateTakesAnOptionalCommit(t *testing.T) {
	var got []string
	cmd := newUpdateCmd(func(_ context.Context, out io.Writer, commit string) error {
		got = append(got, commit)
		return nil
	})
	for _, args := range [][]string{{}, {"abc1234"}} {
		cmd.SetArgs(args)
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
	}
	if strings.Join(got, ",") != ",abc1234" {
		t.Errorf("update got commits %q", got)
	}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"a", "b"})
	if err := cmd.Execute(); err == nil {
		t.Error("two commits were accepted")
	}
	cmd = newUpdateCmd(func(context.Context, io.Writer, string) error {
		return errors.New("the dev channel holds no build")
	})
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	out.Reset()
	if err := cmd.Execute(); err == nil || strings.Contains(out.String(), "Usage:") {
		t.Errorf("a failed update: err %v, printed usage:\n%s", err, out.String())
	}
}
