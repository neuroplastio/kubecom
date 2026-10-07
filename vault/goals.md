# Top-Level Goals

## Vision

A fast, approachable, keyboard-driven terminal UI for observing and operating
Kubernetes clusters — *"kubernetes-dashboard in your terminal"*.

kubecom is the ground-up rewrite of the 2020 **kube-commander** on a modern Go
stack: same idea, no legacy. It reaches feature parity with the original,
eliminates its data-race / focus / redraw bug class by construction, removes the
hard `kubectl` binary dependency, and adds the high-value capabilities the
original lacked.

## Definition of done (v1) — shipped

The v1 promises are met in substance and released (stable `26.10.06`/`26.10.07`
on the engram channels and the packages). The delivery criteria were audited box
by box (M5-01/D174 and the review fold-ins that followed):

- Two-pane browse, generic CRD listing, logs/describe readers, in-place editing,
  the curated in-process action set, context/namespace switching, vim-first
  navigation, fully rebindable keys, responsive cold start, no required
  `kubectl`, and config migration are all done and evidenced.
- No `kubectl` binary is required anywhere except the optional exec fallback.
- The read-only YAML viewer was deliberately folded into the `$EDITOR` edit flow
  (D135/D178).
- The context switcher and the CRD listing closed on the maintainer's recorded
  waivers, with the hermetic coverage named as the standing verification
  (D256).

Two administrative items remain, both tracked on the
[board](tasks/board.md): rewording the delivery criteria to the
channel/launcher model now that goreleaser is retired (ENGRAM-03), and closing
the resolved GitHub issues now that a release carries the fixes.

## What's next

The forward plan — remaining packaging, docs and UX work — is in
[`PLAN.md`](PLAN.md) and the live [`tasks/board.md`](tasks/board.md).

## Non-goals

- **Native Windows support** — dropped. Windows users run kubecom under **WSL2**.
- **Cloning k9s.** k9s is feature-dense but its UX is deliberately *not* our
  model; kubecom stays simpler and more approachable.
- Cluster mutation beyond the curated action set (no arbitrary `apply`, no
  manifests authoring).
- Multi-cluster dashboards / server mode. Single-context, local, zero-deploy.

## Principles

1. **No shared mutable UI state.** Concurrency flows through Bubble Tea messages,
   not mutexes. This is the whole reason for the framework choice — protect it.
2. **In-process first.** Reach for client-go before shelling out. Shell out only
   where genuine interactivity requires it (exec, editor).
3. **Degrade, don't crash.** A missing API group, RBAC denial, or bad namespace
   degrades one feature; it never panics or blocks the UI.
4. **Fast cold start.** Never block first paint on discovery or network round-trips.
5. **Discoverable process.** Knowledge and state live in the vault, not in an
   agent's head or a chat log.
6. **Vim-first, never vim-only.** `hjkl` and friends are the primary path; arrows
   and classic keys are always an equivalent fallback. See
   [`knowledge/keybindings.md`](knowledge/keybindings.md).
7. **Zero hard-coded keys.** All input flows through a configurable action
   registry; no view matches a raw key. Every binding is rebindable via config.
8. **Keep it runnable; dogfood it.** The binary stays launchable every leg. A
   human periodically installs `kubecom` and runs it against a real cluster; each
   leg must incrementally improve — never regress — that real-cluster experience,
   and keep the README's install/usage current (D68). Fake-tested parts are not
   "done" until they work in the running binary.

See [`PLAN.md`](PLAN.md) for the architecture, the release/update model and the
forward roadmap, and [`knowledge/decisions.md`](knowledge/decisions.md) for the
locked decisions.
