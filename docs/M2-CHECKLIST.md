# M2 checklist: runtime controls

New keys since M1 (see [QUICKSTART.md](QUICKSTART.md) §2). About 10 minutes.

```sh
cd ~/claude/Projects/tui-multi-shell/dozer && make && make test && bin/dozer
```

| # | Do | Expect | Result |
|---|---|---|---|
| R1 | `Ctrl-a L`, then `L L L` (no prefix) | Pane 1's right border moves right each press; the status bar says **RESIZE** | |
| R2 | Wait a second, type `L` | A plain `L` is typed into the shell (the repeat window ended) | |
| R3 | `Ctrl-a 3`, `Ctrl-a K K K` | The bottom pane grows upward | |
| R4 | `Ctrl-a 2`, `Ctrl-a L` | Pane 2 is on the right edge, so its left border moves right (pane 2 shrinks), like tmux | |
| R5 | Resize the window after R1–R4 | Your adjusted proportions are kept | |
| R6 | `Ctrl-a =` | Back to the launch sizes | |
| R7 | Push a border as far as it goes | It stops at the minimum pane size | |
| T1 | `Ctrl-a t`, type a name, Enter | The title changes; the cursor sits in the status-bar prompt while typing | |
| T2 | `Ctrl-a t`, clear it (`Ctrl-u`), Enter | The default label comes back | |
| S1 | `Ctrl-a s` twice | The status bar hides (panes gain a row), then returns | |
| S2 | With the bar hidden, press `Ctrl-a` | The bar reappears while PREFIX is pending | |
| K1 | `Ctrl-a x`, then `n` | Nothing happens | |
| K2 | `Ctrl-a x`, then `y` | The pane dies (`✖ SIGHUP`), the layout is unchanged; `Ctrl-a r` brings it back | |
| V1 | `bin/dozer -l 2,1 --heights 70,30 -p top --save -` | Prints a YAML config; `--save my.yaml` writes one, and `bin/dozer -c my.yaml` launches the same thing | |
| V2 | `bin/dozer --save - -c examples/tree.yaml` | Tree form, with the original pane names | |
| P1 | `bin/dozer -c examples/dev.yaml --package /tmp/dev-ws` | Writes `/tmp/dev-ws.sh`, which is executable | |
| P2 | `cd /tmp && DOZER_BIN=~/claude/Projects/tui-multi-shell/dozer/bin/dozer ./dev-ws.sh` | The dev workspace launches; the status bar says `dev-ws` | |
| P3 | `./dev-ws.sh --show-config` | Prints the YAML | |
| P4 | `./dev-ws.sh` without dozer on PATH (no DOZER_BIN) | A clear "needs dozer" message, exit 127 | |
| P5 | A config with `cwd: sub` packaged into a folder, then the script copied elsewhere | The pane starts in `sub/` next to the *copied* script | |
| C1 | `status_bar: top` in a config | The bar at the top; panes below it | |

Report: results, plus your feel for the resize step sizes (2 columns / 1 row per press) and the repeat window (0.7 s).
