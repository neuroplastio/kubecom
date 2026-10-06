# Execution Journal

A compressed, per-month changelog of the rewrite's legs. One file per month,
named **`YYYY-MM.md`**. Entries are appended at the bottom, newest last, each
under a `### YYYY-MM-DD` date heading (add the heading when the day's first leg
lands). This is how a human reviews progress and how a cold-started agent learns
what just happened.

## Entry format

One bullet per **leg** (see [`/CLAUDE.md`](../../CLAUDE.md) and the
[`/do-rewrite-leg`](../../.claude/skills/do-rewrite-leg/SKILL.md) skill). Keep it
to the key facts — the full rationale for a leg belongs in the decision log and
the commit, not here:

```
- **<leg-id>** <short title> — <what changed and why, 1–3 sentences> _(Decisions: Dnn, …)_
```

The parenthetical is omitted when the leg recorded no decision. There is no
`Commit:` field — the entry is written before the commit exists. The **leg id in
the commit message** (e.g. `M0-01`) is the join key between a journal entry, the
board, and git history (D15); the date heading is the second.

## History

Until 2026-10-06 the journal was one file per leg (`YYYY-MM-DD.N.md`, `N` not
zero-padded) — 311 files, ~2 MB. Those were collapsed into these per-month
digests, key facts only (D291); the verbatim entries remain in git history at the
commit before the cleanup. Two consequences worth knowing:

- **Leg ids and decisions survive**, so the journal is still the changelog and
  the D15 join key still resolves.
- **The old `N` sequence is gone.** Nothing sorts by it any more: the newest work
  is simply the bottom of the current month's file. Read the current month, or
  `git log`, for the freshest detail.
