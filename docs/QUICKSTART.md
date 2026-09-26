# dozer quickstart

This page teaches you to use dozer as it exists today (M2: multiple panes with runtime controls). When you've finished it, [`M2-CHECKLIST.md`](M2-CHECKLIST.md) is where you record whether it works. ([`M1-CHECKLIST.md`](M1-CHECKLIST.md) covers the basics, if you haven't done it.)

## What you're looking at

dozer fills your terminal window with several **panes**. Each pane is a real, independent shell (or a command you chose), arranged in rows and columns and separated by thin lines:

```
─ [1] zsh ─────────────────────────┬─ [2] git log ────────────────────────   ← title line of each pane
tomj@mac ~/dozer %                 │* db5558e Add QUICKSTART …
                                   │* f7f9ee3 M0 spike …
                                   │tomj@mac ~/dozer %
─ [3] zsh ─────────────────────────────────────────────────────────────────
tomj@mac ~/dozer %

 dozer │ [1] zsh                                       C-a q quit │ 6152094   ← status bar
```

- **Title line:** the pane number, its label, and its state when the process has ended. The pane you're typing into (the **focused** pane) has a bright cyan title.
- **Status bar:** which pane is focused, a red alert when a pane has died, and hints.

The default layout is `2,1`: two panes on top, one wide pane below. Every pane starts your shell in the folder you launched from.

## 1. Build it

```sh
cd ~/claude/Projects/tui-multi-shell/dozer
make              # builds bin/dozer
```

## 2. Start it and move around

```sh
bin/dozer
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
| `bin/dozer` | Default `2,1` layout, a shell in each pane |
| `bin/dozer -l 2,2,1` | Rows of 2, 2 and 1 panes |
| `bin/dozer -l 3` / `-l 1,3` / `-l 3,3` | Any row pattern, up to 9 panes |
| `bin/dozer -l 2,1 --heights 70,30` | Top row 70% of the height |
| `bin/dozer -l 2,1 --widths 1:30,70` | Row 1 split 30% / 70% |
| `bin/dozer -p top -p 'git log'` | **Run** commands in panes 1, 2, … (you get a prompt when each finishes) |
| `bin/dozer -x 'tail -f log'` | **Exec**: the pane *is* the command; when it ends, the pane is dead (§4) |
| `bin/dozer -p a -p b -p c -p d` | No `-l`: the layout is fitted to the commands (4 → `2,2`) |
| `printf 'top\ngit status\n' \| bin/dozer` | Piped: one pane command per line |
| `bin/dozer --prefix C-b` | Use `Ctrl-b` as the prefix instead |
| `bin/dozer -l 2,1 -p htop --save my.yaml` | Don't launch; write what you'd get as a config file (`--save -` prints it) |

Sizes accept `60` or `60%` (percent), `12c` (fixed rows or columns) and `2fr` (a share of what's left).

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
bin/dozer -c examples/dev.yaml
bin/dozer --check examples/tree.yaml    # validate + show what would launch, without launching
```

To reuse a config as a **profile**, copy it to `~/.config/dozer/NAME.yaml` and run `dozer NAME`. A `.dozer.yaml` in the current folder, or `~/.config/dozer/config.yaml`, is loaded automatically by a bare `dozer`.

## 4. When a pane's process ends

A pane is a **permanent slot**. When its process exits or crashes, the layout does *not* change. Instead the pane is flagged:

- Its title shows the state: `exited` (dimmed, status 0), `✖ exit 3` or `✖ SIGKILL` (red), or `✖ connection lost` (an `-x`/`exec:` ssh pane that exited 255).
- Its last output stays visible but dimmed, with a banner along its bottom edge: `✖ exit 3 at 14:02:31 · C-a r restart`.
- The status bar shows a red `✖ N dead · C-a R`, even when that pane is zoomed out of view.

**Bringing it back is your call:** `Ctrl-a r` (focused pane) or `Ctrl-a R` (all dead panes) re-runs the pane's original command in the same spot. A config can opt a pane into automatic restarts with `restart: on-failure` or `restart: always`. Restarts back off (1 s, 2 s, 4 s … up to 30 s) if the command keeps failing.

dozer itself only quits when you say so (`Ctrl-a q`), even if every pane has died. (`--quit-when-all-exited` changes that.)

## 5. Expected for now (not bugs)

- **No mouse yet.** Clicking doesn't focus panes and apps like htop don't receive clicks until M4. Your terminal's own text selection still works.
- **No scrollback.** Output that scrolls off the top of a pane is gone for now (M4). The scroll wheel may cycle shell history instead.
- **A window that's too small** keeps panes at a minimum size and scrolls the view to the focused pane. The status bar shows `more ◀ ▶` when part of the layout is off-screen.
- **Option key as Meta** depends on your terminal setting, just as without dozer:
  - Terminal.app: Settings → Profiles → Keyboard → "Use Option as Meta key"
  - iTerm2: Profiles → Keys → Left Option key: Esc+

## 6. If something goes wrong

| Symptom | Try |
|---|---|
| The display looks garbled | `Ctrl-a` `Ctrl-l` |
| Keys seem dead | Look at the status bar. **PREFIX** means dozer is waiting (press `Esc`); "Quit dozer?" is waiting for `y` or another key. Or the focused pane is dead: check its title. |
| dozer is stuck | From another terminal: `pkill dozer` |
| Your terminal is weird after dozer exits | Type `reset` + Enter, and please report it: that's a bug |

## 7. Now test it

Work through [`M1-CHECKLIST.md`](M1-CHECKLIST.md). For anything off, a screenshot plus the terminal app name is the most useful report.
