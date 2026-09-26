# M1 checklist: multi-pane on the Mac

Read [QUICKSTART.md](QUICKSTART.md) first. It covers every key and flag used here. About 20–30 minutes. Note your terminal app (Terminal, iTerm2, …).

```sh
cd ~/claude/Projects/tui-multi-shell/dozer && make && make test
```

## A. Layout and chrome (`bin/dozer`)

| # | Do | Expect | Result |
|---|---|---|---|
| A1 | Launch | Three panes (two on top, one wide below); thin lines between them; `[1]` title is cyan; status bar at the bottom | |
| A2 | Type `ls -G` in pane 1 | Output stays inside pane 1; colors right | |
| A3 | Resize the window by dragging | Panes re-proportion, and prompts redraw in every pane | |
| A4 | `bin/dozer -l 2,2,1`, then `-l 3`, then `-l 1,3` | The layouts match the shorthand | |
| A5 | `bin/dozer -l 2,1 --heights 70,30 --widths 1:30,70` | A big top row; its left pane is narrow | |

## B. Focus and keys

| # | Do | Expect | Result |
|---|---|---|---|
| B1 | `Ctrl-a →`, type `echo two` + Enter | It lands in pane 2; pane 2's title is cyan | |
| B2 | `Ctrl-a ↓`, `Ctrl-a ↑`, `Ctrl-a 3`, `Ctrl-a o` | Focus moves as named | |
| B3 | `Ctrl-a h/j/k/l` | Same as the arrows | |
| B4 | Recall history (↑) and edit mid-line in panes 2 and 3 | The cursor stays exactly where edits land (the M0 fix holds in every pane) | |
| B5 | `Ctrl-a z`, then `Ctrl-a z` | Focused pane fills the window, then the layout returns | |
| B6 | Zoom, then `Ctrl-a →` | Leaves zoom and moves focus | |
| B7 | `Ctrl-a Ctrl-l` | Repaint; nothing changes visibly | |
| B8 | `Ctrl-a q`, press `n`; then `Ctrl-a q`, press `y` | The first stays in dozer; the second quits; your terminal is exactly as before | |

## C. Commands at launch

| # | Do | Expect | Result |
|---|---|---|---|
| C1 | `bin/dozer -p top -p 'git log --oneline -5'` | top in pane 1; git log in pane 2, then a prompt; a shell in pane 3 | |
| C2 | Quit `top` with `q` in pane 1 | You're at a prompt in pane 1 (that's `-p`) | |
| C3 | `bin/dozer -x top` then quit top | Pane 1 turns dead (dim `exited`); the layout doesn't change | |
| C4 | `printf 'echo a\necho b\necho c\necho d\n' \| bin/dozer` | Four panes (2,2) showing a/b/c/d; the keyboard still works | |

## D. Configs (`examples/`)

| # | Do | Expect | Result |
|---|---|---|---|
| D1 | `bin/dozer --check examples/tree.yaml` | Prints the layout and three panes; no launch | |
| D2 | `bin/dozer -c examples/dev.yaml` | Shell / git / a ticking clock; the top row is taller | |
| D3 | In dev.yaml's clock pane: `Ctrl-a 3`, `Ctrl-a r` | The clock restarts | |
| D4 | `bin/dozer -c examples/tree.yaml` | A tall left pane, two stacked on the right; the divider joins the title line (`├`) | |
| D5 | `bin/dozer -c examples/monitor.yaml` | Five busy panes updating together; typing in pane 5 stays responsive | |
| D6 | Copy `examples/dev.yaml` to `~/.config/dozer/dev.yaml`, run `bin/dozer dev` | The profile loads by name | |
| D7 | Break a config (a typo'd key, or `layout: "2,x"`) and `--check` it | A clear error with a line number | |

## E. Dead panes (`bin/dozer -c examples/lifecycle.yaml`)

| # | Do | Expect | Result |
|---|---|---|---|
| E1 | Wait ~6 s | [1] dim `exited`; [2] red `✖ exit 3`; [3] red `✖ SIGKILL`; banners at the bottom of each; the status bar says `✖ 3 dead` or `✖ 4 dead` | |
| E2 | Watch [4] | `↻ restarting automatically…`, then it runs again (the gap grows each time) | |
| E3 | Is anything easy to overlook? | Your judgment: is dead-pane chrome loud enough? | |
| E4 | `Ctrl-a 2`, `Ctrl-a r` | Only pane 2 restarts (flash: `restarted [2]`) | |
| E5 | `Ctrl-a R` | All dead panes restart | |
| E6 | In pane 5 type `exit` | Pane 5 is dead; dozer stays open | |
| E7 | `Ctrl-a q` | Asks only if panes are running; `y` quits | |

## F. Small window

| # | Do | Expect | Result |
|---|---|---|---|
| F1 | `bin/dozer -l 3`, then shrink the window narrow | Panes stop shrinking at a minimum; status shows `more ▶` | |
| F2 | `Ctrl-a 3` | The view scrolls to show pane 3; the status shows `more ◀` | |
| F3 | Your judgment (DP-1) | Is scrolling the right behavior, or would you rather auto-zoom? | |

## G. Still open from M0 (do these in any pane)

M0-CHECKLIST sections 2 (vim / top / less / paste / Option key) and 3 (`time seq 1 1000000`), if not done yet.

## Report back

Results for each table, screenshots of anything visually off, and your judgment on E3 and F3.
