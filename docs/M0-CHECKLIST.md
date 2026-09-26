# M0 checklist: Mac verification

New to dozer? Read [QUICKSTART.md](QUICKSTART.md) first; it teaches everything this checklist uses.

> **Since M1:** a bare `bin/dozer` opens three panes. For this checklist, use **`bin/dozer -l 1`** (one pane) wherever it says `bin/dozer`. Redraw moved from `Ctrl-a l` to **`Ctrl-a Ctrl-l`**.

Goal: confirm DP-2 (emulator = charm vt, raw-key passthrough) on real macOS terminals before M1 builds layouts on top of it. About 15 minutes. Note the terminal app and version for each run.

## 0. Build

```sh
cd <your dozer clone>
make            # or: go build -o bin/dozer ./cmd/dozer
make test
bin/dozer -version
```

Prebuilt fallback: `dist/dozer-darwin-arm64`. If macOS blocks it, run `xattr -d com.apple.quarantine dist/dozer-darwin-arm64`.

Report: did `make` and `make test` pass on Go 1.27.1? (Y/N plus any errors)

## 1. Basics (`bin/dozer`)

| # | Do | Expect | Result |
|---|---|---|---|
| 1.1 | Launch | Your normal prompt, plus a reverse-video status bar on the last row | |
| 1.2 | `ls -G`, `git log --oneline --graph --color \| head` | Colors look right | |
| 1.3 | `printf '\e[38;2;255;128;0morange\e[0m 日本語 👍 e\xcc\x81\n'` | Orange text; the CJK, emoji and accented e are aligned, and the prompt isn't shifted afterwards | |
| 1.4 | `Ctrl-a` | Status bar shows **PREFIX** | |
| 1.5 | `Ctrl-a Ctrl-a` while typing a command | The cursor jumps to the start of the line (literal ^A reached the shell) | |
| 1.6 | Resize the window by dragging | Prompt redraws; `stty size` matches (one row less than the window) | |
| 1.6b | `Ctrl-a` then `←` | PREFIX clears; nothing like `[D` gets typed | |
| 1.6c | `Cmd-K`, then `Ctrl-a Ctrl-l` | The screen blanks, then repaints fully | |
| 1.7 | `Ctrl-a q` | dozer exits and your terminal is exactly as before (no stray colors, cursor visible, typing echoes) | |

## 2. Full-screen apps

| # | Do | Expect | Result |
|---|---|---|---|
| 2.1 | `vim` on any file: arrows, `:set nu`, visual mode, `:q` | Arrows move the cursor (they don't insert A/B/C/D); the screen restores on quit | |
| 2.2 | In vim, resize the window | vim redraws to the new size | |
| 2.3 | `htop` (or `top`), then resize, then quit | Bars and colors are fine; it follows the resize | |
| 2.4 | `less` on a long file: `space`, `G`, `/search`, `q` | Works and restores | |
| 2.5 | `vim`, then paste multi-line text with Cmd-V in insert mode | Pasted as-is; no staircase auto-indent (bracketed paste works) | |
| 2.6 | Option-key combos in the shell (`Option-b`, `Option-f`, with "Use Option as Meta" on) | Moves by word, same as outside dozer | |

## 3. Throughput

| # | Do | Expect | Result |
|---|---|---|---|
| 3.1 | `time seq 1 1000000` in dozer | Finishes in a few seconds; `Ctrl-a q` still responds during the flood | seconds: |
| 3.2 | Same command outside dozer | For comparison | seconds: |
| 3.3 | Same inside `bin/dozer -emu vt10x` | For comparison | seconds: |

## 4. A/B: `bin/dozer -emu vt10x`

Repeat 1.3 and 2.1 to 2.5. Anything that works better than in the default (charm) back end? Linux results predict charm wins on true color, wide characters and underline styles.

## 5. Mouse (bonus, only for apps that ask for it)

| # | Do | Expect | Result |
|---|---|---|---|
| 5.1 | `vim`, `:set mouse=a`, click somewhere | The cursor moves there | |
| 5.2 | `htop`, click a column header | Sorts by that column | |

## Report back

Paste the tables with results, plus screenshots of anything visually off, and which terminal app(s) you tested.
