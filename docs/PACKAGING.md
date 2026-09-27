# Saving and sharing workspaces

dozer can turn any workspace (a config file, a long command line, or both) into something you keep or hand to someone else:

| You want | Use | You get |
|---|---|---|
| An editable config file | `--save FILE.yaml` | YAML you can read, edit, keep in a repo, or use as a profile |
| One file anyone with dozer can run | `--package NAME` | `NAME.sh`, a self-contained script with the config inside |
| A config with no file at all | `-c -` or `--config-text` | dozer reads the config from stdin or an argument |

How to write the config itself is in [`CONFIG.md`](CONFIG.md).

---

## 1. `--save`: a config file from anything

`--save` resolves everything dozer was given (config file, profile, flags, piped commands, defaults) into one config file, then exits without launching.

```sh
dozer -l 2,1 -p htop -x 'tail -F app.log' --heights 70,30 --save work.yaml
dozer -c work.yaml                     # same workspace, from the file
dozer -c work.yaml --prefix C-b --save work.yaml   # change a setting and re-save
dozer ... --save -                     # print it instead of writing a file
```

The file for the first command:

```yaml
# Saved by dozer v0.4.0 --save on 2026-09-27 09:16.
# Configuration only: layout, sizes, display settings and each pane's launch command.
version: 1
layout: "2,1"
heights: [70%, 30%]
panes:
  - {run: htop}
  - {exec: tail -F app.log}
  - {}
```

What it saves, and what it doesn't:

- **Saved:** the layout and sizes, every pane's `run:`/`exec:`, title, `cwd`, `env`, `shell` and restart policy, and the workspace settings (`name`, `description`, `prefix`, `status_bar`, `mouse`, `scrollback`, `min_pane`, `quit_when_all_exited`). Only values that differ from the defaults are written, so the file stays short.
- **Written as you wrote them:** `cwd: ~/work` stays `~/work` and `$HOME/x` stays `$HOME/x`, not your machine's expanded paths. The file works for someone else.
- **Not saved:** anything about the running session: scrollback, what's on screen, processes, shell history, or changes made at runtime (resizing with `Ctrl-a H J K L`, renaming with `Ctrl-a t`). dozer is a launcher; it never tries to restore a session. (Saving runtime changes with a key is a possible later addition, SPEC §4.9.)

To use a saved file as a **profile**, put it in `~/.config/dozer/`:

```sh
cp work.yaml ~/.config/dozer/work.yaml
dozer work
```

---

## 2. `--package`: one runnable file

```sh
dozer -c work.yaml --package work      # writes ./work.sh
./work.sh                              # launches the workspace
```

`--package NAME` writes `NAME.sh` (the `.sh` is added if you leave it off, and a path like `tools/work` works, if the folder exists). The package's name (the file name without `.sh`) becomes the workspace's `name`, replacing any `name:` in the config. Anything dozer can launch can be packaged: a config file, a profile, flags, or a mix.

### What the script does

The script **is** the whole package: the config is inside it, and running it reads and writes no other file. When you run it, it:

1. **Checks for dozer:** `dozer` on your `PATH`, or the program named by `$DOZER_BIN`. If it's missing, it says where to get dozer and exits with status 127.
2. **Checks for a terminal:** if its output isn't a terminal (say, it was run from a pipe or a cron job), it says so and exits with status 1.
3. **Runs dozer:** it pipes the embedded config into `dozer -c -`, exactly like `dozer -c - < work.yaml`, and dozer reads the keyboard from the terminal.

### Using a package

| Command | Does |
|---|---|
| `./work.sh` | Launch the workspace |
| `./work.sh --prefix C-b --no-mouse` | Launch it with extra dozer flags (any flag passes through) |
| `./work.sh --show-config` | Print the embedded config, without launching |
| `DOZER_BIN=~/src/dozer/bin/dozer ./work.sh` | Use a dozer that isn't on your `PATH` |
| `dozer --check -c work.sh` | Validate it (dozer reads packages as configs too) |
| `dozer -c work.sh` | Launch it through dozer directly |

### Changing a package

```sh
./work.sh --show-config > work.yaml    # get the config out
$EDITOR work.yaml
dozer -c work.yaml --package work      # rewrite work.sh
```

`--package` only overwrites files that are already dozer packages. If `work.sh` is some other script, dozer refuses and leaves it alone.

### What a package looks like

A package is a short, readable POSIX `sh` script:

```sh
#!/bin/sh
# work: a dozer workspace, packaged by dozer v0.4.0 on 2026-09-27.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#   ... usage notes ...
#
# (the workspace's description, if it has one)

yaml() {
cat <<'DOZER_YAML'
version: 1
name: work
layout: "2,1"
panes:
  - {run: htop}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "work: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "work: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
```

The config sits in a **quoted** heredoc (`<<'DOZER_YAML'`), so the shell never expands anything in it: `$HOME`, backquotes and `$(…)` reach dozer exactly as written.

### Paths in a package

A package makes no assumptions about where it lives or where it's run:

- It never refers to its own location.
- `cwd` and `env` are embedded **as written**, and resolve when the package runs:
  - a relative `cwd` (`api`, `../shared`) resolves against **the folder you run the script from**, so a package for a project works when run from the project root (if you load it as a file instead, `dozer -c tools/work.sh`, it resolves against the script's folder, like any config file);
  - `~` and `$VARS` resolve for **whoever runs it**, on their machine;
  - absolute paths stay absolute.

So `cwd: ~/src/shop` in a package you send to a colleague opens *their* `~/src/shop`.

### Sharing packages

- A package needs only dozer. Send the `.sh` file, or commit it to a repo (a `tools/` folder works well).
- It's a script that runs commands, so read it before running one you were sent: `./tool.sh --show-config` shows exactly what each pane will run.
- Packages record the dozer version that made them, in the header comment. A package made by an older dozer runs on a newer one as long as the config format (`version: 1`) is the same.
- Whether a package could carry dozer itself, for machines without it, is an open design question (SPEC DP-6).

---

## 3. Configs without a file: `-c -` and `--config-text`

This is the mechanism packages use, and you can use it directly:

```sh
dozer -c - < work.yaml                                  # config on stdin
some-generator | dozer -c -                             # from another program
dozer --config-text '{"layout":"3","panes":[{"run":"htop"},{"exec":"tail -F app.log"}]}'
```

- Both accept YAML or JSON (JSON is a subset of YAML).
- With no file behind the config, a relative `cwd` resolves against the folder dozer runs in.
- After reading the config from stdin, dozer reads the keyboard from the terminal (`/dev/tty`), so piping a config in doesn't take over your typing.
- `dozer -c - < work.sh` also works: dozer finds the config inside a package.

---

## 4. Which to use

- **Just for you, on this machine:** a profile (`~/.config/dozer/NAME.yaml`, then `dozer NAME`), or a `.dozer.yaml` in the project folder (then plain `dozer` there).
- **For a project, in its repo:** a `.dozer.yaml` at the root (relative `cwd` resolves against the file's folder), or a package in `tools/`.
- **To hand to someone:** a package. One file, nothing to install beyond dozer, and `--show-config` lets them see what it does.
- **From a script or another tool:** `-c -` or `--config-text`.
