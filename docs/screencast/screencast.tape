# kubecom screencast tape (M5-09) — records docs/screencast.gif with vhs, and the same
# tour as docs/screencast.cast (CAST-01, D290), an asciicast of kubecom's own pane that
# castrec (docs/screencast/castrec) writes while vhs films it.
#
#     make screencast        # builds ./bin/kubecom and ./bin/castrec, then runs vhs on this file
#
# What the recording needs, none of which a sandbox has (which is why the tape is
# committed but the GIF is recorded by a human — see
# vault/human-tasks/2026-07-30-record-screencast.md):
#
#   * `vhs` on PATH — https://github.com/charmbracelet/vhs
#   * a real cluster as the kubeconfig's current context, with pods running whose
#     names/namespaces match the FILTER below, and at least one of them printing
#     logs (the tour opens a log stream and greps it);
#   * `kubecom` on PATH — `make screencast` puts *this checkout's* binary there, so
#     the GIF always shows the code it was recorded from.
#
# TUNE: three strings in the tour are cluster-specific — the resource filter
# (`logspam` below), the log grep query (`info`) and the cluster-search query. The
# filter must leave the pod whose logs the tour opens on the selected row, a pod that
# prints lines the grep matches; the Pods table lists every namespace and the filter
# reads names. The search reuses the filter's string so the cluster-wide demo cannot
# come back empty — it must show the objects the tour has already put on screen
# (feedback 2026-08-09-screencast-tape-tuning). Change any of the three to whatever
# reads best on the cluster you record against, re-run `make screencast`, and commit
# the GIF and the cast with the tape change.
#
# Reruns are reproducible: kubecom's per-context state (remembered pane, menu pins,
# last namespace — CTX-MEM) is pointed at a throwaway XDG dir that the setup below
# wipes before every launch, so the second run opens on the same welcome screen as
# the first.
#
# Every keypress below carries an annotation on the line above it: either
# `# kubecom-action: <id>`, naming the keymap action the press is meant to trigger,
# or `# kubecom-input: <what>` when the keypress is plain text (a shell command, a
# filter query). TestScreencastTapeMatchesTheKeymap (internal/tui/keymap) checks
# every annotated key against the default keymap and fails `make check` on an
# unannotated key — so rebinding an action, or slipping an unchecked keypress into
# the tour, breaks the build instead of quietly turning this tape (and the GIF made
# from it) into a lie about the keys kubecom actually has.

Require kubecom

Output docs/screencast.gif

Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 700
Set Padding 20
Set Framerate 24
Set TypingSpeed 90ms
# A clean prompt to start from: the tour is about kubecom, not about the shell.
# Set up a tmux session for the captions.
Hide
Type@1ms "./docs/screencast/setup.sh" Enter
Sleep 1s

Type@1ms "export PATH=$PWD/bin:$PATH PS1='$ '" Enter

# Point kubecom's state at a throwaway XDG dir and wipe it, so this run starts from
# the welcome screen no matter what the last run left behind (the remembered pane,
# menu pins and last namespace are per-context state, D240). Recorded once, applied
# to every rerun — the recording is reproducible by construction.
# kubecom-input: reset kubecom's per-run state, then redirect its state dirs
Type@1ms "rm -rf /tmp/kubecom-screencast && export XDG_CONFIG_HOME=/tmp/kubecom-screencast && export XDG_CACHE_HOME=/tmp/kubecom-screencast/cache" Enter

# kubecom-input: clear the recording shell
Type@1ms "clear"
# kubecom-input: run it
Enter
Sleep 500ms

# The cast (CAST-01, D290): tmux pipes the pane's output to castrec from here,
# and castrec starts the cast with the pane as it stands, the clean prompt. The
# captions below reach it as markers, each `run-shell` beside its `set`.
# kubecom-input: tmux prefix
Ctrl+B
Type@1ms ":pipe-pane -o 'castrec rec #{socket_path} #{pane_id}'" Enter
Sleep 500ms

# kubecom-input: tmux prefix
Ctrl+B
Type@1ms ":set -g status-left '   #[fg=cyan,bold]Welcome to kubecom!' ; run-shell 'castrec mark #{q:status-left}'" Enter
Show

# ---------------------------------------------------------------- launch -----
# Bare `kubecom` opens the browse UI on the current context. The welcome page is
# up immediately; discovery fills the resource menu behind it (fast cold start).
# kubecom-input: launch kubecom on the current kubeconfig context
Type "kubecom"
# kubecom-input: run it
Enter
Sleep 1s

# ------------------------------------------------------- browse: open Pods ---
Hide
# kubecom-input: tmux prefix
Ctrl+B
Type@1ms ":set -g status-left '   #[fg=cyan,bold]The command palette is your central hub for navigation' ; run-shell 'castrec mark #{q:status-left}'" Enter
Show

# `:` is the command palette — one place you type what you want to do. The verb and
# its argument are one line in one box (PAL-03a): the tour types the verb, Enters to
# commit it, and the kinds open in navigation mode (D272), so `/` filters them on the
# same line — `:resource pods` → Pods.
# kubecom-action: app.palette
Type ":"
Sleep 500ms
# kubecom-input: narrow the palette to the resource switch
Type "resource"
Sleep 500ms
# kubecom-action: nav.drillIn
Enter
Sleep 500ms
# kubecom-action: app.filter
Type "/"
Sleep 300ms
# kubecom-input: narrow the palette's resource argument to Pods
Type "pods"
Sleep 500ms
# kubecom-action: nav.drillIn
Enter
Sleep 3s

