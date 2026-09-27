# Testing dozer

dozer is tested in four layers. The first three are automatic and run in CI on Linux and macOS for every push to `main` and every pull request. The fourth is people using a real terminal, because some things (how Terminal.app draws a glyph, what a trackpad sends) only show up there.

| Layer | Command | What it covers | Time |
|---|---|---|---|
| 1. Unit tests | `make test` | Each package on its own: layout solving, config parsing and saving, packaging, the input router, mouse and key decoding, copy mode, pane lifecycle, and the emulator fork (wrap tracking, reflow) | seconds |
| 2. Race detector | `go test -race ./...` | The same tests, checking for unsafe concurrent access (panes are read and written from several goroutines) | ~10 s |
| 3. End-to-end smoke | `scripts/smoke.sh` | The real binary in a headless tmux window, driven by keystrokes and raw mouse reports, with the screen checked after each step (54 checks) | ~1 min |
| 4. Manual cases | `tests/cases/*.sh` | Scripted scenarios you run by hand on a real terminal and record as PASS/FAIL | minutes each |

```sh
make test && go test -race ./... && make cases && scripts/smoke.sh   # everything automatic
```

---

## 1–2. Unit tests and the race detector

- `make test` runs `go vet`, `go test ./...`, and the tests of the emulator fork in `third_party/vt` (a separate Go module, so the plain `go test ./...` doesn't reach it).
- Tests that start real processes (in `internal/pane`) use `/bin/sh` and take a few seconds, because they wait for automatic restarts and their back-off.
- Tests must not assume Linux: macOS has no `/bin/true`, `/proc`, or GNU tool flags. (A test that used `/bin/true` failed only on macOS; it now writes its own stand-in.)
- Benchmarks for the emulator: `make bench` (throughput of a flood of short lines and of log-like lines).

## 3. End-to-end smoke test

`scripts/smoke.sh` builds `bin/dozer`, starts it inside detached tmux sessions of a known size, sends keys (`tmux send-keys`) and raw SGR mouse reports, and checks the rendered screen (`tmux capture-pane`). It covers:

- the default layout, output with colors and wide characters, vim on the alternate screen, and exact cursor placement during line editing;
- the prefix key, focus, zoom, redraw, runtime resizing and its reset, renaming, and status bar toggling;
- `run:` panes dropping to a prompt after Ctrl-C;
- a 200,000-line flood, the quit confirmation, and pane sizes reaching the child (`stty size`), including after a window resize;
- dead panes (failed, killed, exited), restart, and dozer staying open;
- reflow (a long line survives shrink-then-grow), copy mode (the position tag, copying through OSC 52 into a tmux buffer, search);
- the mouse: click focus, divider drag, wheel history in and out, drag-to-copy, and a flood of 3,000 wheel events not blocking the keyboard;
- a packaged workspace: its name, relative `cwd`, and keyboard input.

Needs `tmux`, `vim` and `python3`. Each check prints `ok` or `FAIL` (most failures also print the screen); the script exits non-zero if anything failed. It's timing-based, so CI runs it a second time if the first attempt fails.

## 4. Manual test cases

`tests/cases/` holds one **runnable script per test case**. Each is a packaged dozer workspace (see [`PACKAGING.md`](PACKAGING.md)) that sets up exactly its scenario. Pane 1 (the guide) shows the steps and the expected result, taken from the workspace's `description`.

### Running a case

```sh
make install                             # dozer on your PATH (the cases run `dozer`)
tests/cases/K03-resize.sh                # run one
tests/cases/K03-resize.sh --show-config  # read it without running
```

**After pulling changes, run `make install` again,** not just `make`: the cases use the `dozer` on your `PATH`, and plain `make` only rebuilds `bin/dozer`. The status bar shows the version actually running (for example `v0.4.0-3-gabc1234`), so you can tell which build you tested.

Alternatively, point the cases at the repo's build without installing: `export DOZER_BIN=$PWD/bin/dozer`.

### Recording a result

Results live in [`tests/cases/README.md`](../tests/cases/README.md), one row per case:

```
| K03 | [Runtime resize …](K03-resize.sh) | M2 | PASS · Terminal.app · 2026-09-26 |
```

- **PASS**, **FAIL** or **PARTIAL**, the terminal app, the date, and a short note for anything unexpected.
- A screenshot is the most useful report for anything visual.
- If a later build fixes a failure, keep the history in the note (`… → fixed in c50eee3; retest: PASS`).
- **Deferred** marks a case whose verification is paused on purpose (for example copy/paste, SPEC DP-9).

### The case groups

| Prefix | Area | Examples |
|---|---|---|
| **C** | Config and commands | `run:` vs `exec:`, per-pane `cwd`/`env`, environment isolation |
| **D** | Dead panes and lifecycle | dead-pane states, restart, automatic restart and its limit, kill, quit |
| **E** | Terminal emulation | colors and Unicode, full-screen programs, output floods, line editing, paste |
| **H** | History and copy mode | scrollback, copying, search, resize reflow |
| **K** | Keys and controls | focus, zoom, resize, rename, status bar, a remapped prefix |
| **L** | Layouts | the default and `2,2,1` layouts, sizes, the tree form, 9 panes, a too-small window |
| **M** | Mouse | focus and border drag, wheel, selection, mouse-aware programs, mouse off |
| **P** | Packaging | flags passing through a package |

Each case also names the **milestone** that introduced what it tests (M0, M1, M2, M4).

### Writing a new case

A case is an ordinary dozer config with a `name:` like `K07-something` and a `description:` in a fixed shape. Package it into `tests/cases/`:

```yaml
name: K07-example
description: |
  K07: What this case checks  [M2]

  Steps:
    1. Do this.
    2. Then this.

  Expect:
    - What should happen.
    - What else should happen.

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
layout: "2,1"
panes:
  - title: guide
    run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR
  - title: subject
    exec: 'the thing under test'
  - title: shell
```

```sh
dozer --check -c k07.yaml                        # validate
dozer -c k07.yaml --package tests/cases/K07-example.sh
```

Then add a row to `tests/cases/README.md` under its group.

Conventions:

- **Pane 1 is the guide.** `less -FXR` shows the description and lets you scroll it if it's taller than the pane; `q` leaves the text on screen and gives you a prompt.
- **Write steps a person can follow without reading code:** name the keys (`Ctrl-a 2`), say where to look, and give expectations you can see on screen.
- **Keep it to one idea.** Several small cases beat one long one; a failure then points at one thing.
- **Label helper panes** with what they're for (`[3] paste here`), and number them in the title the way dozer numbers them.
- **No machine-specific paths.** Use `~`, `/tmp`, or relative paths; the case must run on anyone's machine.

### Changing a case

The `.sh` scripts are the source of truth:

```sh
tests/cases/K03-resize.sh --show-config > k03.yaml
$EDITOR k03.yaml
dozer -c k03.yaml --package tests/cases/K03-resize.sh
```

`make cases` validates every case (it runs `dozer --check` on each), and CI runs it too, so a case that stops parsing fails the build.

---

## Continuous integration

`.github/workflows/ci.yml` runs on every push to `main` and on pull requests:

1. On **Ubuntu and macOS**: `make test`, `go test -race ./...`, `make cases`, and `scripts/smoke.sh` (retried once).
2. Then it cross-compiles `dist/dozer-{darwin,linux}-{arm64,amd64}` and keeps them as a downloadable artifact of the run.
