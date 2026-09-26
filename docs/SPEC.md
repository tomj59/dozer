# dozer — TUI Multi Shell · Specification

Status: Draft v0.12 · Owner: Tom · Last updated: 2026-09-26

## 1. Summary

dozer is a single-binary terminal application for Linux and macOS. It opens one full-screen text UI and runs several independent shells ("panes") inside it, arranged in a flexible row/column layout. Each pane is a real pseudo-terminal, so interactive programs (vim, htop, ssh, less) work inside it.

A bare `dozer` launches the default 3-pane layout with an interactive `$SHELL` in the current directory in every pane. Command-line flags, piped input, or a YAML config file can change the layout, pane sizes, and what each pane launches (an `ssh` session, a script, a `tail -f`, anything).

## 2. Decisions made

| Topic | Decision | Why |
|---|---|---|
| Architecture | **Standalone engine**: dozer owns the PTYs and emulates each pane's terminal itself | No tmux dependency, one binary, full control over UX and future features. Terminal emulation is the hard part, and good Go libraries exist for it. |
| Language | **Go** | Static single binary, trivial cross-compile for linux/darwin × amd64/arm64, strong PTY and TUI libraries, goroutine-per-pane fits the problem. |
| Interaction | **Fully interactive** panes | Every pane must behave like a real terminal: alt screen, colors, mouse-aware apps, resize. |
| Persistence | **No detach/reattach in v1** | Quitting dozer ends its child processes. The code keeps "core" and "UI" apart so a client/server split can be added later (§9). |
| Config format | **YAML** | Nested layouts read naturally; familiar from docker-compose and tmuxinator. |
| Keybindings | **Prefix key** (default `Ctrl-a`, configurable) | Never collides with programs running in a pane. |
| v1 extras | Mouse, scrollback, multi-input mode, pane titles + status bar | All chosen for v1. |
| Pane set | **Built once, at launch** | The pane set is fixed after startup: no adding, splitting or closing panes. Existing split sizes can still be adjusted at runtime (for example 30%\|70% → 50%\|50%). |
| Multi-input | **A modal input state** you enter and leave deliberately, like modes in vim | dozer warns once per session, the first time you enter the mode, and stays quiet after you accept. The warning can be turned off (§4.7). If you turn it off, dozer doesn't argue. |
| `--save` | **Saves configuration only**: layout, sizes, titles, flags, display prefs, and each pane's launch command. It never saves process state, history or remote session state. **Stubbed in v1.** | Only a handful of settings can change after launch (§4.8), so there's little runtime change to save. Revisit once real use shows the runtime settings are worth capturing. |
| Pane lifecycle | **The layout is permanent; dead panes stay put, are flagged loudly, and the user restores them** (§4.10) | dozer's job is to keep a multi-shell skeleton. An exit, crash or dropped ssh session must never collapse or rearrange the layout. Restoring is the user's call (`C-a r`); dozer's job is to make a dead pane impossible to miss. Automatic restart is opt-in per pane (`restart:`). |
| Quitting | **dozer quits only when you tell it to** (`C-a q`), never because panes exited, even the last one | One rule for every layout. A stray `exit` or a crash can't throw away your arrangement. A single-pane dozer is an outlier use, so it follows the same rule rather than getting an exception. `quit_when_all_exited: true` (config, and a flag) opts into the old behavior. |
| Config forms | **YAML is the working copy; a package is the release** (§4.12) | `--save` and YAML files are the editable, work-in-progress form you iterate on. `--package NAME` freezes a config into one self-contained `NAME.sh` that pipes its embedded YAML to `dozer -c -`. No other files are involved, and paths aren't tied to any machine. |
| Small terminals | **Provisional: panes keep a minimum size and the canvas scrolls** | Tune during functional review. See DP-1 in §10. |
| Name | **dozer** | Win-*dows* meets construction equipment. The binary, module and config paths all use it. |
| Pane borders | **Thin, shared one-line dividers** (tmux-style) | Maximizes pane space. Pane titles sit inline in the divider above each pane, and the focused pane's divider is highlighted. |
| Prefix key | `C-a` default, fully configurable (`prefix:` in YAML, `--prefix`) | Picked for familiarity from GNU screen. `C-a C-a` sends a literal `C-a` through. We'll adjust after real use. |
| `run:` start-up | `$SHELL -ic '<cmd>; exec $SHELL'` | Reliable, with no race against shell start-up. The cost is that the command isn't in shell history. See DP-4. |
| Platforms | Linux, macOS. Windows is out of scope. | `creack/pty` covers Unix PTYs. Windows ConPTY could come later behind the PTY interface. |

## 3. Why this is feasible (and where the work is)

Every pane needs three things: a **PTY** with a child process, a **VT emulator** that turns the child's output bytes into a screen grid, and a **compositor** that draws all the grids into one real terminal. The Go building blocks:

