# Task Tracking

Lightweight, in-repo task tracking so any agent (cloud or local, cold or warm)
can see what's in flight and pick up work without external context.

## Where tasks live

- **[`board.md`](board.md)** — the live board: Backlog / In Progress / Blocked.
  Completed items are dropped, not kept (D291); the journal, commits and the
  decision log are their record. There are no milestone files any more (D300):
  the rewrite is complete, and [`../PLAN.md`](../PLAN.md) holds the forward plan.

## Workflow

1. **Pick** the top unblocked item from Backlog (respect milestone order).
2. **Claim** it: move to In Progress, add `@agent-id` and `YYYY-MM-DD`.
3. **Work** on `main` in small commits; reference the task id in commit messages.
4. **Capture** any durable learning in [`../knowledge/`](../knowledge/).
5. **Close**: **drop** the item from the board — completed items are not kept on
   the board (D291); the journal bullet, the commit and the decision log are its
   record. If blocked, move to Blocked with the reason and what would unblock it.

## Task id format

`M<milestone>-<seq>` — e.g. `M1-03`. Keep ids stable once assigned.

## Item template

```
- [ ] **DOC-05** `docs/troubleshooting.md`
      status: todo | owner: — | added: 2026-08-15
      notes: the failure surfaces exist and are documented nowhere a user looks
      → knowledge: decisions.md D268
```

Status vocabulary: `todo` · `in-progress` · `blocked` · `done`.
