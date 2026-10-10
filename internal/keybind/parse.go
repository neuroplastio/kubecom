package keybind

import (
	"errors"
	"fmt"
	"strconv"

	kdl "github.com/calico32/kdl-go"
)

// ParseFile reads a whole KDL file holding a `keymap` document — the shape of
// default.kdl and of config.kdl — into one layer's Doc. name is what its lines are
// called in errors. A file with no `keymap` is an empty Doc.
func ParseFile(name string, src []byte, layer Layer) (Doc, error) {
	res := kdl.ParseStringWithDiagnostics(string(src), kdl.WithVersion(kdl.Version2), kdl.WithSourceName(name))
	for _, d := range res.Diagnostics {
		if d.Severity == kdl.SeverityError {
			return Doc{}, fmt.Errorf("%s: %s", d.Start, d.Message)
		}
	}
	for _, n := range res.Document.Nodes {
		if n.Name() == "keymap" {
			return Parse(name, n.Children().Nodes, layer)
		}
	}
	return Doc{layer: layer}, nil
}

// Parse reads the children of a `keymap` document:
//
//	leader "space"
//	group "g" "+goto"
//	bind "g g" "nav.top"
//	bind "g p" "resources.switch" "pods"
//	unbind "ctrl+b"
//	context "logs" { bind "f" "logs.follow" label="follow" }
//
// Every error names its file:line, and any error refuses the whole document: a
// keymap is used whole or not at all.
func Parse(file string, nodes []*kdl.Node, layer Layer) (Doc, error) {
	p := parser{file: file, doc: Doc{layer: layer}}
	p.nodes(nodes, nil)
	return p.doc, errors.Join(p.errs...)
}

type parser struct {
	file string
	doc  Doc
	errs []error
}

func (p *parser) fail(n *kdl.Node, format string, args ...any) {
	p.errs = append(p.errs, fmt.Errorf("%s: %s", p.source(n), fmt.Sprintf(format, args...)))
}

// source is where n was written: the file its parse named, else the one the parser
// was told.
func (p *parser) source(n *kdl.Node) source {
	file := p.file
	if f := n.Location().Filename; f != "" {
		file = f
	}
	return source{File: file, Line: n.Location().Line}
}

// contextSrc is the context a block is under: the expression as written (nested
// blocks are joined with &&) and its compiled form.
type contextSrc struct {
	src string
	c   *Context
}

// nodes reads a list of keymap nodes under a context (nil: everywhere).
func (p *parser) nodes(nodes []*kdl.Node, ctx *contextSrc) {
	for _, n := range nodes {
		switch n.Name() {
		case "leader":
			p.leader(n, ctx)
		case "base":
			p.base(n, ctx)
		case "bind":
			p.bind(n, ctx)
		case "unbind":
			p.unbind(n, ctx)
		case "group":
			p.group(n, ctx)
		case "context":
			p.context(n, ctx)
		default:
			p.fail(n, "%q is not a keymap node (leader, base, bind, unbind, group, context)", n.Name())
		}
	}
}

func (p *parser) leader(n *kdl.Node, ctx *contextSrc) {
	s, ok := p.strings(n, 1, 1, nil)
	if !ok {
		return
	}
	if ctx != nil {
		p.fail(n, "leader cannot be inside a context")
		return
	}
	k, err := ParseKey(s[0])
	if err != nil {
		p.fail(n, "leader: %v", err)
		return
	}
	p.doc.leader = &k
}

// base says which defaults the keymap starts from: "kubecom", the defaults (as
// when it is not given), or "none" — every key is then the user's to bind.
func (p *parser) base(n *kdl.Node, ctx *contextSrc) {
	s, ok := p.strings(n, 1, 1, nil)
	if !ok {
		return
	}
	switch {
	case ctx != nil:
		p.fail(n, "base cannot be inside a context")
	case s[0] != "kubecom" && s[0] != "none":
		p.fail(n, "base is \"kubecom\" (the defaults) or \"none\", not %q", s[0])
	default:
		p.doc.noBase = s[0] == "none"
	}
}