- **PTY:** `github.com/creack/pty`. Mature and widely used on Linux and macOS.
- **Screen output:** `github.com/charmbracelet/ultraviolet`. It provides a cell buffer plus a diffing renderer that emits only what changed. It uses the same cell type as the emulator, so a pane's screen copies straight into the host buffer. It replaced the original plan of `tcell`: tcell insists on owning keyboard input, and DP-2 chose raw input.
- **Input:** raw bytes from the host terminal (`golang.org/x/term` raw mode), routed by dozer's own small prefix state machine. See DP-2.
- **VT emulation:** `github.com/charmbracelet/x/vt`, carried as a lightly patched local fork (`third_party/vt`) until the fixes land upstream. It was chosen in the M0 spike over `hinshun/vt10x` (DP-2). Emulation is still where most bugs will live.

The rest (layout solving, key routing, borders, status bar) is ordinary application code.

## 4. User-facing behavior

### 4.1 Launch modes

```
dozer                                 # default layout, $SHELL in cwd in every pane
dozer -l 2,2,1                        # row shorthand: 2 panes, 2 panes, 1 pane
dozer -l 2,1 -p "ssh prod" -p htop    # commands fill panes in reading order; the rest get $SHELL
dozer -c ops.yaml                     # load a config file
dozer ops                             # load profile ~/.config/dozer/ops.yaml
printf 'ssh a\nssh b\nhtop\n' | dozer # piped: one pane command per line
dozer -p a -p b -p c -p d            # no -l: the layout is fitted to the commands (4 → 2,2)
dozer -c - < ws.yaml                  # config from stdin (YAML or JSON)
dozer --config-text '{"layout":"3"}'  # config as an argument
./my-tool.sh                          # a packaged workspace (§4.12)
```

Piped input is read to EOF, then dozer reopens `/dev/tty` for interactive input.

### 4.2 Built-in layouts

- **`default`** (alias `2,1`): top row has two equal panes, bottom row has one full-width pane. The top row takes 50% of the height.
- **`2,2,1`**: three rows. Heights split equally and adapt to the terminal size.
- Any row-count shorthand works: `1,3`, `3,3`, `4`, and so on.
- Sizing flags on top of the shorthand:
  - `--heights 60,40` sets the row heights.
  - `--widths 1:30,70` sets the column widths in row 1.
  - Units: `%` (default), `c` (fixed cells), `fr` (weight).

A "square" pane isn't guaranteed. Terminal cells are about 1:2 (width:height), so dozer sizes panes by the percentages given and doesn't try to force an aspect ratio.

### 4.3 How a pane runs its command

| Key | Meaning |
|---|---|
| `run: <cmd>` | Runs `$SHELL -ic '<cmd>; exec $SHELL'`: the command runs in an interactive shell, and when it ends you're back at a prompt in that pane. The command doesn't land in shell history (DP-4). **Default for `-p`.** |
| `exec: <cmd>` | The pane process *is* `$SHELL -lc "<cmd>"`. When it ends, the pane becomes a dead pane (§4.10). |
| (neither) | Plain interactive `$SHELL`. |

Per-pane options: `title`, `cwd`, `env`, `restart: never|on-failure|always` (exec only), `readonly` (ignores input unless unlocked), `group` (multi-input target group), `min: {w, h}`.

### 4.4 Default keybindings (prefix `C-a`)

