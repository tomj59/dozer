# Manual test cases

Each case is a **packaged dozer workspace**: one self-contained script that sets up exactly what the test needs. A **guide** pane (pane 1) shows the steps and the expected result. If the text is taller than the pane, it opens in `less`; scroll, then `q` leaves it on screen.

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
| C02 | [Per-pane cwd and env](C02-cwd-env.sh) | M1 | |
| C03 | [Panes don't inherit the host terminal's identity](C03-env-isolation.sh) | M2 | Partial: no 'Restored session' seen in D03 · run C03 for the env check |

## D: Dead panes & lifecycle

| ID | Case | Milestone | Result |
|---|---|---|---|
| D01 | [Dead-pane states and chrome](D01-dead-states.sh) | M1 | PASS · Terminal.app · 2026-09-26 |
| D02 | [Restoring dead panes (r / R)](D02-restore.sh) | M1 | PASS · Terminal.app · 2026-09-26 |
| D03 | [Automatic restart: back-off and the retry limit](D03-auto-restart.sh) | M2 | PASS · Terminal.app · 2026-09-26 (back-off + retry limit) |
| D04 | [Kill a pane (x)](D04-kill.sh) | M2 | |
| D05 | [quit_when_all_exited: true](D05-quit-when-all-exited.sh) | M1 | |
| D06 | [Quit confirmation](D06-quit-confirm.sh) | M1 | |

## E: Terminal emulation

| ID | Case | Milestone | Result |
|---|---|---|---|
| E01 | [Colors and Unicode](E01-color-unicode.sh) | M0 | |
| E02 | [Full-screen programs side by side](E02-fullscreen-apps.sh) | M0 | |
| E03 | [Heavy output in several panes](E03-flood.sh) | M0 | |
| E04 | [Shell line editing and history](E04-line-editing.sh) | M0 | |
| E05 | [Bracketed paste and Option/Meta keys](E05-paste-and-meta.sh) | M0 | |

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
| L01 | [Default 2,1 layout](L01-default-layout.sh) | M1 | |
| L02 | [Alternative 2,2,1 layout](L02-two-two-one.sh) | M1 | |
| L03 | [Launch sizes: percent, cells, weights](L03-sizes.sh) | M1 | |
| L04 | [Tree layout: tall left, stacked right](L04-tree.sh) | M1 | |
| L05 | [Maximum: nine panes (3,3,3)](L05-nine-panes.sh) | M1 | |
| L06 | [Window too small: minimum size + scrolling view (DP-1)](L06-small-window.sh) | M1 | |

## P: Packaging

| ID | Case | Milestone | Result |
|---|---|---|---|
| P01 | [Package: flags pass through](P01-passthrough.sh) | M2 | |
