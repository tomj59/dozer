# Manual test cases

Each case is a **packaged dozer workspace**: one self-contained script that sets up exactly what the test needs. A **guide** pane (pane 1) shows the steps and the expected result. If the text is taller than the pane, it opens in `less`; scroll, then `q` leaves it on screen.

> **After pulling changes, run `make install`, not just `make`.** The cases run the `dozer` on your PATH; plain `make` only rebuilds `bin/dozer`. The status bar shows the version (commit) actually running.

```sh
make install                      # once: puts dozer on your PATH (or: export DOZER_BIN=$PWD/bin/dozer)
tests/cases/K03-resize.sh         # run one case
tests/cases/K03-resize.sh --show-config   # read it without running
make cases                        # validate every case
```

Record **PASS / FAIL** plus notes (terminal app; a screenshot for anything visual). To change a case: `tests/cases/X.sh --show-config > x.yaml`, edit, then `dozer -c x.yaml --package tests/cases/X.sh`. To add one, write a YAML with `name:` and a `description:` (Steps / Expect) and package it.


## C: Config & commands

| ID | Case | Milestone | Result |
|---|---|---|---|
| C01 | [run: vs exec:](C01-run-vs-exec.sh) | M1 | PASS · Terminal.app · 2026-09-26 |
| C02 | [Per-pane cwd and env](C02-cwd-env.sh) | M1 | PASS · Terminal.app · 2026-09-26 (retest on 7a387d4: cwd ~ → /Users/tomj) |
| C03 | [Panes don't inherit the host terminal's identity](C03-env-isolation.sh) | M2 | PASS · Terminal.app · 2026-09-26 (7a387d4: TERM_PROGRAM=dozer, no TERM_SESSION_ID/TMUX, no 'Restored session') |

## D: Dead panes & lifecycle

| ID | Case | Milestone | Result |
|---|---|---|---|
| D01 | [Dead-pane states and chrome](D01-dead-states.sh) | M1 | PASS · Terminal.app · 2026-09-26 |
| D02 | [Restoring dead panes (r / R)](D02-restore.sh) | M1 | PASS · Terminal.app · 2026-09-26 |
| D03 | [Automatic restart: back-off and the retry limit](D03-auto-restart.sh) | M2 | PASS · Terminal.app · 2026-09-26 (back-off + retry limit) |
| D04 | [Kill a pane (x)](D04-kill.sh) | M2 | PASS · Terminal.app · 2026-09-26 (recheck: 'killed (exit 1)' on cf4b826) |
| D05 | [quit_when_all_exited: true](D05-quit-when-all-exited.sh) | M1 | PASS · Terminal.app · 2026-09-26 |
| D06 | [Quit confirmation](D06-quit-confirm.sh) | M1 | PASS · Terminal.app · 2026-09-26 |

## E: Terminal emulation

| ID | Case | Milestone | Result |
|---|---|---|---|
| E01 | [Colors and Unicode](E01-color-unicode.sh) | M0 | PARTIAL · Terminal.app · 2026-09-26: colors, styles, Unicode pass; shrink then grow loses cropped output → DP-7 (reflow, M4) |
| E02 | [Full-screen programs side by side](E02-fullscreen-apps.sh) | M0 | PASS · Terminal.app · 2026-09-26 (top, less, vim side by side) |
| E03 | [Heavy output in several panes](E03-flood.sh) | M0 | PASS · Terminal.app · 2026-09-26 (3 × 1M lines concurrently, ~2.9 s each) |
| E04 | [Shell line editing and history](E04-line-editing.sh) | M0 | PASS · Terminal.app · 2026-09-26 |
| E05 | [Bracketed paste and Option/Meta keys](E05-paste-and-meta.sh) | M0 | PARTIAL · Terminal.app · 2026-09-26: bracketed paste OK (indent kept), Option-b/f OK; mouse selection spans panes → DP-8 (M4) |

## K: Keys & controls

| ID | Case | Milestone | Result |
|---|---|---|---|
| K01 | [Focus navigation](K01-focus.sh) | M1 | |
| K02 | [Zoom](K02-zoom.sh) | M1 | |
| K03 | [Runtime resize (H J K L) and reset (=)](K03-resize.sh) | M2 | |
| K04 | [Rename (t) and status bar toggle (s)](K04-rename-statusbar.sh) | M2 | |
| K05 | [Remapped prefix (C-b)](K05-prefix-ctrl-b.sh) | M1 | |
| K06 | [Status bar at the top](K06-statusbar-top.sh) | M2 | |

## L: Layout

| ID | Case | Milestone | Result |
|---|---|---|---|
| L01 | [Default 2,1 layout](L01-default-layout.sh) | M1 | PASS (layout, dividers, titles) · Terminal.app · 2026-09-26; resize not reported |
| L02 | [Alternative 2,2,1 layout](L02-two-two-one.sh) | M1 | PASS (rows re-proportion tall/short) · Terminal.app · 2026-09-26 |
| L03 | [Launch sizes: percent, cells, weights](L03-sizes.sh) | M1 | PASS (70/30 height, 30/70 top, 24c + 1fr:2fr bottom measured) · Terminal.app · 2026-09-26 |
| L04 | [Tree layout: tall left, stacked right](L04-tree.sh) | M1 | PASS · Terminal.app · 2026-09-26 |
| L05 | [Maximum: nine panes (3,3,3)](L05-nine-panes.sh) | M1 | PASS · Terminal.app · 2026-09-26 |
| L06 | [Window too small: minimum size + scrolling view (DP-1)](L06-small-window.sh) | M1 | PASS (view scrolls to focused pane, "more ◀" shown) · Terminal.app · 2026-09-26 |

## P: Packaging

| ID | Case | Milestone | Result |
|---|---|---|---|
| P01 | [Package: flags pass through](P01-passthrough.sh) | M2 | |
