package keymap

import (
	"sort"
	"strings"
	"testing"

	"github.com/neuroplastio/kubecom/internal/keybind"
)

// default.kdl loads, and every binding names a registered action with a value
// only where one is taken.
func TestDefaultKDLLoads(t *testing.T) {
	km := keybind.Build(DefaultDoc())
	if err := km.Err(); err != nil {
		t.Fatal(err)
	}
	bound := map[Action]bool{}
	for _, e := range km.Bindings() {
		bound[Action(e.Binding.Action)] = true
	}
	for _, a := range Actions() {
		if !bound[a] && !PaletteOnly(a) {
			t.Errorf("%s has no default binding", a)
		}
		if bound[a] && PaletteOnly(a) {
			t.Errorf("%s is palette-only but bound", a)
		}
	}
}

func TestCheckBinding(t *testing.T) {
	for _, tc := range []struct {
		action string
		args   []string
		want   string
	}{
		{"nav.down", nil, ""},
		{"ns.switch", []string{"kube-system"}, ""},
		{"resources.switch", []string{"pods"}, ""},
		{"nav.dwn", nil, `"nav.dwn" is not an action`},
		{"nav.down", []string{"1"}, "nav.down takes no value"},
		{"ns.switch", []string{"a", "b"}, "ns.switch takes one value, not 2"},
	} {
		err := CheckBinding(tc.action, tc.args)
		if (tc.want == "") != (err == nil) || (err != nil && !strings.Contains(err.Error(), tc.want)) {
			t.Errorf("%s %v: %v, want %q", tc.action, tc.args, err, tc.want)
		}
	}
}

// The port keeps every key (D303 pt 6): on every surface, each key the old flat
// map, its confirm and table side tables and the text-field routes resolved, the
// context keymap resolves to the same action — and nothing more. Each scenario is
// a path and the old resolver that served it: the browse map through the
// sequencer, the table's TableAction first, the confirm modal's ConfirmAction
// then the browse map (confirmResolved), and a text field's "mapped and types no
// text" (routeFilterKey, routeLogsFilterKey, routePickerKey, routeSearchKey,
// routeModalPromptKey).
func TestDefaultKDLKeepsTheOldKeys(t *testing.T) {
	old := DefaultKeymap()
	km := keybind.Build(DefaultDoc())

	type resolver func(c chord, text bool) (Action, bool)
	browse := func(c chord, _ bool) (Action, bool) { return old.exact(seq{c}) }
	field := func(c chord, text bool) (Action, bool) {
		if text {
			return "", false
		}
		return old.exact(seq{c})
	}
	table := func(c chord, _ bool) (Action, bool) {
		if a, ok := old.tableBySeq[seq{c}.key()]; ok {
			return a, true
		}
		return old.exact(seq{c})
	}
	confirm := func(c chord, _ bool) (Action, bool) {
		if a, ok := old.confirmBySeq[seq{c}.key()]; ok {
			return a, true
		}
		return old.exact(seq{c})
	}

	cluster := &keybind.Cluster{Context: "kind-dev", Namespace: "default"}
	pods := &keybind.Table{Resource: "pods", Kind: "Pod", Namespaced: true}
	scenarios := []struct {
		name string
		path keybind.Path
		old  resolver
		// seqs is whether the surface fed the sequencer (multi-key sequences).
		seqs bool
	}{
		{"menu", keybind.Path{Cluster: cluster, Menu: true}, browse, true},
		{"menu filter", keybind.Path{Cluster: cluster, Menu: true, Field: true}, field, false},
		{"table", keybind.Path{Cluster: cluster, Table: pods}, table, true},
		{"table filter", keybind.Path{Cluster: cluster, Table: pods, Field: true}, field, false},
		{"sort", keybind.Path{Cluster: cluster, Table: pods, Sort: true}, browse, true},
		{"logs", keybind.Path{Cluster: cluster, Table: pods, Logs: &keybind.Logs{Follow: true}}, browse, true},
		{"logs grep", keybind.Path{Cluster: cluster, Table: pods, Logs: &keybind.Logs{}, Field: true}, field, false},
		{"viewer", keybind.Path{Cluster: cluster, Table: pods, Viewer: &keybind.Viewer{Content: "secret"}}, browse, true},
		{"search results", keybind.Path{Cluster: cluster, Table: pods, Search: &keybind.Search{}}, browse, false},
		{"search query", keybind.Path{Cluster: cluster, Menu: true, Search: &keybind.Search{}, Field: true}, field, false},
		{"unhealthy", keybind.Path{Cluster: cluster, Menu: true, Unhealthy: true}, browse, true},
		{"forwards", keybind.Path{Cluster: cluster, Table: pods, Forwards: true}, browse, true},
		{"help", keybind.Path{Cluster: cluster, Menu: true, Help: true}, browse, true},
		{"picker", keybind.Path{Cluster: cluster, Table: pods, Picker: &keybind.Picker{ID: "ports"}}, browse, false},
		{"picker query", keybind.Path{Cluster: cluster, Table: pods, Picker: &keybind.Picker{ID: "palette"}, Field: true}, field, false},
		{"prompt", keybind.Path{Cluster: cluster, Table: pods, Prompt: true, Field: true}, field, false},
		{"confirm", keybind.Path{Cluster: cluster, Table: pods, Confirm: true}, confirm, false},
		{"no cluster", keybind.Path{}, browse, true},
	}

	for _, sc := range scenarios {
		for _, tok := range keyUniverse(old) {
			k, err := keybind.ParseKey(tok)
			if err != nil {
				t.Fatalf("%s: %v", tok, err)
			}
			// A key the old syntax could not write (`+`) was bound to nothing.
			var want Action
			if c, err := parseChord(tok); err == nil {
				want, _ = sc.old(c, k.Text())
			}
			var got Action
			if m := km.Match([]keybind.Key{k}, sc.path); m.Exact != nil {
				got = Action(m.Exact.Action)
			}
			if got != want {
				t.Errorf("%s: %s resolves to %q, the old keymap to %q", sc.name, tok, got, want)
			}
		}
		if !sc.seqs {
			continue
		}
		for key, a := range old.bySeq {
			s := strings.Split(key, "\x00")
			if len(s) < 2 {
				continue
			}
			ks, err := keybind.ParseKeys(strings.Join(s, " "))
			if err != nil {
				t.Fatal(err)
			}
			if m := km.Match(ks, sc.path); m.Exact == nil || Action(m.Exact.Action) != a || m.Prefix {
				t.Errorf("%s: %s is %+v, the old keymap's %s", sc.name, keybind.FormatKeys(ks), m, a)
			}
			if m := km.Match(ks[:1], sc.path); m.Prefix != old.hasExtension(seq{chord(s[0])}) {
				t.Errorf("%s: %s as a prefix: %v", sc.name, s[0], m.Prefix)
			}
		}
	}
}

// keyUniverse is every key the old maps bind, plus every printable ASCII
// character, every named key and every ctrl+letter — so "nothing more" is tested
// too.
func keyUniverse(km *Keymap) []string {
	seen := map[string]bool{}
	for _, idx := range []map[string]Action{km.bySeq, km.confirmBySeq, km.tableBySeq} {
		for key := range idx {
			if !strings.Contains(key, "\x00") {
				seen[key] = true
			}
		}
	}
	for r := '!'; r <= '~'; r++ {
		seen[string(r)] = true
	}
	for name := range specialNames {
		seen[name] = true
	}
	for r := 'a'; r <= 'z'; r++ {
		seen["ctrl+"+string(r)] = true
		seen["alt+"+string(r)] = true
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
