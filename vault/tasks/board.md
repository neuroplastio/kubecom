# Task Board

Live board for the kubecom rewrite. See [`README.md`](README.md) for workflow and
the item template. Status: `todo` · `in-progress` · `blocked` · `done`.

_Last updated: 2026-10-06 — vault cleanup (D291): completed items dropped, journals compressed to per-month digests, and the backlog reduced to the remaining work._

## In Progress

## Blocked

- [ ] **M5-11** Make the rewrite the default branch (`v1` → `main`)
      status: blocked | owner: — | added: 2026-07-30
      notes: Blocked on human task `2026-07-30-first-release-tag` — the maintainer deferred the
      tag on 2026-08-09 ("a bit too early for that"), so the block stands; renaming the branch before
      a release exists would retarget every clone and PR for a tree nobody can install yet, and
      the rename also dissolves the `@v1` collision the tag is what actually fixes. (The release-
      namespaces blocker that also gated M5-11 was resolved 2026-08-09 — D253/D254 folded in.)
      The agent share is preparation: what to rename, `master` kept as the permanent 2020 reference
      (D14, do *not* delete), the workflow `branches:` lists (both already name `main`) and the
      README/vault links that say `v1`, and neuroplast.io's URL for the screencast cast (D290 pt 3). The act itself is a GitHub admin setting — a human's.
      → milestone: M5

## Backlog

### M5 — Release & docs

_(M0–M4 are **done**; their exit criteria and history live in
[`../milestones/`](../milestones/), [`../knowledge/decisions.md`](../knowledge/decisions.md)
and the journal. Everything still open is below.)_

- [ ] **STORY-06i-3** Reverse-selector relations — a pod's Services (and the workloads whose selector matches it) folded into `kube.Relations`, so the popup answers "who routes to this pod" as well as "who made it"
      status: todo | owner: — | added: 2026-08-21
      notes: Third slice of STORY-06i, deliberately after the popup: unlike every 06i-1 edge this one cannot be read off the object — it needs a List of the namespace's Services and a client-side selector match, so it is a different cost class and gets its own leg. Bound it (namespace-scoped, one List) and let a failure degrade the relation away rather than fail the whole set.
      → milestone: M5 · knowledge: decisions.md D268

- [ ] **TAPE-01** Re-cut `docs/screencast.tape` around the main story
      status: todo | owner: — | added: 2026-08-15
      notes: Depends on STORY-04 (the main story) and lands after STORY-06 so the GIF shows
      the refined UX, not the one the stories found fault with. Today's tape is a **feature
      tour** — palette, filter, logs, describe, search, theme, help — assembled before any
      user path existed; the maintainer wants it re-cut so it walks the main story instead.
      The three guards in `internal/tui/keymap/screencast_test.go` (annotated keypresses
      checked against `DefaultKeymap`, headline actions pressed, README/GIF agreement) hold
      across the re-cut and the headline list may need revising with it. The tape also stops
      depending on an ad-hoc cluster once STORY-01 lands. **Re-recording is the maintainer's**
      (D181): it needs ttyd + ffmpeg, a real terminal and the fixture cluster.
      → milestone: M5 · knowledge: decisions.md D268, D181

- [ ] **DOC-03** README becomes a landing page
      status: todo | owner: — | added: 2026-08-15
      notes: Today's README is 212 lines and carries a full feature catalogue ("What kubecom
      can do", six subsections) that duplicates what `docs/` should own. Cut it to what a
      landing page owes a reader: what/why, the screencast, a short install block linking
      `docs/install.md` (DOC-01's shape, keep it), one screen of tour, and links out. The
      install-path drift guards (`TestReadmeBrewTapMatchesTheCask`,
      `TestReadmeAURPackageMatchesTheConfig`, `TestScreencastAssetAndReadmeAgree`) all read
      `README.md`, so the block they check must survive the cut.
      → milestone: M5 · knowledge: decisions.md D268

- [ ] **DOC-04** `docs/usage.md` — the guided tour the README no longer carries
      status: todo | owner: — | added: 2026-08-15
      notes: Depends on STORY-04. The prose walks the **main story**, so the doc, the tape
      and the story tell one story in three media (D268 pt 4). Absorbs the feature catalogue
      DOC-03 cuts, and links `docs/keybindings.md` for the full keymap rather than restating
      keys — a second hand-written key list is a second thing to drift (D51).
      → milestone: M5 · knowledge: decisions.md D268

- [ ] **DOC-05** `docs/troubleshooting.md`
      status: todo | owner: — | added: 2026-08-15
      notes: The failure surfaces exist and are documented nowhere a user looks: the auth
      diagnosis (`internal/tui/authdiag.go`), browse failures, the log file at
      `os.UserCacheDir()/kubecom/kubecom.log`, discovery partial-failure reporting (DISC-01),
      Gatekeeper quarantine on macOS, and the WSL2 note. Best written after STORY-05, which
      is the first time anyone hits these paths without knowing the code.
      → milestone: M5 · knowledge: decisions.md D268

- [ ] **BOX-04** The body's two-column dead gap — the menu's frame renders two columns narrower than the width `resize` sizes it to (its `View` hands `m.width-2` to a border-box frame), so the browse body is 78 columns wide on an 80-column terminal and the rightmost two are bare canvas
      status: todo | owner: — | added: 2026-08-20
      notes: Agent-found while landing D284 (whose pt 3 is the workaround: composite from `rightPaneX()`, the *rendered* menu width, not `menuPaneWidth`). The fix is to settle one border-box convention across the two panes — the table already renders exactly the width it is sized to — and then `rightPaneX()` may collapse back to `paneWidths()`. Guard it with a body-width assertion (`lipgloss.Width(browseBody()) == m.width`), the one property no current test states.
      → milestone: M5 · knowledge: decisions.md D284
