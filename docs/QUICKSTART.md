# dozer quickstart (M0)

This page teaches you to use dozer as it exists today. When you've finished it, [`M0-CHECKLIST.md`](M0-CHECKLIST.md) is where you record whether it works.

## What you're looking at

M0 is the engine test, not the finished tool. Right now dozer shows **one** shell filling your terminal window, with a one-line **status bar** at the bottom. Multiple panes and layouts come in M1.

That single pane is the hard part. dozer runs a real shell on its own pseudo-terminal, emulates that terminal itself, and redraws the result inside your real terminal. If this layer is solid, everything later sits on it. So the job right now is: **does a shell inside dozer feel exactly like a shell outside it?**

```
┌──────────────────────────────────────────────────────────────┐
│ tomj@mac ~/claude/Projects/tui-multi-shell/dozer %           │ ← your normal shell, running
│                                                              │   inside dozer
│                                                              │
│                                                              │
│ dozer f7f9ee3 │ emu: charm │ C-a q: quit │ tomj@mac: ~/…     │ ← dozer's status bar
└──────────────────────────────────────────────────────────────┘
```

The status bar shows:

- the dozer version
- the emulator back end in use (`charm` by default)
- a reminder of how to quit
- the window title your shell sets, if it sets one

## 1. Build it (one time)

In a terminal on your Mac:

```sh
cd ~/claude/Projects/tui-multi-shell/dozer
make              # builds bin/dozer
```

`make` uses your Go install. If `make` isn't available, `go build -o bin/dozer ./cmd/dozer` does the same thing.

## 2. Start it and look around

```sh
bin/dozer
```

The screen clears and you get your normal prompt, with the status bar at the bottom. Now use it like any terminal:

```sh
ls -G
echo hello
cd ..
```

Nothing about typing, history (↑/↓), tab completion or `Ctrl-c` should feel different.

## 3. The one new idea: the prefix key

A shell inside dozer needs every normal keystroke. So dozer reserves exactly one key, the **prefix**, `Ctrl-a`, and listens for its commands only right after that key.

1. Press and release **`Ctrl-a`**. The status bar changes to **PREFIX** and the cursor hides. dozer is now waiting for one command key.
2. Press a **command key**:

| After `Ctrl-a`, press | What happens |
|---|---|
| `q` | **Quit dozer.** The shell inside is closed, and your terminal is back exactly as it was. |
| `l` (lowercase L) | Redraw the whole screen. Use it if the display ever looks wrong. |
| `Ctrl-a` again | Send one real `Ctrl-a` to the shell. In zsh or bash that jumps to the start of the line. |
| `Esc`, an arrow key, or anything else | Cancels; you're back to normal typing. |

Try it now:

- `Ctrl-a` then `l` → nothing visible should change. It's a repaint.
- Type `echo abc`, then `Ctrl-a Ctrl-a` → the cursor jumps to the start of the line.
- `Ctrl-a` then `q` → dozer exits.

That's the entire command set in M0. Everything else goes straight to your shell.

## 4. Ways to leave

| Action | Result |
|---|---|
| `Ctrl-a` `q` | Quits dozer. |
| Type `exit` in the shell | The shell ends, so dozer ends too. (In M0 the pane *is* the session.) |
| Close the terminal window | Also fine. dozer and its shell are hung up cleanly. |

If a command exits with an error code, dozer prints `dozer: pane exited with status N` after it closes. That's information, not a crash.

## 5. Starting dozer with a command

These are the M0 flags:

| Command | What it does |
|---|---|
| `bin/dozer` | Your login shell (`$SHELL`) in the pane |
| `bin/dozer -p top` | **Run** `top` in your shell; when you quit `top` (`q`), you're left at a shell prompt inside dozer |
| `bin/dozer -x top` | **Exec** `top`: `top` *is* the pane, so quitting `top` also exits dozer |
| `bin/dozer -emu vt10x` | Same as `bin/dozer`, using the alternative emulator (for comparison only) |
| `bin/dozer -version` | Print the version and exit |
| `bin/dozer -h` | List the flags |

Quote commands that contain spaces: `bin/dozer -p 'tail -f /var/log/system.log'`.

The difference between `-p` and `-x` is the same one you'll later write in YAML as `run:` and `exec:`. `-p` is for "start this, and leave me a shell afterwards". `-x` is for "this pane only exists to show this command".

## 6. Things that are *expected* in M0 (not bugs)

- **One pane only.** Layouts, pane switching and multi-input come in M1 and later.
- **No scrollback.** Output that scrolls off the top is gone for now. The scroll wheel may even cycle through shell history, because in full-screen mode many terminals turn the wheel into ↑/↓ key presses. Scrollback arrives in M4.
- **`Cmd-K` (clear) in Terminal/iTerm2.** It wipes the host screen behind dozer's back, so parts may stay blank until they change. Press `Ctrl-a` `l` to repaint, or use `clear` instead.
- **Option key as Meta.** Word-jumping with Option-b/f works only if your terminal sends Option as Meta, exactly as without dozer:
  - Terminal.app: Settings → Profiles → Keyboard → "Use Option as Meta key"
  - iTerm2: Profiles → Keys → Left Option key: Esc+
- **The status bar costs one row.** `stty size` reports one row less than your window.

## 7. If something goes wrong

| Symptom | Try |
|---|---|
| The display looks garbled | `Ctrl-a` `l` |
| Keys seem dead | Look at the status bar. If it says **PREFIX**, dozer is waiting for a command key; press `Esc` to cancel. |
| dozer is stuck and won't quit | From another terminal: `pkill dozer` |
| Your terminal is weird after dozer exits (no echo, odd colors) | Type `reset` and press Enter, even if you can't see it. **Also report it: that's a bug.** |

## 8. Now test it

Open [`M0-CHECKLIST.md`](M0-CHECKLIST.md) and work through it with the steps above. For anything that seems off, a screenshot plus the terminal app name (Terminal, iTerm2, Ghostty, …) is the most useful report.
