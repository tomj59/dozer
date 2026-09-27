# dozer quickstart

This page teaches you to use dozer in about ten minutes: panes, the command key, layouts, scrollback and the mouse. When you want to build your own workspaces, go on to the [configuration guide](CONFIG.md).

## What you're looking at

dozer fills your terminal window with several **panes**. Each pane is a real, independent shell (or a command you chose), arranged in rows and columns and separated by thin lines:

```
─ [1] zsh ─────────────────────────┬─ [2] git log ────────────────────────   ← title line of each pane
tomj@mac ~/dozer %                 │* db5558e Add QUICKSTART …
                                   │* f7f9ee3 M0 spike …
                                   │tomj@mac ~/dozer %
─ [3] zsh ─────────────────────────────────────────────────────────────────
tomj@mac ~/dozer %

 dozer │ [1] zsh                                        C-a q quit │ v0.4.0   ← status bar
```

- **Title line:** the pane number, its label, and its state when the process has ended. The pane you're typing into (the **focused** pane) has a bright cyan title.
- **Status bar:** which pane is focused, a red alert when a pane has died, and hints.

The default layout is `2,1`: two panes on top, one wide pane below. Every pane starts your shell in the folder you launched from.

## 1. Install it

From the dozer folder (needs Go 1.24 or later):

```sh
make install                  # builds dozer and copies it to /usr/local/bin
make install PREFIX=~/.local  # or into ~/.local/bin
```

The examples below run `dozer` from your `PATH`. (`make` alone builds `bin/dozer`; use `bin/dozer` in the examples if you haven't installed it.)

## 2. Start it and move around

```sh
dozer
```

You get three shells. Typing goes to the focused pane: pane `[1]`, top left. To use the others you need the **prefix key**.

### The prefix key: `Ctrl-a`

Every shell needs every normal keystroke, so dozer reserves exactly one key, **`Ctrl-a`**, and listens for a single command key right after it. The status bar shows **PREFIX** while it waits.

| After `Ctrl-a`, press | What happens |
|---|---|
| `→` `←` `↑` `↓` (or `l` `h` `k` `j`) | Focus the pane in that direction |
| `1` … `9` | Focus pane N |
| `o` | Focus the next pane |
| `z` | **Zoom**: the focused pane fills the window. `Ctrl-a z` again restores the layout. |
| `H` `J` `K` `L` | **Resize**: move the focused pane's border left, down, up or right. Keep pressing the letter (no prefix needed) while the status bar says **RESIZE**. |
| `=` | Reset all sizes to how the layout launched |
| `[` | **Copy mode**: scroll back through the pane's history, select and copy (§4) |
| `t` | Rename the focused pane (type, then Enter; Esc cancels; empty restores the default) |
| `s` | Hide or show the status bar |
| `x` | Kill the focused pane's process (asks first). The pane stays, dead, until you restart it. |
| `r` | **Restart** the focused pane's command (works whether it's dead or alive) |
| `R` | Restart **every dead** pane |
| `Ctrl-l` | Redraw everything (if the screen ever looks wrong) |
| `q` | **Quit dozer.** If any pane is still running, it asks first: `y` quits, any other key stays. |
| `Ctrl-a` | Send one real `Ctrl-a` to the shell (jump to the start of the line) |
| `Esc` | Cancel |

Try it now:

1. `Ctrl-a` `→` → pane `[2]`'s title turns cyan and your typing goes there.
2. `Ctrl-a` `↓` → pane `[3]`.
3. `Ctrl-a` `z` → pane 3 fills the window; `Ctrl-a` `z` again restores the layout.
4. `Ctrl-a` `q`, then `y` → dozer quits.

## 3. Choosing the layout and what runs

### From the command line

| Command | Result |
|---|---|
| `dozer` | Default `2,1` layout, a shell in each pane |
| `dozer -l 2,2,1` | Rows of 2, 2 and 1 panes |
| `dozer -l 3` / `-l 1,3` / `-l 3,3` | Any row pattern, up to 9 panes |
| `dozer -l 2,1 --heights 70,30` | Top row 70% of the height |
| `dozer -l 2,1 --widths 1:30,70` | Row 1 split 30% / 70% |
| `dozer -p top -p 'git log'` | **Run** commands in panes 1, 2, … (you get a prompt when each finishes) |
| `dozer -x 'tail -f log'` | **Exec**: the pane *is* the command; when it ends, the pane is dead (§4) |
| `dozer -p a -p b -p c -p d` | No `-l`: the layout is fitted to the commands (4 → `2,2`) |
| `printf 'top\ngit status\n' \| dozer` | Piped: one pane command per line |
| `dozer --prefix C-b` | Use `Ctrl-b` as the prefix instead |
| `dozer -l 2,1 -p htop --save my.yaml` | Don't launch; write what you'd get as a config file (`--save -` prints it) |

