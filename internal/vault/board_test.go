package vault

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// boardPath is the live task board, read relative to this package (the pattern
// internal/version uses for `.goreleaser.yml`).
var boardPath = filepath.Join("..", "..", "vault", "tasks", "board.md")

// The board is the working area: In Progress, Blocked, Backlog and the per-line
// planning prose. Completed items are **dropped** rather than indexed (D291,
// superseding the Done-list half of D102/D224/D225); git history, the journal and
// the decision log are the record of finished work. Two conventions are still
// worth executing:
//
//   - a deferral must name where the work went (D226 pt 2), because a real,
//     unblocked item once sat in a closed line's prose while three consecutive
//     legs reported that nothing unblocked remained; and
//   - nothing on the board is marked done, so the cleanup cannot silently
//     regress into a second Done list.

// deferralPhrases are the ways the board has actually deferred work in prose. It
// is a backstop and not the spec (D226 pt 2, under D224 pt 3): the reply to a
// deferral phrased some other way is to name its destination like every other
// one, never to add the phrase here so the check keeps passing.
var deferralPhrases = []string{
	"deliberately left",
	"its own small item",
	"is a candidate to",
	"raise it as",
}

// boldDestination is the two written-down endings D226 pt 2 allows: a filed item
// id, or the decision a constraint was moved into. Both must be **bold**, and
// that is the load-bearing part of the pattern. A bare `Dnn` would not do —
// almost every paragraph on this board cites one, so accepting an unbolded
// reference would have passed all four of the paragraphs this test exists for
// (the CRD-PIN one names D202, the PAL one D209, the AUTH one D216). Bolding is
// what makes the reference a *destination* rather than a citation.
//
// `**no**` and `**shows**` — both real, both in the paragraph this was written
// against — are emphasis, so an id needs a hyphen and a decision needs its
// digits. A bold sentence contains spaces and matches neither.
var (
	boldDestination = regexp.MustCompile(`\*\*(?:[A-Z][A-Z0-9]*(?:-[A-Za-z0-9]+)+|D\d{1,4})\*\*`)
	untilAsked      = "no item until asked"
	blankLine       = regexp.MustCompile(`^\s*$`)
)

// TestBoardDeferralsNameTheirDestination is the executable half of D226 pt 2.
//
// The failure it guards is one an Orient cannot see. Four closed lines had ended
// with "two things it deliberately left, either its own small item if a dogfood
// wants them", and none of the seven things named was a `- [ ]`. So the board
// truthfully showed five open items — four of them blocked — while a real,
// unblocked one sat in a paragraph, and three consecutive legs reported that
// nothing unblocked remained.
//
// The paragraph, not the line, is the unit: these sentences wrap across four or
// five lines at the board's margin, and the id that answers them is routinely on
// a different line from the phrase that raises them.
func TestBoardDeferralsNameTheirDestination(t *testing.T) {
	for _, p := range prosePargraphs(t) {
		joined := strings.Join(strings.Fields(p.text), " ")
		phrase := ""
		for _, d := range deferralPhrases {
			if strings.Contains(joined, d) {
				phrase = d
				break
			}
		}
		if phrase == "" {
			continue
		}
		if boldDestination.MatchString(joined) || strings.Contains(joined, untilAsked) {
			continue
		}
		t.Errorf("board.md:%d defers work (%q) without naming where it went (D226 pt 2)\n"+
			"  want one of: a filed **ID**, a bold **Dnn**, or the words %q\n  got:  %s",
			p.line, phrase, untilAsked, truncate(joined, 160))
	}
}

// TestBoardDropsCompletedItems guards the cleanup convention itself (D291): the
// board holds open work only. A `- [x]` line is a second Done list trying to
// grow back; its place is the journal/commit/decision log. The check is cheap
// and catches the regression at the moment a leg writes one.
func TestBoardDropsCompletedItems(t *testing.T) {
	for _, l := range boardLines(t) {
		if strings.HasPrefix(strings.TrimSpace(l.text), "- [x] ") {
			t.Errorf("board.md:%d marks an item done — the board holds open work only; "+
				"completed items are dropped (D291)\n  %s", l.line, truncate(l.text, 120))
		}
	}
}

// prosePargraphs splits the board into blank-line-separated blocks, each tagged
// with the line its first line sits on.
func prosePargraphs(t *testing.T) []boardLine {
	t.Helper()
	var out []boardLine
	cur := boardLine{}
	flush := func() {
		if strings.TrimSpace(cur.text) != "" {
			out = append(out, cur)
		}
		cur = boardLine{}
	}
	for _, l := range boardLines(t) {
		if blankLine.MatchString(l.text) {
			flush()
			continue
		}
		if cur.line == 0 {
			cur.line = l.line
		}
		cur.text += l.text + "\n"
	}
	flush()
	return out
}

// boardLine is one line of the board with its 1-based number, so a failure names
// a place the reader can open.
type boardLine struct {
	line int
	text string
}

// boardLines is every line of the board.
func boardLines(t *testing.T) []boardLine {
	t.Helper()
	data, err := os.ReadFile(boardPath)
	if err != nil {
		t.Fatalf("read %s: %v", boardPath, err)
	}
	var out []boardLine
	for i, text := range strings.Split(string(data), "\n") {
		out = append(out, boardLine{line: i + 1, text: text})
	}
	return out
}

// truncate keeps a failure message readable when the offending line is the very
// thing being complained about.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