func (p *parser) bind(n *kdl.Node, ctx *contextSrc) {
	args, ok := p.strings(n, 2, -1, map[string]kdl.ValueKind{"label": kdl.String})
	if !ok {
		return
	}
	seq, err := parseSeq(args[0])
	if err != nil {
		p.fail(n, "bind: %v", err)
		return
	}
	b := Binding{seq: seq, Action: args[1], layer: p.doc.layer, src: p.source(n)}
	if len(args) > 2 {
		b.Args = args[2:]
	}
	if v := n.Prop("label"); v.Kind() == kdl.String {
		b.Label = v.String()
	}
	if ctx != nil {
		b.ctx = ctx.c
	}
	p.doc.binds = append(p.doc.binds, b)
}

// unbind takes a key sequence away from every lower layer and earlier line where
// its context holds: a binding to nothing, which wins by the same precedence as
// any other, so the key does there what it would do unbound.
func (p *parser) unbind(n *kdl.Node, ctx *contextSrc) {
	args, ok := p.strings(n, 1, 1, nil)
	if !ok {
		return
	}
	seq, err := parseSeq(args[0])
	if err != nil {
		p.fail(n, "unbind: %v", err)
		return
	}
	b := Binding{seq: seq, unbind: true, layer: p.doc.layer, src: p.source(n)}
	if ctx != nil {
		b.ctx = ctx.c
	}
	p.doc.binds = append(p.doc.binds, b)
}

func (p *parser) group(n *kdl.Node, ctx *contextSrc) {
	args, ok := p.strings(n, 2, 2, nil)
	if !ok {
		return
	}
	seq, err := parseSeq(args[0])
	if err != nil {
		p.fail(n, "group: %v", err)
		return
	}
	g := group{seq: seq, Label: args[1], layer: p.doc.layer, src: p.source(n)}
	if ctx != nil {
		g.ctx = ctx.c
	}
	p.doc.groups = append(p.doc.groups, g)
}

func (p *parser) context(n *kdl.Node, outer *contextSrc) {
	if len(n.Arguments()) != 1 || n.Arg(0).Kind() != kdl.String || len(n.Properties()) > 0 {
		p.fail(n, "context takes one expression and a block")
		return
	}
	src := n.Arg(0).String()
	if outer != nil {
		src = "(" + outer.src + ") && (" + src + ")"
	}
	c, err := CompileContext(src)
	if err != nil {
		p.fail(n, "%v", err)
		return
	}
	p.nodes(n.Children().Nodes, &contextSrc{src: src, c: c})
}

// strings reads a node's arguments as strings (a number is written as its
// digits), at least lo and at most hi of them (hi < 0: no limit), and checks its
// properties against the allowed kinds. A node with children is refused: only
// context has a block.
func (p *parser) strings(n *kdl.Node, lo, hi int, props map[string]kdl.ValueKind) ([]string, bool) {
	if len(n.Children().Nodes) > 0 {
		p.fail(n, "%s takes no block", n.Name())
		return nil, false
	}
	args := n.Arguments()
	if len(args) < lo || (hi >= 0 && len(args) > hi) {
		p.fail(n, "%s takes %s", n.Name(), arity(lo, hi))
		return nil, false
	}
	out := make([]string, len(args))
	for i, a := range args {
		switch a.Kind() {
		case kdl.String:
			out[i] = a.String()
		case kdl.Int:
			out[i] = strconv.Itoa(a.Int())
		default:
			p.fail(n, "%s's argument %d is not a string", n.Name(), i+1)
			return nil, false
		}
	}
	for name, v := range n.Properties() {
		kind, ok := props[name]
		switch {
		case !ok:
			p.fail(n, "%s has no property %q", n.Name(), name)
			return nil, false
		case v.Kind() != kind:
			p.fail(n, "%s's %s is the wrong kind of value", n.Name(), name)
			return nil, false
		}
	}
	return out, true
}

func arity(lo, hi int) string {
	switch {
	case lo == 1 && hi == 1:
		return "one argument"
	case lo == hi:
		return fmt.Sprintf("%d arguments", lo)
	case hi < 0:
		return fmt.Sprintf("at least %d arguments", lo)
	}
	return fmt.Sprintf("%d to %d arguments", lo, hi)
}
