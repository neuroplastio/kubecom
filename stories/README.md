# Stories

A **story** is a goal a user has, walked against a known cluster, with the keystrokes
recorded — the instrument this project uses to find out whether kubecom is actually
pleasant to use (D268).

**The way to argue for a change in how kubecom feels is to write a story.** Not a
description of the change — a description of the situation that made you want it.

A UX argument written as an opinion ("the logs view should have X") can only be
agreed or disagreed with. The same argument written as a story — *here is what I
was trying to do, here is where I stalled* — can be **walked by somebody else**, and
either it stalls them too or it does not. That is the difference between a
preference and a finding, and it is the only way this project has of telling them
apart, because there is no user research team here and no analytics.

[neuroplastio's contributing guide](https://github.com/neuroplastio/.github/blob/main/CONTRIBUTING.md)
has the general method. This page is kubecom's own: the shape its stories share,
the rule its test enforces, the fixture, and the trace.

## What a story is

A goal somebody has, written so a stranger can walk it against a known cluster. A
story states the state it begins from, the task in the user's words, and what to
observe. The existing ones are listed [below](#the-stories) — read one before
writing yours; they share a shape:

| section | what goes in it |
|---|---|
| the metadata block | whether it is the main story, which fixture, where to write the trace |
| **The situation** | the circumstances, in the second person. Why is this person at their terminal? |
| **Your goal** | what they are trying to achieve — never how |
| **When you're done** | what to report back |

## The one hard rule: a story never names a key

A story that says "press `L` to open the logs" measures whether you can follow
instructions. A story that says "find out what that pod is printing" measures
whether kubecom is discoverable — which is the thing actually in question. Which
keys the walker reaches for **is the data**, so a story that supplies them has
destroyed its own measurement.

The same rule applies to naming the feature: "use the resource palette to…" is a
key instruction wearing a different hat. Describe the goal and stop.

A test enforces this (`internal/stories`), so a story that names keys, or lacks one
of the three sections or its fixture line, fails `make check` rather than quietly
producing a useless walk.

## When a story is not the right tool

Bugs, crashes and wrong output do not need one — they need an issue with the steps.
Stories are for the class of problem where nothing is broken and the thing is still
unpleasant to use, which is the class that otherwise gets argued about forever.

## The cluster

Every story runs against the fixture in [`cluster/`](cluster/), which is committed so
the ground does not move between runs:

```bash
./stories/cluster/up.sh      # destroys and rebuilds — always the same starting state
./stories/cluster/down.sh    # remove it
```

It is [k3d](https://k3d.io) — k3s in docker — so it is real Kubernetes and still
disposable. Needs `k3d`, `kubectl` and a running docker; a rebuild takes a couple of
minutes.

| namespace | what's in it | why |
|---|---|---|
| `shop` | storefront (nginx + ConfigMap), checkout (Secret, two containers), logspam, Services, Ingress | the healthy state a story starts from; the subjects for logs, the secret viewer and the container picker |
| `data` | redis StatefulSet + bound PVCs, DaemonSet, a completed Job, a CronJob every 2 min | the kinds a Deployment-only fixture never reaches; the CronJob also makes the table change under a live watch, unprompted |
| `broken` | crashloop, bad-image, unschedulable, never-ready, an unbindable PVC | five failures with five *different* answers — logs, events, the scheduler, the probe, the claim |

The `broken` namespace never becomes ready. That is the point, and `up.sh` does not wait
on it.

## The stories

| id | path | what it walks |
|---|---|---|
| S01 | [`s01-first-contact.md`](s01-first-contact.md) | get oriented on a cluster you have never seen |
| S02 | [`s02-the-failing-pod.md`](s02-the-failing-pod.md) | the **main story** — five failures, five different answers |
| S03 | [`s03-read-the-logs.md`](s03-read-the-logs.md) | find the failures in a chatty stream |
| S04 | [`s04-change-something.md`](s04-change-something.md) | scale, restart, and edit by hand |
| S05 | [`s05-half-a-name.md`](s05-half-a-name.md) | find an object when you only half-remember its name |

S02 is the **main story** — the screencast (TAPE-01) and `docs/usage.md` (DOC-04) are
both cut against it, so the demo, the docs and the test path stay one product.

## Walking one

```bash
./stories/cluster/up.sh                    # the fixture — same starting state every time
./stories/cluster/run.sh s02               # trace -> ~/traces/s02.jsonl
```

Then walk it, and answer the questions at the end in your own words. The trace goes
with what you wrote: it is what makes the story evidence rather than an anecdote.

Without `run.sh`, `export KUBECOM_KEYLOG=~/traces/s02.jsonl` before `kubecom` records
the same trace, but against your real config and cache.

`run.sh` launches kubecom with a **throwaway user dir** (`XDG_CONFIG_HOME`/
`XDG_CACHE_HOME` under `$TMPDIR`, wiped before every launch), so a walk starts with
no config, no per-context menus/state, no discovery cache and no log history — the
same guarantee `up.sh` gives the cluster — and never reads or writes your real
`~/.config/kubecom`. The trace still lands in `~/traces/<id>.jsonl`, outside the
wipe, so it survives.

Then read the trace back with `kubecom keys analyze ~/traces/s02.jsonl`: it
reports the unresolved presses ranked by frequency, the presses on text surfaces
(which may be ordinary typing or a navigation reach the surface swallowed),
the action counts, the longest
pauses (where the walker stopped to think), and the sequences that were started
but never finished. The most interesting line in a raw trace is a keypress that
resolved to no action — that is someone reaching for a key kubecom does not have.
That record is invisible in every other kind of report, including a screen
recording, where it looks like nothing happening. The gaps between timestamps are
the second signal: they are where the walker stopped to think.

A trace records every keypress, which means everything you typed — filter queries,
namespace and object names. Read it before you share it. (Keys pressed inside an
exec shell or your `$EDITOR` never reach it; kubecom is suspended while those run.)
