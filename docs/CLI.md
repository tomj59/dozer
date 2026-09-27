# Command-line reference

```
dozer [flags] [profile | config-file]
```

Every flag can be written with one dash or two (`-l` = `--l`, `--prefix` = `-prefix`). Flags go **before** a profile or file name: `dozer --prefix C-b work`, not `dozer work --prefix C-b`. `dozer -h` prints a short version of this page.

## Arguments

| Argument | Loads |
|---|---|
| (none) | `./.dozer.yaml`, else `~/.config/dozer/config.yaml`, else the built-in default (three shells, layout `2,1`) |
| `NAME` | A file called `NAME` if there is one, else the profile `~/.config/dozer/NAME.yaml` (or `.yml`) |
| `path/to/file.yaml` | That file |

`~/.config` is `$XDG_CONFIG_HOME` when that's set. See [`CONFIG.md`](CONFIG.md) §6 for the full lookup and override rules.

## Flags

### What to run

| Flag | Meaning |
|---|---|
| `-c FILE` | Load a config file: YAML, JSON, or a packaged `.sh` script. `-c -` reads it from stdin. |
| `--config-text TEXT` | The config itself, as YAML or JSON text. |
| `-l LAYOUT` | Layout shorthand: panes per row, e.g. `2,1`, `2,2,1`, `3`, `1,3` (at most 9 panes). |
| `--heights SIZES` | Row heights, e.g. `70,30`. Sizes: `60`/`60%` percent, `12c` cells, `2fr` weight, `auto`. |
| `--widths ROW:SIZES` | Widths in one row, e.g. `1:30,70`. Repeat it for more rows. |
| `-p CMD` | **Run** a command in the next pane, then leave a prompt (`run:`). Repeatable. |
| `-x CMD` | **Exec** a command as the next pane's process (`exec:`); when it ends, the pane is dead. Repeatable. |

`-p` and `-x` fill panes in reading order, in the order you give them: `-p htop -x 'tail -F log' -p 'git status'` sets panes 1, 2 and 3. Quote each command.

With no `-l` and no config (no `-c`, `--config-text`, profile or auto-found `.dozer.yaml`), 1–3 commands use the default `2,1` layout (unused panes get a shell). More commands than that fit the layout to them, in two rows: 4 → `2,2`, 5 → `3,2`, … 9 → `5,4`. With a config, the commands must fit its layout.

`-l`, `-p` and `-x` without a config also skip the auto-found `.dozer.yaml` and `config.yaml`; piped commands don't.

### Settings

| Flag | Meaning |
|---|---|
| `--prefix KEY` | The command key: `C-a` (default) … `C-z`. |
| `--no-mouse` | Leave the mouse to your terminal (same as `mouse: false`). |
| `--quit-when-all-exited` | Quit once every pane has ended (default: stay, showing dead panes). |

### Instead of launching

| Flag | Does |
|---|---|
| `--check` | Validate the config and print what would launch, then exit. |
| `--save FILE` | Write the resolved config as YAML to `FILE` (`-` = stdout), then exit. See [`PACKAGING.md`](PACKAGING.md). |
| `--package NAME` | Write `NAME.sh`, a self-contained runnable script with the config inside, then exit. See [`PACKAGING.md`](PACKAGING.md). |

`--check` with `--save` prints the check and then saves. `--check` with `--package` only prints the check.
| `-version` | Print the version and exit. |

### For development

| Flag | Meaning |
|---|---|
| `-emu NAME` | Terminal emulator back end: `charm` (default) or `vt10x` (a limited comparison back end: no scrollback, no wide characters). |

## Piped input

If stdin is a pipe (and not used for `-c -`), dozer reads it as **one command per line**, like repeated `-p`:

```sh
printf 'htop\ntail -F app.log\ngit status\n' | dozer
```

- Blank lines and lines starting with `#` are skipped.
- Piped commands come after any `-p`/`-x` flags.
- After reading the pipe, dozer reads the keyboard from the terminal (`/dev/tty`), so it stays interactive.

## Examples

```sh
dozer                                   # default: three shells, 2 on top, 1 below
dozer -l 2,2,1                          # five shells
dozer -l 2,1 -p htop -p 'git log'       # commands fill panes in reading order
dozer -l 1 -x 'tail -F /var/log/syslog' # one pane that is the command
dozer -p a -p b -p c -p d               # no -l: fitted to 2,2
dozer -l 2,1 --heights 70,30 --widths 1:30,70
dozer work                              # profile ~/.config/dozer/work.yaml
dozer -c work.yaml --prefix C-b         # a config, with an override
dozer --check -c work.yaml              # validate without launching
dozer -l 2,1 -p htop --save work.yaml   # a command line into a config file
dozer -c work.yaml --package work       # a config into ./work.sh
dozer -c - < work.yaml                  # config from stdin
```

## Environment

Read by dozer:

| Variable | Used for |
|---|---|
| `SHELL` | Each pane's shell, unless the config sets `shell:` (fallback `/bin/sh`). |
| `XDG_CONFIG_HOME` | Where profiles live (`$XDG_CONFIG_HOME/dozer/`; default `~/.config/dozer/`). |
| `SSH_CONNECTION`, `SSH_TTY` | Over ssh, copying doesn't use the local clipboard command (it would fill the remote machine's clipboard). |
| `DISPLAY`, `WAYLAND_DISPLAY` | On Linux, which clipboard command to use (`wl-copy`, `xclip`, `xsel`). |
| `DOZER_BIN` | Packaged scripts only: the dozer to run instead of the one on `PATH`. |

Set by dozer for every pane: `TERM`, `COLORTERM`, `TERM_PROGRAM`, `TERM_PROGRAM_VERSION`, `DOZER`, `DOZER_PANE`, `DOZER_NAME`, `DOZER_DESCRIPTION` (see [`CONFIG.md`](CONFIG.md) §4.6).

## Exit status

| Status | Meaning |
|---|---|
| 0 | Quit normally (`Ctrl-a q`), or `--check`/`--save`/`--package`/`-version` succeeded |
| 1 | No terminal to run in, or a runtime error (e.g. `--save` couldn't write the file) |
| 2 | The config or flags are invalid (the message says what and where) |
| 127 | Packaged scripts only: dozer wasn't found |

## Keys and mouse

Keys, copy mode and the mouse are covered in [`QUICKSTART.md`](QUICKSTART.md); the complete key table is in [`SPEC.md`](SPEC.md) §4.4.
