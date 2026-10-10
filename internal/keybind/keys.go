package keybind

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Key is one keystroke as the keymap compares it: modifiers and a base, either
// one character ("j", "G", "+") or a named key ("enter", "f1", "space"). It is
// canonical by construction — ParseKey and a front end's adapter fold the ways one
// keystroke can be spelled into one Key — so two Keys are the same keystroke
// exactly when they are ==, and a Key is a map key.
//
// Shift is kept only on a named key (shift+tab): a shifted letter is its capital
// (`shift+h` is `H`), because that is how a terminal reports it on every path.
type Key struct {
	Ctrl, Alt, Shift bool
	Name             string
}

// namedKeys are the keys a sequence writes by name, kubecom's set. There is one
// spelling per key (no `escape` beside `esc`).
var namedKeys = map[string]bool{
	"up": true, "down": true, "left": true, "right": true,
	"enter": true, "esc": true, "tab": true, "space": true,
	"home": true, "end": true, "pgup": true, "pgdn": true,
	"backspace": true, "delete": true, "insert": true,
	"f1": true, "f2": true, "f3": true, "f4": true, "f5": true, "f6": true,
	"f7": true, "f8": true, "f9": true, "f10": true, "f11": true, "f12": true,
}

// Named reports whether name is a key written by name (`enter`, `pgdn`, `f5`).
func Named(name string) bool { return namedKeys[name] }

// String writes the key the way ParseKey reads it, modifiers in a fixed order:
// `alt+ctrl+x`, `shift+tab`, `G`.
func (k Key) String() string {
	var b strings.Builder
	if k.Alt {
		b.WriteString("alt+")
	}
	if k.Ctrl {
		b.WriteString("ctrl+")
	}
	if k.Shift {
		b.WriteString("shift+")
	}
	b.WriteString(k.Name)
	return b.String()
}

// Text reports whether the key types text into a field: one character or space,
// with no ctrl or alt. It is the test the field rule (D303 pt 5) applies.
func (k Key) Text() bool {
	if k.Ctrl || k.Alt {
		return false
	}
	return k.Name == "space" || utf8.RuneCountInString(k.Name) == 1
}

// ParseKey reads one key: modifiers (`ctrl+`, `alt+`, `shift+`) and a base, one
// character or a named key. Case folds as the terminal does: `ctrl+D` is `ctrl+d`,
// `shift+h` is `H`. A shifted character other than a letter is written as the
// character it types (`!`, not `shift+1`), and ctrl+shift on a letter is refused,
// since most terminals send it as the plain ctrl chord.
func ParseKey(s string) (Key, error) {
	var k Key
	rest := s
	for {
		lower := strings.ToLower(rest)
		switch {
		case len(rest) > len("ctrl+") && strings.HasPrefix(lower, "ctrl+"):
			k.Ctrl, rest = true, rest[len("ctrl+"):]
			continue
		case len(rest) > len("alt+") && strings.HasPrefix(lower, "alt+"):
			k.Alt, rest = true, rest[len("alt+"):]
			continue
		case len(rest) > len("shift+") && strings.HasPrefix(lower, "shift+"):
			k.Shift, rest = true, rest[len("shift+"):]
			continue
		}
		break
	}
	if rest == "" {
		return Key{}, fmt.Errorf("%q is not a key", s)
	}
	if name := strings.ToLower(rest); namedKeys[name] {
		k.Name = name
		return k, nil
	}
	r, n := utf8.DecodeRuneInString(rest)
	if n != len(rest) || r == utf8.RuneError {
		if strings.Contains(rest, "+") {
			return Key{}, fmt.Errorf("%q is not a key (modifiers are ctrl+, alt+ and shift+)", s)
		}
		if k.Ctrl || k.Alt || k.Shift {
			return Key{}, fmt.Errorf("%q is not a key", s)
		}
		return Key{}, fmt.Errorf("%q is not a key (a sequence's keys are separated by spaces: `g g`)", s)
	}
	if r == ' ' {
		k.Name = "space"
		return k, nil
	}
	if k.Shift {
		switch {
		case !unicode.IsLetter(r):
			return Key{}, fmt.Errorf("%q: write the character shift types, not shift+%c", s, r)
		case k.Ctrl:
			return Key{}, fmt.Errorf("%q: terminals send ctrl+shift+%c as ctrl+%c", s, unicode.ToLower(r), unicode.ToLower(r))
		}
		r, k.Shift = unicode.ToUpper(r), false
	}
	if k.Ctrl {
		r = unicode.ToLower(r)
	}
	k.Name = string(r)
	return k, nil
}

// leaderToken is how a sequence writes the leader alias.
const leaderToken = "<leader>"

// step is one key of a sequence as written: a key, or the leader alias, which
// stands for whatever key the keymap's `leader` names when it is built.
type step struct {
	leader bool
	key    Key
}

// sequence is a key sequence as written, before the leader is resolved.
type sequence []step

// parseSeq reads a sequence: keys separated by spaces (`g g`, `ctrl+x d`), each
// read by ParseKey, or `<leader>`.
func parseSeq(s string) (sequence, error) {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return nil, fmt.Errorf("an empty key sequence")
	}
	out := make(sequence, 0, len(fields))
	for _, f := range fields {
		if f == leaderToken {
			out = append(out, step{leader: true})
			continue
		}
		k, err := ParseKey(f)
		if err != nil {
			return nil, err
		}
		out = append(out, step{key: k})
	}
	return out, nil
}

// ParseKeys reads a sequence with no leader in it into its keys, for a caller
// that has keys as text — a trace's tokens, a test's.
func ParseKeys(s string) ([]Key, error) {
	seq, err := parseSeq(s)
	if err != nil {
		return nil, err
	}
	out := make([]Key, len(seq))
	for i, st := range seq {
		if st.leader {
			return nil, fmt.Errorf("%q: %s is the keymap's to resolve", s, leaderToken)
		}
		out[i] = st.key
	}
	return out, nil
}

// String writes the sequence the way parseSeq reads it.
func (s sequence) String() string {
	parts := make([]string, len(s))
	for i, st := range s {
		if st.leader {
			parts[i] = leaderToken
		} else {
			parts[i] = st.key.String()
		}
	}
	return strings.Join(parts, " ")
}

// FormatKeys writes resolved keys the way a sequence is written: `g g`.
func FormatKeys(keys []Key) string {
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k.String()
	}
	return strings.Join(parts, " ")
}
