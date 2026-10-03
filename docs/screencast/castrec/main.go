// Castrec records the screencast as an asciicast (v3) in the same run that
// records the GIF (CAST-01, D290). vhs has no cast output, and a recording of
// the whole terminal would hold the tour's tmux status line and the setup it
// types while hidden. castrec records only the pane kubecom runs in, and keeps
// the tour's captions as markers, for a player to show as it likes.
//
// tmux runs both halves, from the tape:
//
//	pipe-pane -o 'castrec rec #{socket_path} #{pane_id}'
//	set -g status-left '…' ; run-shell 'castrec mark #{q:status-left}'
//
// rec reads what the pane's program writes, as tmux pipes it, and starts the
// cast with the pane as it stood when the pipe opened, so the prompt drawn
// before it is in the cast. mark notes a caption and the time it was set.
// When the pipe closes (`pipe-pane` with no command), rec writes the cast with
// the captions at their times.
//
// Both run in the tmux server's working directory, the repository's root when
// `make screencast` runs the tape. mark prints nothing and always succeeds:
// tmux would show its output, or its failure, over the recording.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// castPath is where the cast goes, beside the GIF; marks are kept beside it
// until rec merges them in.
const castPath = "docs/screencast.cast"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: castrec rec <tmux socket> <pane> | castrec mark <caption…>")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "rec":
		if len(os.Args) != 4 {
			fmt.Fprintln(os.Stderr, "usage: castrec rec <tmux socket> <pane>")
			os.Exit(2)
		}
		pane, err := tmuxPane(os.Args[2], os.Args[3])
		if err == nil {
			err = record(castPath, pane, os.Stdin, time.Now)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "castrec:", err)
			os.Exit(1)
		}
	case "mark":
		_ = mark(castPath+".marks", strings.Join(os.Args[2:], " "), time.Now())
	default:
		fmt.Fprintf(os.Stderr, "castrec: unknown command %q\n", os.Args[1])
		os.Exit(2)
	}
}

// pane is the pane as it stood when the recording began.
type pane struct {
	cols, rows int
	term       string
	screen     string // its cells, with their SGR, a line per row
	x, y       int    // its cursor
}

// tmuxPane asks tmux for the pane's size, terminal and screen.
func tmuxPane(socket, id string) (pane, error) {
	out, err := exec.Command("tmux", "-S", socket, "display-message", "-p", "-t", id,
		"#{pane_width} #{pane_height} #{cursor_x} #{cursor_y} #{default-terminal}").Output()
	if err != nil {
		return pane{}, fmt.Errorf("tmux display-message: %w", err)
	}
	var p pane
	if _, err := fmt.Sscan(string(out), &p.cols, &p.rows, &p.x, &p.y, &p.term); err != nil {
		return pane{}, fmt.Errorf("tmux display-message %q: %w", out, err)
	}
	screen, err := exec.Command("tmux", "-S", socket, "capture-pane", "-p", "-e", "-t", id).Output()
	if err != nil {
		return pane{}, fmt.Errorf("tmux capture-pane: %w", err)
	}
	p.screen = string(screen)
	return p, nil
}

// event is one line of the cast: output ("o") or a marker ("m"), at a time.
type event struct {
	at   time.Time
	code string
	data string
}

// record reads the pane's output until the pipe closes, then writes the cast
// to path, with the marks kept at path+".marks" since the recording began.
func record(path string, p pane, in io.Reader, now func() time.Time) error {
	start := now()
	evs := []event{{start, "o", opening(p)}}
	var carry []byte // the start of a character the last read cut in two
	buf := make([]byte, 32<<10)
	for {
		n, err := in.Read(buf)
		if n > 0 {
			b := append(carry, buf[:n]...)
			b, carry = splitIncomplete(b)
			if len(b) > 0 {
				evs = append(evs, event{now(), "o", string(b)})
			}
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
	}
	if len(carry) > 0 {
		evs = append(evs, event{now(), "o", string(carry)})
	}
	marks, err := readMarks(path+".marks", start)
	if err != nil {
		return err
	}
	cast, err := encode(p, start, append(evs, marks...))
	if err != nil {
		return err
	}
	if err := os.WriteFile(path+".tmp", cast, 0o644); err != nil {
		return err
	}
	if err := os.Rename(path+".tmp", path); err != nil {
		return err
	}
	if err := os.Remove(path + ".marks"); !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// opening draws the pane as it stood: its cells, then its cursor.
func opening(p pane) string {
	lines := strings.Split(strings.TrimSuffix(p.screen, "\n"), "\n")
	return "\x1b[H\x1b[2J" + strings.Join(lines, "\r\n") + "\x1b[0m" +
		"\x1b[" + strconv.Itoa(p.y+1) + ";" + strconv.Itoa(p.x+1) + "H"
}

// splitIncomplete holds back a character the end of b cuts in two, for the
// next read: a cast's output is text, and half a character is not.
func splitIncomplete(b []byte) (whole, rest []byte) {
	for i := len(b) - 1; i >= 0 && i >= len(b)-utf8.UTFMax; i-- {
		if utf8.RuneStart(b[i]) {
			if !utf8.FullRune(b[i:]) {
				return b[:i], append([]byte(nil), b[i:]...)
			}
			break
		}
	}
	return b, nil
}

// markLine is a mark as mark keeps it, one JSON object per line.
type markLine struct {
	T     int64  `json:"t"` // Unix nanoseconds
	Label string `json:"label"`
}

// tmuxStyle is tmux's inline style, #[fg=cyan,bold], which a caption set as
// the status line carries.
var tmuxStyle = regexp.MustCompile(`#\[[^\]]*\]`)

// mark appends a caption to the marks file, as tmux's status line shows it:
// without its style and the padding around it.
func mark(path, caption string, at time.Time) error {
	label := strings.TrimSpace(tmuxStyle.ReplaceAllString(caption, ""))
	if label == "" {
		return nil
	}
	line, err := json.Marshal(markLine{at.UnixNano(), label})
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, err = f.Write(append(line, '\n'))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

// readMarks reads the marks made since start: a file a broken run left
// behind holds older ones.
func readMarks(path string, start time.Time) ([]event, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var evs []event
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		var m markLine
		if json.Unmarshal(sc.Bytes(), &m) != nil {
			continue
		}
		if at := time.Unix(0, m.T); !at.Before(start) {
			evs = append(evs, event{at, "m", m.Label})
		}
	}
	return evs, sc.Err()
}

// encode writes the cast: the header, then each event at its interval from
// the one before (asciicast v3), output before a marker made at the same
// moment.
func encode(p pane, start time.Time, evs []event) ([]byte, error) {
	sort.SliceStable(evs, func(i, j int) bool { return evs[i].at.Before(evs[j].at) })
	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	header := map[string]any{
		"version":   3,
		"term":      map[string]any{"cols": p.cols, "rows": p.rows, "type": p.term},
		"timestamp": start.Unix(),
		"title":     "kubecom",
	}
	if err := enc.Encode(header); err != nil {
		return nil, err
	}
	last := start
	for _, ev := range evs {
		var dt time.Duration
		if ev.at.After(last) {
			dt, last = ev.at.Sub(last), ev.at
		}
		data, err := json.Marshal(ev.data)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&out, "[%s, %q, %s]\n", strconv.FormatFloat(dt.Seconds(), 'f', 6, 64), ev.code, data)
	}
	return out.Bytes(), nil
}
