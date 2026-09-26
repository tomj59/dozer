# dozer

A TUI multi-shell for Linux and macOS: one terminal window holding several live shells in a row/column layout, each set up at launch by flags, piped commands or a YAML file.

> **Status: M2.** Multiple panes in rows/columns, focus, zoom, runtime resize, dead-pane handling, YAML configs, `--save` and `--package`. See [`docs/SPEC.md`](docs/SPEC.md) for the design and roadmap.

**New here? Start with [`docs/QUICKSTART.md`](docs/QUICKSTART.md).**

## Build

Requires Go 1.24+.

```sh
make            # builds bin/dozer for this machine
make test       # vet + tests
make dist       # cross-compiles dist/dozer-{darwin,linux}-{arm64,amd64}
```

## Try it

```sh
bin/dozer                          # default 2,1 layout: two panes on top, one below
bin/dozer -l 2,2,1                 # five panes
bin/dozer -p htop -p 'git log'     # run commands in panes 1, 2 … then leave a prompt
bin/dozer -x 'tail -f log'         # the pane IS the command
bin/dozer -c examples/dev.yaml     # a YAML config; see examples/
bin/dozer --check examples/tree.yaml
bin/dozer -c examples/dev.yaml --package dev-ws   # → dev-ws.sh, a shareable runnable script
```

Prefix key: **Ctrl-a**, then: arrows/hjkl/1-9/o to focus, `z` zoom, `H J K L` resize, `=` reset sizes, `t` rename, `s` status bar, `x` kill, `r`/`R` restart, `Ctrl-l` redraw, `q` quit, `Esc` cancel.

## Layout of the code

| Path | What |
|---|---|
| `cmd/dozer` | CLI entry point |
| `internal/app` | Event loop, focus/zoom, compositor, chrome |
| `internal/layout` | Layout shorthand, tree, and solver |
| `internal/config` | YAML config, profiles, `--check` |
| `internal/host` | Raw mode, alt screen, size, mode mirroring |
| `internal/input` | Prefix-key router over raw bytes (bracketed-paste aware) |
| `internal/pane` | PTY + child process + emulator |
| `internal/emu` | Emulator adapter (charm vt, vt10x) and benchmarks |
| `third_party/vt` | Patched fork of `charmbracelet/x/vt`; see `DOZER_PATCHES.md` |
| `examples/` | Example configs (all loaded by the tests) |
| `scripts/smoke.sh` | Headless end-to-end checks inside tmux |

## License

GPL-3.0. See [LICENSE](LICENSE). `third_party/vt` is MIT (see its LICENSE).