| Keys | Action |
|---|---|
| `C-a ←↑↓→` / `h j k l` | Move focus |
| `C-a 1-9` | Focus pane N |
| `C-a o` | Focus the next pane |
| `C-a H J K L` | Move the focused pane's border left/down/up/right (tmux-style: the border on that side, or the opposite one at an edge). 2 columns / 1 row per press. Repeatable without the prefix for 0.7 s (status: RESIZE). Existing splits only. |
| `C-a =` | Reset split sizes to the launch layout |
| `C-a z` | Zoom or unzoom the focused pane |
| `C-a [` | Scroll/copy mode (vi keys, `/` search, `q` exits) |
| `C-a m` | Enter multi-input mode for the focused pane's group |
| `C-a M` | Enter multi-input mode for all panes |
| `C-a Esc` | Leave multi-input mode |
| `C-a r` | Restart the focused pane's process with its launch command (works on live panes too) |
| `C-a R` | Restart every dead pane |
| `C-a x` | Kill the pane's process (confirm). The pane stays, as a dead pane. |
| `C-a t` | Rename the pane title |
| `C-a s` | Toggle the status bar |
| `C-a C-l` | Redraw the whole screen (for example after the host terminal's Cmd-K cleared it). Moved from `C-a l` in M1, because `l` is focus-right. |
| `C-a Esc` (while prefix is pending) | Cancel the prefix |
| `C-a ?` | Help overlay |
| `C-a q` | Quit dozer (confirm) |
| `C-a C-a` | Send a literal `C-a` to the pane |

All of these can be remapped in the config.

### 4.5 Mouse

> **Status (M1):** not implemented. With several panes, raw mouse reports must be translated to the pane under the pointer, so host mouse mirroring is switched off until M4. Terminal text selection works normally in the meantime.


- Click a pane to focus it.
- Drag a border to resize.
- Scroll the wheel to enter scrollback.
- If the program in a pane has turned on mouse reporting (vim, htop), mouse events inside that pane pass through to it instead. `Shift` forces dozer to handle the event.
- Mouse support can be turned off entirely: `mouse: false`.

### 4.6 Status bar and titles

- Borders are thin one-line dividers shared between neighboring panes. Each pane's divider shows `[N] title`. The default title is the command, or an OSC-set title if the program sets one. Exit codes are shown too.
- The focused pane has a highlighted border.
- A one-line status bar shows the profile name, the current input mode, the prefix-pending indicator, zoom state, a **dead-pane count** (for example `✖ 2 dead · C-a R`) when any pane is dead, and the clock.

### 4.7 Input modes

dozer has an explicit input-mode state machine. Only one mode is active at a time.

| Mode | Where keystrokes go | Enter | Leave |
|---|---|---|---|
| **Normal** | Focused pane only | (default) | — |
| **Prefix** | dozer command table (a single key) | `C-a` | Automatically, after one key or a timeout |
| **Multi-input** | Every non-readonly pane in the target set (a group, or all panes) | `C-a m` / `C-a M` | `C-a Esc` only. There's no timeout and no automatic exit. |
| **Scroll/copy** | Scrollback navigation for the focused pane | `C-a [` | `q` / `Esc` |

Multi-input rules:

- **First entry:** the first time you enter multi-input in a session, dozer shows a one-time confirmation listing the target panes. After you accept, you're never prompted again that session.
- **Turning the warning off:** `--suppress-multi-warn` or `multi_input: { warn: false }` in YAML removes the confirmation entirely.
- **While active:**
  - The target panes get a distinct border color.
  - The status bar shows a persistent `MULTI →N` badge.
  - `Esc` itself is sent to the panes, because vim in the targets needs it. Only `C-a Esc` leaves the mode.
- **Delivery:** keystrokes are delivered to each target in order, byte for byte, with no per-command prompts.

### 4.8 Runtime control surface (the "control panel")

This is every setting that can change after launch, as currently specified. **It's the complete list.** If it isn't in this table, it's fixed at launch.

| # | Setting | How to change it | Saveable? |
|---|---|---|---|
| 1 | **Split sizes** | `C-a H J K L`, `C-a =` to reset (M2); drag a border with the mouse (M4) | **Yes.** The only substantial one. |
| 2 | **Pane titles** | `C-a t` | **Yes** |
| 3 | **Status bar on/off** | `C-a s` | **Yes** (display pref) |
| 4 | Zoomed pane | `C-a z` | No. It's view state. |
| 5 | Focused pane | Arrows, `1-9`, mouse click | No. It's view state. |
| 6 | Input mode (multi-input, scroll) | `C-a m` / `M` / `Esc` / `[` | No. It's transient by design. |
| 7 | Multi-input warning, accepted this session | Accepting the prompt | No. It resets each session on purpose; use the flag or YAML to turn it off for good. |
| 8 | Viewport pan (DP-1) | `C-a PgUp/PgDn/Home/End`, mouse wheel | No. It's view state. |
| 9 | Pane process running/exited | `C-a r` / `C-a x` | No. It's process state. |

**Fixed at launch:**

- The pane set and layout shape
- Each pane's command (`run`/`exec`), shell, cwd, env, and group membership
- Prefix key and keybindings
- Mouse on/off
- Scrollback length
- `min_pane`
- The multi-input warning flag

**Result:** there are three saveable controls (sizes, titles, status bar) plus view-state toggles. That matches the "2 levers and 1 knob" case, so `--save` stays small.

**Rule:** a runtime control is added only because it's useful while working, never to give `--save` more to do. When a new one is added, this table gets a row.

### 4.9 `--save` (stubbed)

Scope, now and later: **configuration only**. dozer never tries to restore process, history, shell or remote-session state.

- **v1 (stub), implemented in M2:** `dozer <flags> --save <file.yaml>` resolves the launch inputs (CLI flags, piped commands, config file, defaults) into one canonical YAML file and exits without launching. This is useful on its own, because it turns a long command line into a profile. `--save -` writes to stdout.
- **Later (runtime save):** `C-a S` writes the same YAML with the current values of controls 1–3 from §4.8 merged in. This is deferred until real use shows runtime tuning is worth capturing.
- Both paths use the same serializer. The only difference is whether the runtime deltas get merged in.

### 4.10 Pane lifecycle and dead panes

A pane's **slot** in the layout is permanent for the whole session. Only the **process** inside it comes and goes.

| State | Meaning | How it looks |
|---|---|---|
| **Running** | The process is alive | Normal divider and title |
| **Exited** | The process ended with status 0 | Dimmed divider; title `[N] title · exited` |
| **Failed** | Non-zero status, or killed by a signal | Red divider; title `[N] title · ✖ exit 1` or `✖ SIGKILL` |
| **Disconnected** | Heuristic: the command was `ssh …` and it exited 255 | Red divider; title `[N] title · ✖ connection lost` |

When a pane dies:

1. Its last screen stays visible but dimmed, so you can see what happened (the error message, or the ssh "Connection closed" line).
2. A one-line banner at the bottom of the pane reads: `✖ exited 1 at 14:02:31 · C-a r restart`.
3. The status bar shows the dead-pane count, so a dead pane outside your focus (or zoomed out of view) is still noticed.
4. Nothing else changes: neighbors keep their size and the layout keeps its shape.

**Restoring:**

- `C-a r` re-runs the pane's *launch* command (its `run:`/`exec:`, cwd and env) in the same slot at the same size.
- `C-a R` restores every dead pane at once.
- The restarted pane starts on a fresh screen; the dead output is discarded (scrollback, M4, may keep it later).

**Automatic restart** (`restart: on-failure|always`) stays opt-in per pane, for panes like `tail -F` or a dashboard. Automatic restarts use a back-off (1 s, 2 s, 4 s … capped at 30 s), so a crash-looping command can't spin.

**Quitting:** dozer never exits because panes died, even when all of them are dead. It shows every dead pane and waits for `C-a r`/`C-a R`/`C-a q`. `C-a q` asks for confirmation (§4.4) when any pane is still running. Restoring the whole arrangement after quitting dozer means relaunching with the same flags or profile. That's exactly what a profile or `--save` (§4.9) is for.

> **M0 note:** the spike still quits when its single shell exits. This section is implemented in M2.

### 4.11 Styles and themes (planned)

**Goal:** each pane can look like a different terminal profile, for example pane 1 in a green-on-black "Homebrew" style and pane 2 in "Classic" black-on-white. The chrome (dividers, titles, the status bar and dead-pane signals) has a theme of its own.

There are two levels:

| Level | Controls | Example YAML |
|---|---|---|
| **Pane style** | Default foreground and background, the 16-color ANSI palette, cursor color, bold-as-bright | `panes: [{style: homebrew}, {style: classic}]` or inline `style: {fg: "#28fe14", bg: "#000000"}` |
| **Chrome theme** | Divider, title, focused-title, dead-pane (exited/failed) colors, banner, status bar | `theme: {focus: cyan, failed: red, …}` |

Built-in pane styles would be named after familiar looks (`homebrew`, `classic`, `pro`, `solarized-dark` …) using dozer's own palette values. The default style is "inherit": the host terminal's own colors, which is today's behavior.

**How it would work (DP-5):**

- dozer rewrites colors when it composes the frame; programs in the pane don't know about the style.
- Cells with no explicit color get the pane's fg/bg.
- ANSI colors 0–15 are remapped through the pane's palette.
- 256-color and 24-bit colors pass through unchanged, as terminals do.
- The emulator answers color queries (OSC 10/11/4) with the pane's style, so programs that detect light or dark backgrounds (vim, bat, delta) adapt per pane.
- The cursor color follows the focused pane (OSC 12 on focus change).

**Dead-pane signals stay readable on any style.** Exited and failed markers are chrome-theme colors drawn on the title line and banner, which dozer owns, never inside the pane's colors. Dimming a dead pane uses the faint attribute, which works on any background. A style that would make the signal hard to see, such as a red-background pane, is the chrome theme's problem to solve (for example with a contrasting banner), not the pane's.

Scheduled after M5; see the roadmap. Nothing is exposed in the config yet.

### 4.12 Packaged workspaces (`--package`)

`dozer <flags or -c file> --package my-tool` writes **`my-tool.sh`**, one self-contained POSIX `sh` script. **The script is the whole package.** It carries its configuration inside, and running it reads and writes no other files. It does three things:

1. **Checks the environment:** a `dozer` binary is on `PATH` (or at `$DOZER_BIN`), and it's running in a terminal.
2. **Checks for dozer:** if dozer is missing, it prints where to get it and exits 127.
3. **Runs:** `yaml | dozer -c - "$@"`. The embedded YAML is piped into dozer, which is the same as `dozer -c - < config.yaml`. dozer then reads the keyboard from the terminal (`/dev/tty`).

| Command | Does |
|---|---|
| `./my-tool.sh` | Launches the workspace |
| `./my-tool.sh --prefix C-b` | Passes extra flags through to dozer |
| `./my-tool.sh --show-config` | Prints the embedded YAML |
| `dozer --check -c - < my-tool.sh`, `dozer -c my-tool.sh` | dozer also reads a package as a config source |

To change a package: `./my-tool.sh --show-config > my-tool.yaml`, edit, then `dozer -c my-tool.yaml --package my-tool.sh`.

**No filesystem assumptions:**

- The script never refers to its own location.
- cwd and env are embedded **exactly as written** (`sub`, `~/work`, `$HOME`), not as expanded on the packaging machine, and they resolve when the script runs.
- A relative `cwd` resolves against the folder you run the script from. `~` and `$VARS` resolve for whoever runs it.
- Only absolute paths the author wrote stay absolute.

**Inline equivalents** (the same config-without-a-file path the package uses):

- `dozer -c -` reads YAML (or JSON, which is a subset of YAML) from stdin.
- `dozer --config-text '{"layout":"2,1","panes":[{"run":"htop"}]}'` takes it as an argument.

**Safety:** `--package` overwrites only files that are already dozer packages; it refuses to overwrite any other script. The YAML sits in a quoted heredoc, so the shell never expands it.

**Dependency:** dozer itself. See DP-6 for making packages carry it.

## 5. Configuration (YAML)

Locations: `-c <file>`, otherwise `./.dozer.yaml`, otherwise `~/.config/dozer/config.yaml` (XDG). CLI flags override the file.

```yaml
version: 1
name: ops
shell: /bin/zsh            # default: $SHELL
cwd: ~/work                # default: current directory
env: { STAGE: prod }
prefix: C-a
mouse: true
scrollback: 10000          # lines per pane
status_bar: bottom         # top | bottom | off

# Shorthand form:
#   layout: "2,2,1"
#   panes: [ {run: "ssh a"}, {run: "ssh b"}, ... ]   # reading order

# Full tree form:
layout:
  rows:
    - size: 60%
      cols:
        - { pane: api, size: 50% }
        - { pane: db }
    - { pane: logs }

multi_input:
  warn: true               # false = never confirm (same as --suppress-multi-warn)

min_pane: { w: 20, h: 5 }  # provisional default, see DP-1
quit_when_all_exited: false  # default: dozer stays open with dead panes (§4.10)

panes:
  api:  { title: API,  run: "ssh api-1", group: hosts }
  db:   { title: DB,   run: "psql", cwd: ~/db }
  logs: { title: Logs, exec: "tail -F /var/log/app.log", readonly: true, restart: always }

keys:                      # optional overrides
  zoom: z
  quit: q
```

Validation errors are reported with the line number. `dozer --check <file>` validates a file and prints the resolved layout and panes without launching anything.

**Implemented so far (M1):**

- Top-level keys: `version`, `name`, `description` (free text: shown by `--check`, in package headers, and to every pane as `$DOZER_DESCRIPTION`), `shell`, `cwd`, `env`, `prefix`, `layout` (shorthand or tree), `heights`, `widths` (sizes also accept `auto`), `panes` (list or named map), `min_pane`, `quit_when_all_exited`, `status_bar` (M2).
- Pane keys: `title`, `run`, `exec`, `cwd`, `env`, `shell`, `restart`.
- A bare string in the pane list is shorthand for `run:`.

Keys from later milestones are accepted but produce a warning in `--check` (they're ignored for now):

- top level: `mouse`, `scrollback`, `multi_input`, `keys`
- pane: `readonly`, `group`, `min`

Unknown keys are errors. Working examples ship in `examples/`, and every one is loaded by the test suite.

## 6. Architecture

```
            ┌──────────── core (UI-independent) ─────────────┐
 CLI/YAML → │ Config → Session{ LayoutTree, Panes[] }        │
            │   Pane = PTY + child proc + VT emulator +      │
            │          scrollback ring + meta(title, state)  │
            └──────────────▲──────────────────┬──────────────┘
                   commands│ (focus, resize,  │ events (dirty, exit,
                   input,  │  multi-in...)    │  title, bell)
            ┌──────────────┴──────────────────▼──────────────┐
            │ UI: InputRouter (mode FSM, mouse, multi-in)    │
            │     Renderer (layout → canvas → viewport,      │
            │     borders, status bar, damage tracking)      │
            └──────── ultraviolet renderer · raw stdin ───────┘
```

Packages:

- `cmd/dozer`: entry point
- `config`: load, merge, validate
- `layout`: tree, solver, presets, shorthand parser
- `pane`: PTY, process, emulator adapter, scrollback
- `session`: pane set, focus, multi-input groups
- `input`: input-mode state machine (normal, prefix, multi, scroll), key encoding
- `render`: compositor
- `ui`: event loop

Concurrency: one reader goroutine per PTY feeds its emulator (guarded by a mutex) and sends a dirty signal. The renderer coalesces redraws to at most about 60 fps, so a pane printing a flood of output can't starve input.

Layout tree is the source of truth. The launch layout is parsed into an immutable **pane set** plus a mutable **split-size tree**. Runtime resizing changes only the sizes in that tree, never its shape. The tree serializes straight back to the §5 YAML, and that's what makes `--save` cheap. `C-a =` restores the launch sizes.

Resize: a terminal resize makes dozer re-solve the layout, then call `pty.Setsize` and resize the emulator for each pane. Each child process gets `SIGWINCH` from its PTY.

Input fidelity (decided in M0, DP-2): the prefix state machine sees raw input bytes and passes everything else through untouched; keys are never decoded and re-encoded. So the bytes arrive encoded the way the focused program expects, dozer **mirrors that program's modes onto the host terminal**:

- cursor-key mode (DECCKM) and keypad mode
- bracketed paste
- focus events
- mouse tracking

Bracketed-paste content is never scanned for the prefix. Mouse reports will need translating from screen to pane coordinates once there's more than one pane (M1/M4).

Environment for child processes: `TERM=xterm-256color`, `COLORTERM=truecolor`, `TERM_PROGRAM=dozer`, `TERM_PROGRAM_VERSION`, `DOZER=1`, `DOZER_PANE=<id>`, `DOZER_NAME`, `DOZER_DESCRIPTION`. Variables that identify the *host* terminal are removed: `TERM_SESSION_ID`, `ITERM_SESSION_ID`, `LC_TERMINAL`, `TMUX`, `STY`, and the kitty, WezTerm, Alacritty, VTE, Windows Terminal, Ghostty and Konsole ones. Inside a pane, dozer is the terminal. (Found in Mac testing: Terminal.app's `TERM_SESSION_ID` made every pane's zsh restore and save the same window session.)

## 7. Extensibility points

These are designed in from the start, even where v1 uses only one implementation:

- **`PaneSource` interface**: local PTY now. Later: remote or container execution, and Windows ConPTY.
- **Layout preset registry**: user-defined named layouts in the config.
- **Action registry**: every keybinding maps to a named action, so a future control socket or scripting layer can call the same actions.
- **Core/UI message boundary**: the path to detach/reattach (§9).
- **Hooks**: `on_start`, `on_exit` per pane (v1.x).

## 8. Non-goals (v1)

- Detach/reattach, and sessions that outlive the dozer process
- Windows
- Tabs or multiple windows
- Creating, splitting or closing panes at runtime. The layout is a one-time construction event at launch; resizing existing splits is in scope.
- Plugin runtime
- Built-in SSH client (panes just run the system `ssh`)

## 9. Roadmap

| Milestone | Scope |
|---|---|
| **M0 Spike** ✅ (Mac: basics, cursor fix confirmed; sections 2–3 pending) | One pane in full screen, driven through the chosen emulator. Pass criteria: vim, htop, and `less` all work, resize is correct, `cat` of a large file stays smooth. Decide the emulator library and the input approach. |
| **M1 Layouts** (Linux ✅, Mac pending) | Layout tree and solver, `default` and `2,2,1` presets, `-l`, `--heights`/`--widths`, `-p`/`-x` (repeatable), piped input, thin dividers with titles. **Pulled forward:** focus (arrows/hjkl/1-9/o), zoom, quit confirm, dead panes with `C-a r`/`R` (from M2); the YAML config subset, profiles and `--check` (from M3); `examples/`. DP-1 viewport, auto-follow only. |
| **M2 Control** (Linux ✅, Mac pending) | Prefix FSM, focus, zoom, keyboard resize, quit (with confirm), kill, restart. Dead-pane states, chrome, banner and status count (§4.10); `C-a r`/`C-a R`; `quit_when_all_exited`. |
| **M3 Config** | YAML schema, profiles, `--check`, `run`/`exec`, per-pane options, titles, status bar. |
| **M4 History & mouse** | Scrollback ring, scroll/copy mode with search, OSC 52 clipboard, mouse focus/resize/scroll and passthrough. |
| **M5 Multi-input** | Input-mode state machine, groups, one-time warning plus its suppress flag, visual indicators, readonly panes. |
| **M3 (add-on)** ✅ (done in M2) | `--save` stub: resolve the launch inputs into canonical YAML (§4.9). It shares the YAML serializer with `--check`. |
| **M3 (add-on)** ✅ | `--package NAME`: packaged workspace scripts (§4.12). |
| **Deferred** | `C-a S` runtime save of sizes, titles and status bar. Revisit after functional review. |
| **M6 Ship** | goreleaser, Homebrew tap, `dozer --help`/man page, example configs. |
| **Manual test cases** ✅ | `tests/cases/*.sh`: one packaged workspace per test case, with a guide pane that shows its steps and expectations; indexed in `tests/cases/README.md`, validated by `make cases` and the unit tests. |
| **Functional review** | Try the small-terminal behavior (DP-1) and multi-input ergonomics with real use, then adjust. |
| **Styles** (after M5) | Pane styles and chrome themes (§4.11, DP-5): built-in named styles, inline colors, OSC color query replies, per-focus cursor color. |
| **Later** | Pane output logging, control socket (`dozer send`), detach via a client/server split, and possibly runtime split/close. |

## 10. Architectural decision points

This is a running log of the moments where one choice leads down a distinctly different road. Each entry records the provisional choice, what it commits the code to, and when it has to be settled. **Escalate** means the choice gets called out for a "do this, not that" discussion before we commit to it.

### DP-1 · Terminal too small for the layout

**Status:** provisional. Revisit at functional review.

**Provisional behavior (scrolling, per guidance):**

1. Each pane has a minimum size: `min_pane`, default 20×5 cells, which a pane's `min` can override.
2. The layout solver never shrinks a pane below its minimum. If the terminal is smaller than the layout's total minimum, the solver produces a **virtual canvas** bigger than the screen.
3. The screen is a **viewport** onto that canvas. It auto-scrolls to keep the focused pane fully visible. `C-a` + `PgUp/PgDn/Home/End`, or the mouse wheel over the border gutter, pans the viewport manually. Scroll indicators (`◀ ▲ ▼ ▶`) show hidden content.

**Implemented in M1:** points 1–5, except manual panning. The viewport follows focus, and the status bar shows `more ◀▲▼▶`.
4. A pane's PTY size is its layout size, not what's currently visible. Partly hidden panes keep their real size, so programs inside don't reflow while you pan.
5. Zoom (`C-a z`) still works as a manual escape hatch.

**What this commits the code to (decide by M1):**

- The compositor renders into an off-screen canvas with a viewport offset. It does not draw straight to screen coordinates.
- All mouse hit-testing goes through a viewport→canvas transform.
- The layout solver's output type is `canvasSize + rects`, not just `rects`.

Doing this in M1 is cheap. Retrofitting it later touches the renderer, mouse input and layout.

**Alternative roads, if review rejects scrolling:**

- (a) Auto-zoom the focused pane and minimize the others to one-line title strips.
- (b) Shrink panes below their minimum and let the programs inside cope.

Both fit the canvas design, since (a) and (b) are just different solver policies. So choosing scrolling now doesn't lock us in. That's why no escalation is needed yet.

### DP-2 · Emulator library and input path (M0)

**Status:** decided on Linux; waiting on Mac confirmation (M0 checklist).

**Decision:**

- **Emulator:** `charmbracelet/x/vt`, behind the `internal/emu` adapter.
- **Input:** raw-byte passthrough plus mode mirroring.
- **Output:** the ultraviolet renderer.

`hinshun/vt10x` stays selectable with `--emu vt10x` for A/B testing during M0 and will be removed in M1.

**Why charm vt over vt10x (measured in the spike):**

| | charm vt | vt10x |
|---|---|---|
| Maintenance | Active (commits in 2026-09) | Last commit 2022-03 |
| Wide chars / graphemes | Yes | No (one rune per cell) |
| 24-bit color | Correct | Collides with the 256 range: `RGB(0,0,5)` renders as palette color 5 (magenta) |
| Underline styles (curly etc.) | Yes | Dropped |
| Scrollback | Built in | None |
| Mode callbacks (for mirroring) | Yes | Polling only; no bracketed paste |
| Raw throughput (1 MB of short lines, 120×40 pane) | 3.6 MB/s after patches | 9.8 MB/s |
| End to end, `seq 1 1000000` in dozer (cloud VM) | 2.5 s | 1.2 s (bare tmux: 0.7 s) |

**The fork (heads-up rather than escalation):** as shipped, charm vt scrolled at about 0.2 MB/s, so `seq 1 1000000` took more than a minute. Profiling found two O(n) hot paths:

- Every line feed copied every cell of the screen.
- A full scrollback shifted all of its lines on each new line.

Small patches (about 50 lines; three for speed, one for a data race, recorded in `third_party/vt/DOZER_PATCHES.md`) make it about 20× faster (0.18 → 3.7 MB/s). They're wired in with a `replace` directive, and the upstream tests still pass. The plan is to send them upstream and drop the fork. If upstream rejects them, carrying the fork is cheap.

**Watch items:**

- Throughput is still about 2× behind vt10x. The remaining cost is per scroll: blank-filling the new line, trimming the scrolled-off line, and marking lines as changed. A deeper fix (lazily created blank lines, a compact scrollback) can come later if real use needs it.
- Scrollback memory: every cell is about 112 bytes, so scrollback is sized by content, not pane width. Still, 10,000 lines of wide colored output per pane can reach tens of MB. Revisit with the M4 scrollback work.
- ultraviolet and x/vt are pre-1.0 (pseudo-versions), so API churn is possible. The adapter layer contains it.
- **Found in Mac testing (fixed):** the host cursor lagged one keystroke behind the shell's, which made line editing drift and scramble. The cause is in `uv.TerminalScreen.Flush`: it queues the cursor move *after* writing out the frame, so each move went out one frame late. dozer now drives `uv.TerminalRenderer` directly (`internal/host`), and `scripts/smoke.sh` checks the cursor after each edit. Report upstream.

### DP-3 · Core/UI boundary (M1)

**Status:** decided in principle. Core and UI talk only through commands and events (§6). It costs a little ceremony now and keeps detach/reattach possible later. **Escalate** if the boundary starts to cost real performance in the render path.

### DP-4 · How `run:` starts its command

**Status:** decided for v1; revisit at functional review.

`run:` uses `$SHELL -ic '<cmd>; exec $SHELL'`. The alternative, typing the command into a live shell after it starts, puts the command in history but races with shell start-up because there's no reliable "prompt ready" signal. If real use shows history matters, add `run_mode: type` with a start-up delay as an opt-in. It needs no structural change either way.

### DP-5 · Who owns a pane's colors

**Status:** decided in principle, to support §4.11. Cheap to honor now, expensive to retrofit.

dozer is the color authority. The compositor maps each cell's *semantic* color (default / ANSI 0–15 / 256 / RGB) to what is drawn, per pane. This commits the code to:

1. **Keep cell colors semantic end to end.** The emulator adapter must never flatten "default" or "ANSI red" into RGB. charm vt already keeps `nil`, `BasicColor`, `IndexedColor` and `RGB` distinct.
2. **Style the chrome from named theme tokens,** not hard-coded colors. In M1 the colors are constants in `internal/app/render.go` (dim, focus, fail); they become a theme struct when §4.11 lands.
3. **Keep pane styling in one place:** the compositor's cell copy (plus OSC replies in the emulator). Never in the pane or PTY layer.

**Alternative rejected:** emitting OSC 10/11/4 to the host terminal on focus change. The host can hold only one palette at a time, so unfocused panes would render in the wrong style.

### DP-6 · Should packages carry dozer itself?

**Status:** open. v1 packages need dozer on `PATH` (§4.12). That keeps scripts tiny (under 2 KB), readable, and diffable in git.

If sharing with people who don't have dozer turns out to matter, there are two roads:

| Option | Pros | Cons |
|---|---|---|
| **(a) Embed binaries** (`--package --standalone`): base64 dozer builds for darwin/linux × arm64/amd64 inside the script, extracted to `~/.cache/dozer/<version>/` on first run | Truly self-contained; works offline | About 12 MB per script (4 × 3 MB); opaque; every package pins a dozer version |
| **(b) Bootstrap**: the script downloads the matching dozer release from GitHub on first run | Small script; version-pinned | Needs network and published releases (M6); a download-and-run step some users won't accept |

A version pin is also worth considering: the script records the dozer version that made it and warns on a mismatch. **Escalate** when M6 packaging (releases, Homebrew) is designed, because (b) depends on it.

## 11. Risks

| Risk | Mitigation |
|---|---|
| Emulator fidelity (edge-case escape sequences, wide or emoji characters) | M0 spike with a real-app checklist; the adapter layer makes the library swappable |
| Heavy output floods the UI | Coalesced rendering (≤60 fps); patched emulator hot paths (DP-2). Emulator throughput is the current bottleneck, not rendering. |
| Key encoding loss (Alt/Meta, modified arrows, macOS Option key) | Raw-byte passthrough; document macOS "Option as Meta" terminal settings |
| Differences between host terminals | Test on iTerm2, Terminal.app, kitty, Alacritty, WezTerm, GNOME Terminal, and inside tmux/ssh |
| Name clash | Resolved: the project is named **dozer** (a play on "Win-dows" and construction equipment) |

## 12. Open questions

1. Resolved in v0.4: the name is **dozer**.
2. Resolved in v0.3: `--save` covers configuration only (no working directories, history or process state) and starts as a stub (§4.9). The runtime control surface is listed in full (§4.8).
3. Resolved in v0.2:
   - Runtime splitting: not in v1.
   - Multi-input safety: a modal state with a one-time warning.
   - `--save`: stretch goal.
   - Minimum size: provisional scrolling (DP-1).