Sizes accept `60` or `60%` (percent), `12c` (fixed rows or columns) and `2fr` (a share of what's left). Every flag is in [`CLI.md`](CLI.md).

### From a config file (YAML)

The folder `examples/` has ready-made configs, and each one explains itself at the top:

| File | Shows |
|---|---|
| `reference.yaml` | Every key, with defaults (same as a bare `dozer`) |
| `dev.yaml` | A coding workspace: shell, git log, an auto-restarting clock |
| `monitor.yaml` | The `2,2,1` layout full of live output |
| `tree.yaml` | The tree form: a tall left pane with two stacked on the right |
| `lifecycle.yaml` | Panes that die in different ways, to see dead-pane handling |
| `ssh.yaml` | A template for watching servers (edit the hosts first) |

```sh
dozer -c examples/dev.yaml
dozer --check examples/tree.yaml    # validate + show what would launch, without launching
```

To write your own, start from one of these and read the [configuration guide](CONFIG.md): it walks through a first config step by step, then covers every layout option and key.

### Saving and sharing a setup

```sh
dozer -l 2,1 -p htop --save my.yaml          # a command line → an editable config file
cp my.yaml ~/.config/dozer/my.yaml; dozer my # … used as a profile
dozer -c examples/dev.yaml --package dev-ws  # a config → ./dev-ws.sh, one runnable file
./dev-ws.sh                                  # launch it (extra flags pass through)
./dev-ws.sh --show-config                    # see the config inside
```

A `.dozer.yaml` in the current folder, or `~/.config/dozer/config.yaml`, is loaded automatically by a bare `dozer`. [`PACKAGING.md`](PACKAGING.md) covers saving, packages and sharing in full.

## 4. Scrollback, copying and the mouse

Every pane keeps its history (10,000 lines by default; `scrollback:` in a config changes it).

**With the mouse** (on by default):

- **Click** a pane to focus it. **Drag** a divider or a title line to resize.
- **Wheel** over a shell to scroll back through its history; wheel back down to the bottom to return. Over `less` or `man` the wheel scrolls the page; programs that use the mouse themselves (vim with `mouse=a`, htop) get it directly.
- **Drag** inside a pane to select. The selection stays inside that pane, even if you drag across its border, and it's copied to your clipboard when you let go. Paste as usual (Cmd-V).
- Want your terminal's own selection back for a moment? iTerm2: hold Option. Terminal.app: View → Allow Mouse Reporting (⌘R) toggles it. Or start dozer with `--no-mouse` (or `mouse: false` in a config).

**With the keyboard**, `Ctrl-a [` opens **copy mode** on the focused pane. The status bar says `COPY`, and a tag like `[120/3000]` shows how far back you are. The program keeps running; the view just holds still.

| Keys | |
|---|---|
| `↑↓←→` / `h j k l` | move |
| `Ctrl-u` `Ctrl-d` · `PgUp` `PgDn` | half page / page |
| `g` / `G` | oldest line / back to the live screen |
| `w` `b` `e` · `0` `$` | words · line start and end |
| `v` / `V` | select characters / whole lines |
| `y` or `Enter` | copy to the clipboard and leave |
| `/` `?` + text, `n` `N` | search down / up, next / previous (lower case ignores case) |
| `q` or `Esc` | leave |

A line that wrapped on screen copies as one line. If you make the window narrower and then wider again, text rewraps and comes back intact.

## 5. When a pane's process ends

A pane is a **permanent slot**. When its process exits or crashes, the layout does *not* change. Instead the pane is flagged:

- Its title shows the state: `exited` (dimmed, status 0), `✖ exit 3` or `✖ SIGKILL` (red), or `✖ connection lost` (an `-x`/`exec:` ssh pane that exited 255).
- Its last output stays visible but dimmed, with a banner along its bottom edge: `✖ exit 3 at 14:02:31 · C-a r restart`.
- The status bar shows a red `✖ N dead · C-a R`, even when that pane is zoomed out of view.

**Bringing it back is your call:** `Ctrl-a r` (focused pane) or `Ctrl-a R` (all dead panes) re-runs the pane's original command in the same spot. A config can opt a pane into automatic restarts with `restart: on-failure` or `restart: always`. Restarts back off (1 s, 2 s, 4 s … up to 30 s) if the command keeps failing.

dozer itself only quits when you say so (`Ctrl-a q`), even if every pane has died. (`--quit-when-all-exited` changes that.)

## 6. Expected for now (not bugs)

- **Copying is tmux-style** (copy mode, or a mouse drag), not the OS's own highlight-and-Cmd-C, which can't know about panes. It works; making it feel natural is parked for now (SPEC DP-9).
- **Copy mode is per pane.** Switching focus leaves a pane in copy mode where it was; `q` there brings it back to live. Restarting a pane clears its history.
- **Full-screen programs have no history** in dozer (they draw on the alternate screen, like in any terminal). Copy mode then covers what's on screen.
- **A window that's too small** keeps panes at a minimum size and scrolls the view to the focused pane. The status bar shows `more ◀ ▶` when part of the layout is off-screen.
- **Option key as Meta** depends on your terminal setting, just as without dozer:
  - Terminal.app: Settings → Profiles → Keyboard → "Use Option as Meta key"
  - iTerm2: Profiles → Keys → Left Option key: Esc+

## 7. If something goes wrong

| Symptom | Try |
|---|---|
| The display looks garbled | `Ctrl-a` `Ctrl-l` |
| Keys seem dead | Look at the status bar. **PREFIX** means dozer is waiting (press `Esc`); "Quit dozer?" is waiting for `y` or another key. Or the focused pane is dead: check its title. |
| dozer is stuck | From another terminal: `pkill dozer` |
| Your terminal is weird after dozer exits | Type `reset` + Enter, and please report it: that's a bug |

## 8. Now test it

Every test is also a ready-to-run script in [`tests/cases/`](../tests/cases/README.md). Each one sets up exactly its scenario and shows its own steps and expected result in pane 1:

```sh
tests/cases/K03-resize.sh
```

How to run, record and write cases is in [`TESTING.md`](TESTING.md). For anything off, a screenshot plus the terminal app name is the most useful report.