# ---------------------------------------------------------------- filter -----
Hide
# kubecom-input: tmux prefix
Ctrl+B
Type@1ms ":set -g status-left '   #[fg=cyan,bold]Filter live resources as you type' ; run-shell 'castrec mark #{q:status-left}'" Enter
Show

# The table is a live watch, not a snapshot. `/` narrows it as you type; Enter
# keeps the narrowing and hands the keys back to navigation, the one pod left
# selected (Enter on a row would open its actions, D271).
# kubecom-action: app.filter
Type "/"
Sleep 500ms
# kubecom-input: TUNE — filter text, a pod that logs
Type "logspam"
Sleep 500ms
# kubecom-input: enter
Enter
Sleep 1500ms

# ------------------------------------------------------------------ logs -----
Hide
# kubecom-input: tmux prefix
Ctrl+B
Type@1ms ":set -g status-left '   #[fg=cyan,bold]Press L to tail logs for the selected pod' ; run-shell 'castrec mark #{q:status-left}'" Enter
Show

# `L` gives logs the whole screen and tails the container live.
# kubecom-action: res.logs
Type "L"
Sleep 5s
Hide
# kubecom-input: tmux prefix
Ctrl+B
Type@1ms ":set -g status-left '   #[fg=cyan,bold]Grep logs instantly without interrupting the stream' ; run-shell 'castrec mark #{q:status-left}'" Enter
Show

# `/` greps the stream *while it keeps following* — nothing is re-fetched.
# kubecom-action: app.filter
Type "/"
Sleep 500ms
# kubecom-input: TUNE — the log grep query
Type "info"
Sleep 2s
# kubecom-action: nav.back
Escape
# kubecom-action: app.quit
Type "q"
Sleep 1500ms

# -------------------------------------------------------------- describe -----
Hide
# kubecom-input: tmux prefix
Ctrl+B
Type@1ms ":set -g status-left '   #[fg=cyan,bold]Press d to describe the selected resource' ; run-shell 'castrec mark #{q:status-left}'" Enter
Show

# `d` describes the selected row in the shared centered viewer; it scrolls like
# any other view.
# kubecom-action: res.describe
Type "d"
Sleep 3s
# kubecom-action: nav.halfPageDown
Ctrl+D
Sleep 2500ms
# kubecom-action: nav.back
Escape
Sleep 1s

# -------------------------------------------------- cluster-wide search -------
Hide
# kubecom-input: tmux prefix
Ctrl+B
Type@1ms ":set -g status-left '   #[fg=cyan,bold]Ctrl+F searches across all resources in the cluster' ; run-shell 'castrec mark #{q:status-left}'" Enter
Show

# Ctrl+F searches across kinds at once — Kind, namespace, name — ranked, best
# match first, streaming in as each kind answers.
# kubecom-action: search.cluster
Ctrl+F
Sleep 1500ms
# kubecom-input: the same string the filter step already matched, so the search
# cannot demo zero results — the cluster-wide pass must find the pod on screen
Type "logspam"
Sleep 4s
# kubecom-action: nav.back
Escape
Sleep 500ms
# kubecom-action: nav.back
Escape
Sleep 1s

# ------------------------------------------------------------- theme preview ----
Hide
# kubecom-input: tmux prefix
Ctrl+B
Type@1ms ":set -g status-left '   #[fg=cyan,bold]Press T to preview the themes' ; run-shell 'castrec mark #{q:status-left}'" Enter
Show

# `T` opens the palette on its `:theme ` stage. Moving the cursor previews each
# palette across the whole screen live — painted background included — so you can
# try the look before committing; `esc` keeps the theme you opened with (THEME-07).
# kubecom-action: theme.switch
Type "T"
Sleep 1s
# kubecom-action: nav.down
Down
Sleep 1s
# kubecom-action: nav.down
Down
Sleep 1s
# kubecom-action: nav.back
Escape
Sleep 1s

# -------------------------------------------------------------------- help -----
Hide
# kubecom-input: tmux prefix
Ctrl+B
Type@1ms ":set -g status-left '   #[fg=cyan,bold]Press ? for the full keymap' ; run-shell 'castrec mark #{q:status-left}'" Enter
Show

# `?` overlays the complete keymap — every action, the vim key and its classic
# fallback side by side (M2-01d), rebindable from config.
# kubecom-action: app.help
Type "?"
Sleep 2s
# kubecom-action: nav.back
Escape
Sleep 1s

# ------------------------------------------------------------------ quit -----
Hide
# kubecom-input: tmux prefix
Ctrl+B
Type@1ms ":set -g status-left '   #[fg=cyan,bold]Thanks for watching' ; run-shell 'castrec mark #{q:status-left}'" Enter
Show

# kubecom-action: app.quit
Type "q"
Sleep 2s

Hide
# Closing the pipe ends the cast: castrec writes docs/screencast.cast.
# kubecom-input: tmux prefix
Ctrl+B
Type@1ms ":pipe-pane" Enter
Sleep 500ms
Type@1ms "./docs/screencast/teardown.sh" Enter
Sleep 1s
