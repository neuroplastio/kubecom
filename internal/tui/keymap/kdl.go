package keymap

import (
	_ "embed"
	"fmt"
	"sync"

	"github.com/neuroplastio/kubecom/internal/keybind"
)

// This file is the action registry's side of the context keymap (D303): the
// embedded default.kdl as the default layer, and the registry's check of a
// document's bindings — that each names a registered action, and gives a value
// only to an action that takes one.

// defaultFile is what default.kdl's lines are called in errors and listings.
const defaultFile = "default.kdl"

//go:embed default.kdl
var defaultSource []byte

// DefaultSource is default.kdl as embedded — what `kubecom keys default` prints,
// to read or to start a `base "none"` keymap from.
func DefaultSource() []byte { return defaultSource }

// DefaultDoc is the default keymap as the default layer. default.kdl is embedded
// and tested, so an error in it is a broken build, not a user's mistake.
func DefaultDoc() keybind.Doc { return defaultDoc() }

var defaultDoc = sync.OnceValue(func() keybind.Doc {
	d, err := keybind.ParseFile(defaultFile, defaultSource, keybind.Default)
	if err == nil {
		err = d.Check(CheckBinding)
	}
	if err != nil {
		panic(fmt.Sprintf("keymap: %v", err))
	}
	return d
})

// argumentActions are the palette's argument verbs: an action that takes a value
// (the namespace, the kind, the theme…). Bound with no value its key opens the
// palette's stage for it; bound with one it runs with that value (D303 pt 2).
var argumentActions = map[Action]struct{}{
	ActionResources: {},
	ActionTheme:     {},
	ActionNamespace: {},
	ActionContext:   {},
	ActionPin:       {},
	ActionActions:   {},
}

// TakesArgument reports whether a is an argument verb.
func (a Action) TakesArgument() bool { _, ok := argumentActions[a]; return ok }

// CheckBinding is the registry's check of one binding: the action must be
// registered, and a value is allowed only for an argument verb, and only one.
func CheckBinding(action string, args []string) error {
	a := Action(action)
	switch {
	case !a.Valid():
		return fmt.Errorf("%q is not an action (`kubecom keys default` lists them)", action)
	case len(args) > 0 && !a.TakesArgument():
		return fmt.Errorf("%s takes no value", action)
	case len(args) > 1:
		return fmt.Errorf("%s takes one value, not %d", action, len(args))
	}
	return nil
}
