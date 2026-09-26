# dozer

A TUI multi-shell for Linux and macOS: one terminal window holding several live shells in a row/column layout, each set up at launch by flags, piped commands or a YAML file.

> **Status: M0 spike.** One full-screen pane is used to validate terminal emulation, input passthrough and rendering. Layouts arrive in M1. See [`docs/SPEC.md`](docs/SPEC.md) for the full design and roadmap.

## Build

Requires Go 1.24+.

```sh
make            # builds bin/dozer for this machine
make test       # vet + tests
make dist       # cross-compiles dist/dozer-{darwin,linux}-{arm64,amd64}
```

## Try the M0 spike

```sh
bin/dozer                    # your $SHELL in one pane
bin/dozer -p htop            # run htop, drop to a prompt when it exits
bin/dozer -x 'tail -f log'   # the pane IS the command
bin/dozer -emu vt10x         # A/B: the alternative emulator back end
```

Prefix key: **Ctrl-a**. `Ctrl-a q` quits, and `Ctrl-a Ctrl-a` sends a literal Ctrl-a to the shell.

## Layout of the code

| Path | What |
|---|---|
| `cmd/dozer` | CLI entry point |
| `internal/app` | Wires host terminal, input and panes; render loop |
| `internal/host` | Raw mode, alt screen, size, mode mirroring |
| `internal/input` | Prefix-key router over raw bytes (bracketed-paste aware) |
| `internal/pane` | PTY + child process + emulator |
| `internal/emu` | Emulator adapter (charm vt, vt10x) and benchmarks |
| `third_party/vt` | Patched fork of `charmbracelet/x/vt`; see `DOZER_PATCHES.md` |
| `scripts/smoke.sh` | Headless end-to-end checks inside tmux |

## License

GPL-3.0. See [LICENSE](LICENSE). `third_party/vt` is MIT (see its LICENSE).
