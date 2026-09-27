# Changelog

dozer's version is `0.<milestone>.<fix>` until 1.0: the minor number follows the roadmap milestone the release completes (docs/SPEC.md §9). `dozer -version` and the status bar show the exact build, for example `v0.4.0` or, for a later commit, `v0.4.0-3-gabc1234`.

## v0.4.0: history and mouse (M4) · 2026-09-27

**Added**

- **Scrollback:** every pane keeps its history (`scrollback:`, default 10,000 lines).
- **Copy mode (`Ctrl-a [`):** browse history with vi keys; select by character (`v`) or line (`V`); search up and down with smart case (`/ ? n N`); copy with `y`. A soft-wrapped line copies as one line.
- **Clipboard:** copies go through OSC 52 (works over ssh in terminals that support it) and, locally, through `pbcopy` / `wl-copy` / `xclip` / `xsel`.
- **Resize reflow:** text rewraps when a pane gets narrower and comes back intact when it widens. Rows that no longer fit move into history instead of being cut off.
- **Mouse:** click to focus, drag a divider or title line to resize, the wheel scrolls history (or `less`, or mouse-aware programs), dragging selects within one pane. Programs that use the mouse (vim, htop) get their events in pane coordinates; Shift keeps an event for dozer. `mouse: false` or `--no-mouse` turns it off.
- `mouse:` and `scrollback:` config keys; `--no-mouse` flag.
- Documentation: a configuration guide (`docs/CONFIG.md`), packaging (`docs/PACKAGING.md`), the command line (`docs/CLI.md`) and testing (`docs/TESTING.md`); this changelog.
- 9 manual test cases (H01–H04, M01–M05); 13 more end-to-end checks (54).

**Fixed**

- A flood of wheel events (a trackpad flick) no longer locks up the keyboard: frames are rate-limited without blocking input, identical wheel events are merged, and input to a pane is queued.
- Wide characters at the right edge wrap as in xterm instead of being cut in half.
- Terminal.app ignores OSC 52, so copies now also use the local clipboard command.
- A packaging test failed on macOS (it relied on `/bin/true`).
- Stopping a `run:` command with Ctrl-C ended the pane under bash and zsh; it now drops to a prompt as intended.
- `make cases` printed a broken case's error but still succeeded; it now fails.

**Known limitations**

- Copy/paste works but doesn't feel like the OS's own; its verification is parked (SPEC DP-9).
- Mac verification of M4 is in progress: H01 passed; H03, H04, M01, M02, M04 and M05 are pending.

## v0.3.0: control, configs and packages (M2) · 2026-09-26

**Added**

- Runtime controls: resize a pane's border (`Ctrl-a H J K L`, repeatable) and reset sizes (`=`), kill a pane (`x`), rename (`t`), toggle the status bar (`s`).
- Dead panes: states (exited, failed, killed, connection lost), dimmed content, a banner and a status bar count; restart one (`r`) or all (`R`).
- Automatic restarts for `exec:` panes (`restart: on-failure | always`) with back-off and a limit (`max_restarts`, default 5).
- `status_bar: top | bottom | off`, `description:`, `quit_when_all_exited:`.
- `--save`: turn any command line and config into a YAML file.
- `--package`: a self-contained, shareable script with the config inside; `-c -` and `--config-text` for configs without a file.
- 27 manual test cases as packaged scripts; CI on Linux and macOS with a headless end-to-end smoke test.

**Fixed**

- Panes no longer inherit the outer terminal's identity (Terminal.app's "Restored session" in every pane).
- `cwd: ~` was ignored (a bare `~` is YAML's null).
- Killed panes say "killed" whatever exit status the program reports.

## v0.2.0: layouts (M1) · 2026-09-26

**Added**

- Several panes in a row/column layout: shorthand (`2,1`, `2,2,1`, …), `--heights`/`--widths`, and a YAML tree form for any arrangement.
- `-p` (run) and `-x` (exec) commands, piped commands, and a layout fitted to the number of commands.
- Focus (arrows, hjkl, 1–9, o), zoom, quit confirmation.
- YAML configs, profiles (`~/.config/dozer/NAME.yaml`, `./.dozer.yaml`) and `--check`; `examples/`.
- A too-small window keeps a minimum pane size and scrolls the view (DP-1).

## v0.1.0: spike (M0) · 2026-09-26

**Added**

- One pane in full screen on a real pseudo-terminal, with a swappable terminal emulator (a patched fork of `charmbracelet/x/vt`, SPEC DP-2).
- Raw keyboard pass-through with a single prefix key (`Ctrl-a`), and the program's key modes mirrored to the terminal.

**Fixed**

- The cursor lagged one keystroke behind while editing a command line.
- Emulator throughput was about 0.2 MB/s; fork patches brought it to about 3.7 MB/s.
