# kubecom Vault

This directory is the **single source of truth and working memory** for kubecom —
the Kubernetes dashboard in your terminal. The 2020 kube-commander → kubecom
rewrite is **done** and v1 is shipped on engram update channels via a thin
launcher (see [`PLAN.md`](PLAN.md)). The vault keeps the project's goals, forward
plan, open work, accumulated knowledge and history in the repo, under version
control, discoverable without external context.

> **If you are an agent picking this project up: read this file, then
> [`goals.md`](goals.md), then [`PLAN.md`](PLAN.md), then the open items in
> [`tasks/board.md`](tasks/board.md). Record anything you learn in
> [`knowledge/`](knowledge/).**

## Branch model

- **`main`** — the default branch. **All work happens here**; this vault lives on
  `main`.
- **`v0`** — the original 2020 codebase plus an announcement note, kept as the
  historical reference (D14). Left untouched. It was named `master` until the
  2026-10-07 rename.

## Layout

| Path | Purpose |
|------|---------|
| [`../CLAUDE.md`](../CLAUDE.md) | Agent operating guide: autonomous model, the leg loop, hard rules |
| [`goals.md`](goals.md) | Vision, definition of done, non-goals, principles |
| [`PLAN.md`](PLAN.md) | Current architecture, the release/update model, and the forward roadmap |
| [`tasks/`](tasks/) | Task board + workflow conventions |
| [`feedback/`](feedback/) | Human → agent inbox; checked before every leg, preempts the board, deleted once addressed (D69) |
| [`human-tasks/`](human-tasks/) | Agent → human tasks (only-a-human work); can **block** board items (D79). Currently empty |
| [`knowledge/`](knowledge/) | Durable knowledge: decisions, target stack, keybindings, legacy findings |
| [`journal/`](journal/) | Execution journal: one key-fact file per month, `YYYY-MM.md` |

The rewrite ran **autonomously**: agents self-assigned work and progressed one
small **leg** at a time via the [`/do-rewrite-leg`](../.claude/skills/do-rewrite-leg/SKILL.md)
skill, pushing directly to `main`. That model continues for the remaining work. A
claim on the board is committed and pushed **before** implementation so it acts as
a lock (D16). A human reviews periodically via this vault and the journal. See
[`../CLAUDE.md`](../CLAUDE.md) for the full operating model.

Milestones (M0–M5) were retired in the 2026-10-07 cleanup: the rewrite is
complete, and finished work is recorded in [`knowledge/decisions.md`](knowledge/decisions.md),
the journal and git rather than in milestone files (D291/D300).

## How agents use the vault

1. **Orient** — read [`goals.md`](goals.md), [`PLAN.md`](PLAN.md), the top of the
   board, and the tail of the current month's journal.
2. **Check `feedback/` and `human-tasks/`** — an open `human-tasks/` file's
   `Blocks:` gates what you may pick; if it blocks everything, stop and report
   rather than inventing busywork (D79). Then drain `feedback/`: any file but its
   `README.md` is unaddressed feedback that preempts the board; address it and
   delete the file (D69).
3. **Pick work** — take the next item from [`tasks/board.md`](tasks/board.md).
   Move it to In Progress with your agent id and a timestamp.
4. **Do the work** on `main`, in small, reviewable commits.
5. **Capture knowledge** — anything non-obvious you discover goes into
   `knowledge/` immediately, so the next agent (or a cold-started you) doesn't
   re-derive it.
6. **Update state** — drop finished board items (completed work is dropped, not
   indexed — D291), note blockers.
7. **Leave a trail** — commit messages and the task board should let a fresh agent
   reconstruct where things stand.

## Conventions

- Markdown only; keep files small and single-purpose so they're cheap to load.
- Cross-link with relative links. Prefer linking over duplicating.
- Status vocabulary (used everywhere): `todo` · `in-progress` · `blocked` · `done`.
- Dates are absolute (`YYYY-MM-DD`), never "yesterday"/"last week".
- Decisions are append-only in [`knowledge/decisions.md`](knowledge/decisions.md);
  supersede rather than delete.
