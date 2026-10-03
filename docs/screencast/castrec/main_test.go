package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// chunks is the pane's output as tmux pipes it: a read at a time, each read
// a moment after the one before.
type chunks struct {
	reads []string
	clock *clock
}

func (c *chunks) Read(p []byte) (int, error) {
	if len(c.reads) == 0 {
		return 0, io.EOF
	}
	n := copy(p, c.reads[0])
	c.reads = c.reads[1:]
	c.clock.tick(100 * time.Millisecond)
	return n, nil
}

type clock struct{ t time.Time }

func (c *clock) now() time.Time       { return c.t }
func (c *clock) tick(d time.Duration) { c.t = c.t.Add(d) }

// castLines decodes a cast into its header and its events.
func castLines(t *testing.T, data []byte) (map[string]any, [][]any) {
	t.Helper()
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	var header map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &header); err != nil {
		t.Fatalf("header %q: %v", lines[0], err)
	}
	var evs [][]any
	for _, l := range lines[1:] {
		var ev []any
		if err := json.Unmarshal([]byte(l), &ev); err != nil {
			t.Fatalf("event %q: %v", l, err)
		}
		evs = append(evs, ev)
	}
	return header, evs
}

func TestRecordMergesTheCaptionsAtTheirTimes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "screencast.cast")
	c := &clock{time.Unix(1_800_000_000, 0)}
	start := c.now()

	// A mark a broken run left behind, and two from this one: the first
	// between the first and second reads, the second after the last.
	for _, m := range []struct {
		at      time.Time
		caption string
	}{
		{start.Add(-time.Hour), "   #[fg=cyan,bold]Stale"},
		{start.Add(150 * time.Millisecond), "   #[fg=cyan,bold]Filter live resources as you type"},
		{start.Add(time.Second), "   #[fg=cyan,bold]Thanks for watching"},
	} {
		if err := mark(path+".marks", m.caption, m.at); err != nil {
			t.Fatal(err)
		}
	}
	p := pane{cols: 80, rows: 24, term: "xterm-256color", screen: "\x1b[36m$\x1b[39m \n\n", x: 2, y: 0}
	in := &chunks{reads: []string{"kubecom", "\r\n\x1b[?1049h"}, clock: c}
	if err := record(path, p, in, c.now); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	header, evs := castLines(t, data)
	if header["version"] != float64(3) || header["title"] != "kubecom" {
		t.Errorf("header %v", header)
	}
	if term := header["term"].(map[string]any); term["cols"] != float64(80) || term["rows"] != float64(24) || term["type"] != "xterm-256color" {
		t.Errorf("term %v", term)
	}
	want := [][]any{
		{0.0, "o", "\x1b[H\x1b[2J\x1b[36m$\x1b[39m \r\n\x1b[0m\x1b[1;3H"},
		{0.1, "o", "kubecom"},
		{0.05, "m", "Filter live resources as you type"},
		{0.05, "o", "\r\n\x1b[?1049h"},
		{0.8, "m", "Thanks for watching"},
	}
	if len(evs) != len(want) {
		t.Fatalf("events %v, want %v", evs, want)
	}
	for i := range want {
		for j := range want[i] {
			if evs[i][j] != want[i][j] {
				t.Errorf("event %d is %v, want %v", i, evs[i], want[i])
				break
			}
		}
	}
	if _, err := os.Stat(path + ".marks"); !os.IsNotExist(err) {
		t.Errorf("the marks file outlived the recording: %v", err)
	}
}

// A read that ends inside a character keeps its start for the next read: a
// cast's output is JSON text, and half a character would be a U+FFFD.
func TestRecordKeepsACharacterAReadCutInTwo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "screencast.cast")
	c := &clock{time.Unix(1_800_000_000, 0)}
	check := "✓" // three bytes
	in := &chunks{reads: []string{"ok " + check[:1], check[1:] + " done"}, clock: c}
	if err := record(path, pane{cols: 10, rows: 2, term: "xterm"}, in, c.now); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_, evs := castLines(t, data)
	var out []string
	for _, ev := range evs[1:] {
		out = append(out, ev[2].(string))
	}
	if got := strings.Join(out, "|"); got != "ok |✓ done" {
		t.Errorf("output %q, want %q", got, "ok |✓ done")
	}
}

func TestMarkKeepsTheCaptionAsTheStatusLineShowsIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "marks")
	at := time.Unix(1_800_000_000, 5)
	for _, caption := range []string{"   #[fg=cyan,bold]Press ? for the full keymap", "  ", "#[default]"} {
		if err := mark(path, caption, at); err != nil {
			t.Fatal(err)
		}
	}
	evs, err := readMarks(path, at)
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 1 || evs[0].data != "Press ? for the full keymap" || !evs[0].at.Equal(at) {
		t.Errorf("marks %+v, want the one caption, unstyled, at its time", evs)
	}
}
