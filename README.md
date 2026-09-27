# dozer

Several live shells in one terminal window, arranged in rows and columns and set up at launch by flags, piped commands or a small YAML file. For Linux and macOS.

```
─ [1] zsh ─────────────────────────┬─ [2] git log ────────────────────────
tomj@mac ~/shop %                  │* db5558e Add checkout flow
                                   │* f7f9ee3 Initial commit
                                   │tomj@mac ~/shop %
─ [3] server ─────────────────────────────────────────────────────────────
listening on :8080
GET /cart 200 3ms

 shop │ [1] zsh                                      C-a q quit │ v0.4.0
```

Each pane is a real terminal, so vim, htop, ssh and less work inside it. A pane whose program ends stays put, flagged, until you restart it. dozer is a standalone program: it doesn't need tmux or screen.

> **Version 0.4.0** (milestone M4 of the [roadmap](docs/SPEC.md#9-roadmap)): layouts, runtime controls, YAML configs, shareable packages, scrollback with copy mode and search, text that rewraps on resize, and the mouse. See the [changelog](CHANGELOG.md).

## Install

Needs Go 1.24 or later.

```sh
git clone https://github.com/tomj59/dozer && cd dozer
make install                  # builds and copies dozer to /usr/local/bin
make install PREFIX=~/.local  # or into ~/.local/bin
```

`make` alone builds `bin/dozer` without installing it.

## Use it

```sh
dozer                                   # three shells: two on top, one below
dozer -l 2,2,1                          # five shells in rows of 2, 2 and 1
dozer -l 2,1 -p htop -p 'git log'       # run commands in panes 1, 2 …, then leave a prompt
dozer -l 1 -x 'tail -F app.log'         # one pane that IS the command
dozer -c examples/dev.yaml              # a workspace from a config file
dozer work                              # the profile ~/.config/dozer/work.yaml
```

Inside, **Ctrl-a** is the command key: then `←↑↓→` (or `1`–`9`) to move between panes, `z` to zoom one, `[` to scroll back through its history, and `q` to quit. The mouse works too: click to focus, drag a border to resize, wheel to scroll back.

A workspace config looks like this:

```yaml
name: shop
layout: "2,1"
heights: [65, 35]
panes:
  - { title: editor, run: vim . }
  - { title: git,    run: git log --oneline -15 }
  - { title: server, cwd: backend, exec: make run, restart: on-failure }
```

And any workspace can become one file you can share:

```sh
dozer -c shop.yaml --package shop       # writes shop.sh
./shop.sh                               # anyone with dozer can run it
```

## Documentation

| | |
|---|---|
| [Quickstart](docs/QUICKSTART.md) | Learn dozer in ten minutes |
| [Configuration guide](docs/CONFIG.md) | Build your own workspaces: layouts, sizes, panes, every key |
| [Saving and sharing](docs/PACKAGING.md) | `--save`, `--package`, configs without files |
| [Command line](docs/CLI.md) | Every flag, the environment, exit codes |
| [Testing](docs/TESTING.md) | Automatic tests, the manual test cases, writing a new case |
| [Specification](docs/SPEC.md) | Design decisions, architecture, roadmap |
| [Examples](examples/) | Ready-to-run configs |

## Develop

```sh
make            # build bin/dozer
make test       # vet and unit tests (including the emulator fork)
go test -race ./...
make cases      # validate the manual test cases
scripts/smoke.sh  # headless end-to-end checks (needs tmux, vim and python3)
make dist       # cross-compile dist/dozer-{darwin,linux}-{arm64,amd64}
```

| Path | What |
|---|---|
| `cmd/dozer` | Command-line entry point |
| `internal/app` | Event loop, focus and zoom, compositor, chrome, copy mode and mouse handling |
| `internal/layout` | Layout shorthand, tree, and solver |
| `internal/config` | YAML configs, profiles, `--check`, `--save`, `--package` |
| `internal/pane` | Pseudo-terminal, child process, emulator, lifecycle, copy-mode model |
| `internal/emu` | Terminal emulator adapter and benchmarks |
| `internal/input` | Prefix-key router over raw bytes; key and mouse decoding |
| `internal/host` | The outer terminal: raw mode, drawing, mode mirroring, clipboard escape |
| `internal/clip` | Local clipboard commands |
| `third_party/vt` | Patched fork of `charmbracelet/x/vt` ([what changed](third_party/vt/DOZER_PATCHES.md)) |
| `examples/` | Example configs (all loaded by the tests) |
| `tests/cases/` | Manual test cases, one runnable script each |
| `scripts/smoke.sh` | Headless end-to-end checks in tmux |

## License

GPL-3.0; see [LICENSE](LICENSE). `third_party/vt` is MIT (see its LICENSE).
