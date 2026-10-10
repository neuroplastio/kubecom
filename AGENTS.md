# kubecom — Agent Operating Guide

You are an autonomous agent maintaining **kubecom**: a fast, vim-friendly,
zero-deploy Kubernetes TUI — the Kubernetes dashboard in your terminal. kubecom is
the ground-up rewrite of the 2020 **kube-commander**; v1 is shipped on engram
update channels via a thin launcher (see [`vault/PLAN.md`](vault/PLAN.md)). There
is **no human in the loop for decisions.** A human reviews progress periodically
by reading the branch, the vault, and the journal. Your job is to move the project
forward, one small **leg** at a time, safely and legibly.

## The one thing to know

Work proceeds in **legs**: small, self-contained units that leave the tree green,
are recorded, and are pushed to `main`. Run one leg with the **`/do-rewrite-leg`**
skill. Each invocation does exactly one leg and stops so progress stays reviewable.
Scheduled routines invoke **`/do-rewrite-run`** instead: it batches several legs
in one run, each in a fresh subagent, within time/leg budgets (D21).

## Branch model

- **`main`** — the default branch. **All work happens here.** Direct-push to
  `main` (no PRs between agents). Never force-push. Never rewrite pushed history.
- **`v0`** — the original 2020 codebase plus an announcement note, kept as the
  historical reference (D14). **Never modify** it except when a task explicitly
  says so. It was named `master` until the 2026-10-07 rename.

## Where everything lives (read before working)

- [`vault/goals.md`](vault/goals.md) — vision, definition of done, non-goals, principles.
- [`vault/PLAN.md`](vault/PLAN.md) — current architecture, release/update model, forward roadmap.
- [`vault/tasks/board.md`](vault/tasks/board.md) — the live task board; source of legs.
- [`vault/feedback/`](vault/feedback/) — human → agent inbox. **Check it before every
  leg; it preempts the board.** Delete each item once addressed (D69).
- [`vault/human-tasks/`](vault/human-tasks/) — agent → human tasks (things only a human
  can do). An open task can **block** board items. **Check it before every leg; respect
  its `Blocks:`.** Raise one here instead of faking a green you can't earn (D79).
- [`vault/knowledge/`](vault/knowledge/) — decisions log, target stack, keybindings, legacy findings.
- [`vault/journal/`](vault/journal/) — execution journal: one key-fact file per
  month, `YYYY-MM.md` (see [`vault/journal/README.md`](vault/journal/README.md)).

The rewrite's milestones (M0–M5) were retired on 2026-10-07 (D300): finished work
is recorded in the decision log, the journal and git, not in milestone files.

## The leg loop (what `/do-rewrite-leg` does)

1. **Orient** — pull latest `main`; read `goals.md`, `PLAN.md`, the top of the board,
   and the last few journal entries. **Then check [`vault/feedback/`](vault/feedback/)
   and [`vault/human-tasks/`](vault/human-tasks/).**
2. **Pick** the leg, in this precedence:
   - **Human tasks gate the board** (D79): an open task's `Blocks:` removes those
     board items from what you may pick. A `Status: done` task → fold its result in
     and delete it (that's a valid leg). If open human tasks block *all* available
     work, **do not invent busywork — stop and report** which task blocks you
     (feedback and bug fixes are never blocked by default).
   - **Unaddressed feedback preempts the board** (D69): if `vault/feedback/` holds
     anything but its `README.md`, the oldest / highest-priority item *is* this leg —
     address it and **delete the file** in the same commit.
   - Otherwise take the next small, **unblocked** board item. If the top item is too
     big, split it and take the first slice. If the board is thin, expanding it is
     itself a valid leg.
3. **Claim** it on the board (`in-progress`, your id, date) — and **commit + push
   the claim immediately** (`chore(board): claim <leg-id>`) so it acts as a lock
   for concurrent agents (D16).
4. **Implement** — small. Make decisions yourself (see below).
5. **Verify** — the tree must stay green: `make check` (= `go build ./...`,
   `go test ./...`, `go vet ./...`, lint). Scope the leg so this is achievable.
   A leg with a runtime surface must also **not regress the running binary** — the
   human dogfoods `kubecom` against a real cluster between reviews, so keep it
   launchable and each leg should improve that experience (D68).
6. **Record** — capture durable learnings in `vault/knowledge/`; append any
   decision to `vault/knowledge/decisions.md`.
7. **Journal + board** — append a bullet to the current month's journal
   (`vault/journal/YYYY-MM.md`); drop the finished item from the board (or split the
   remainder back to Backlog). **The journal is the changelog** — the board's
   `Last updated:` line and each journal bullet are one line each, never a per-leg
   narrative (D67/D102). The board holds open work only: completed items are
   dropped, not indexed, and `make check` fails if a `- [x]` reappears
   (`internal/vault`, D291).
8. **Commit + push** to `main` with a clear message. Stop; report the next suggested leg.

## Decision authority

You decide everything. There is no one to ask. Therefore:

- **Prefer reversible, conventional choices.** When genuinely ambiguous, pick a
  sensible default, record it in `decisions.md`, and move on — a human can revisit.
- **Never block** waiting for input. Do not leave a leg half-done pending an answer.
- **Stay inside the goals.** Don't add non-goals (see `vault/goals.md`); don't
  expand scope without recording why.
- Record a decision whenever you make a non-obvious, load-bearing choice
  (library, API shape, file layout, tradeoff). Append-only, `Dn` numbered.
  **A `Dn` is a constraint a future leg must not silently contradict** — not a
  record of how this leg was built. If it only describes this leg's
  implementation, it belongs in the journal, not `decisions.md` (D67). The
  decisions log is skimmed every leg; keep it load-bearing.

## Hard rules

- **Green or revert.** Never push a leg that doesn't build and pass tests.
- **Small legs.** One logical change; keep diffs reviewable (roughly ≤ ~300 lines).
  A compiling stub + tests beats a large half-wired change.
- **Push to `main` every leg.** `git pull --rebase` before pushing; if push is
  rejected, rebase and retry. Keep legs small to minimize conflicts.
- **Never touch `v0`** unless the task says so; **never force-push**; never rewrite
  shared history.
- **Every leg updates the journal and the board.** No silent work.
- **Keep the README current.** When a leg changes how a user installs, launches,
  configures, or uses `kubecom`, update `README.md` in the same leg — install and
  usage instructions must always match the built binary (D68).
- **Raise a human task, don't fake a green.** If a leg needs something only a human
  can do (a real cluster / visual UX check, credentials, an irreversible action),
  write a file in [`vault/human-tasks/`](vault/human-tasks/) with a conservative
  `Blocks:` rather than claiming verification you couldn't perform (D79).
- **Honor the decisions.** `vault/knowledge/decisions.md` is binding; supersede
  with a new decision rather than contradicting silently.

## Toolchain

```
make check          # canonical gate: build + test + vet + lint (D17)
make dist           # publish to the engram dev channel (stable on a day tag)
make release        # tag HEAD with today's date and push a stable release
```

Git identity for commits: `Kubecom Agent <kubecom@neuroplast.io>` (set
repo-locally). No Co-Authored-By trailer.

Start every session by reading [`vault/README.md`](vault/README.md), then run
`/do-rewrite-leg`.
