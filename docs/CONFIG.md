# Writing dozer configurations

A **config** describes a workspace: how the window is divided into panes, and what each pane runs. It's a small YAML file. This guide goes from a first config to every key dozer understands.

- New to dozer? Read [`QUICKSTART.md`](QUICKSTART.md) first. It covers the keys and the mouse.
- Want to share a workspace as one runnable file? See [`PACKAGING.md`](PACKAGING.md).
- Looking for a command-line flag? See [`CLI.md`](CLI.md).

Contents:

1. [A first config, step by step](#1-a-first-config-step-by-step)
2. [How dozer thinks about panes](#2-how-dozer-thinks-about-panes)
3. [Layouts](#3-layouts)
4. [Panes](#4-panes)
5. [Workspace settings](#5-workspace-settings)
6. [Where configs come from, and what overrides what](#6-where-configs-come-from-and-what-overrides-what)
7. [Checking a config](#7-checking-a-config)
8. [Recipes](#8-recipes)
9. [Reference: every key](#9-reference-every-key)

---

## 1. A first config, step by step

**Step 1: the smallest config.** An empty file is a valid config: it gives the same workspace as a bare `dozer` (three shells in the `2,1` layout). Start with a name:

```yaml
name: api
```

**Step 2: pick a layout.** `"2,1"` means two panes in the top row and one below. Quote it, so YAML reads it as text.

```yaml
name: api
layout: "2,1"
```

**Step 3: say what each pane runs.** Panes are listed in reading order: left to right, top to bottom. A pane you don't list gets a plain shell.

```yaml
name: api
layout: "2,1"
panes:
  - title: server
    exec: npm run dev            # the pane *is* the dev server
  - title: git
    run: git status              # runs, then leaves you at a prompt
  - title: shell                 # a plain shell
```

**Step 4: size things.** Give the top row 70% of the height:

```yaml
heights: [70, 30]
```

**Step 5: check it, then run it.**

```sh
dozer --check api.yaml     # validate and show what would launch
dozer -c api.yaml          # launch
```

To use it by name from anywhere, copy it to `~/.config/dozer/api.yaml` and run `dozer api` (§6).

---

## 2. How dozer thinks about panes

- **A layout is a set of slots.** Each slot is a pane: a title line on top, and the pane's terminal below it. Side-by-side panes are separated by a one-column divider.
- **Panes are numbered in reading order**, `[1]` to `[9]`: left to right, then top to bottom (for tree layouts: the order the `pane:` leaves appear). The number is what `Ctrl-a 1`…`9` focuses, and what the `panes:` list follows.
- **Every pane is a real terminal** running one process: a shell, a shell that ran a command first (`run:`), or a command on its own (`exec:`).
- **Slots are permanent.** When a pane's process ends, the pane stays where it is, flagged as dead, until you restart it (`Ctrl-a r`) or a restart policy does (§4.7). The layout never reshuffles.
- **At most 9 panes** per workspace.

---

## 3. Layouts

There are two ways to write a layout: the **shorthand** covers rows of panes; the **tree** covers anything else.

### 3.1 Shorthand: panes per row

A comma-separated count of panes in each row, top to bottom.

| `layout:` | Result |
|---|---|
| `"1"` | One pane |
| `"2"` | Two side by side |
| `"3"` | Three side by side |
| `"2,1"` (the default, also `default`) | Two on top, one full-width below |
| `"2,2,1"` | Two, two, then one full-width |
| `"1,3"` | One wide on top, three below |
| `"3,3,3"` | A 3×3 grid (the maximum: 9 panes) |

Rows share the height equally and panes share their row's width equally, unless you give sizes.

### 3.2 Sizes

Sizes appear in `heights`, `widths` and the tree's `size:`. Each one is:

| Written | Means |
|---|---|
| `60` or `60%` | 60% of the space (a bare number is a percent) |
| `12c` | exactly 12 cells (rows, or columns) |
| `2fr` | a **weight**: a share of what's left after `%` and `c` sizes |
| `auto` | the same as `1fr` (also what you get with no size at all) |

How they combine, within one split:

1. Percent and cell sizes take their space first.
2. Weights (`fr`, `auto`) share whatever is left, in proportion.
3. If the numbers don't add up (say, `30%` and `30%` with nothing else), dozer scales them to fill the space, keeping their proportions.
4. No pane gets smaller than `min_pane` (default 20 columns × 5 rows of content). If the window is too small for everything, dozer keeps the minimum and lets you scroll the view to the focused pane (the status bar shows `more ◀ ▲ ▼ ▶`).

Widths are shared after the one-column dividers are taken out. A pane's height includes its title line, so a `12c` row shows 11 rows of terminal.

**Tip:** mix fixed and flexible sizes. `[24c, 1fr, 2fr]` gives a 24-column pane, then splits the rest 1:2.

### 3.3 `heights` and `widths` (for shorthand layouts)

```yaml
layout: "2,3"
heights: [70%, 30%]          # one size per row
widths:
  1: [30%, 70%]              # row 1: one size per pane
  2: [24c, 1fr, 2fr]         # row 2
```

- `heights` has one size per row; `widths` is keyed by row number (1 = top) and has one size per pane in that row. Leave a row out to split it equally.
- They only apply to the shorthand. With a tree layout, put `size:` on the tree's nodes instead (dozer tells you if you mix them).
- The same settings exist as flags: `--heights 70,30` and `--widths 1:30,70` (see [`CLI.md`](CLI.md)).

### 3.4 Tree form: any arrangement

When rows of panes aren't enough, for example a tall pane on the left with two stacked on the right, write the layout as a tree. Each node is exactly one of:

- `{rows: [ … ]}`: its children stacked top to bottom
- `{cols: [ … ]}`: its children side by side
- `{pane: NAME}`: a pane slot, with a name you choose

Any node can also have `size:` (its size along its parent's direction).

```yaml
#   ┌──────── edit ────────┬──── logs ────┐
#   │                      │              │
#   │  (60% wide)          ├──── shell ───┤
#   │                      │  (12 rows)   │
#   └──────────────────────┴──────────────┘
layout:
  cols:
    - pane: edit
      size: 60%
    - rows:
        - pane: logs
        - pane: shell
          size: 12c
panes:                       # keyed by the names in the layout
  edit:  { run: vim . }
  logs:  { exec: tail -F app.log }
  shell: {}
```

Rules:

- Pane names must be unique. They're only labels for the config: panes are still numbered in the order their leaves appear (`edit` is `[1]`, `logs` `[2]`, `shell` `[3]`).
- With a tree, `panes:` is usually a **map** keyed by those names, and a pane listed there is titled with its name unless you set `title:`. (A list in reading order also works; then titles follow the usual defaults, §4.3.)
- A pane named in the layout but missing from `panes:` gets a plain shell, titled with the shell's name.
- Nesting can go as deep as you like; the 9-pane limit still applies.

The same layout in flow style, which some people find easier to read:

```yaml
layout: {cols: [{pane: edit, size: 60%}, {rows: [{pane: logs}, {pane: shell, size: 12c}]}]}
```

### 3.5 Which to use

Use the shorthand when your layout is rows of panes. It's shorter, and it works with `--heights`/`--widths` on the command line. Use the tree for columns that span several rows, or any nesting. `dozer --check` prints the tree dozer built either way, for example `rows(cols([1] [2])@70% [3]@30%)`.

---

## 4. Panes

### 4.1 Listing panes

`panes:` is either a **list** in reading order or, with a tree layout, a **map** by name.

```yaml
panes:
  - {}                         # [1] plain shell
  - "git log --oneline -20"    # [2] a bare string means run:
  - title: logs                # [3] the full form
    exec: tail -F app.log
```

- You can list fewer panes than the layout has; the rest are plain shells.
- Listing more panes than the layout has is an error.
- `{}` is a pane with nothing set: a plain shell with the workspace defaults.

### 4.2 What a pane runs: `run:`, `exec:` or a shell

| Pane has | dozer starts | When the command ends |
|---|---|---|
| nothing | `$SHELL -l` (an interactive login shell) | The pane is dead (`exit` in the shell) |
| `run: CMD` | `$SHELL -ic 'trap : INT; CMD; exec $SHELL'` | You're back at a prompt in the same pane |
| `exec: CMD` | `$SHELL -lc 'CMD'` | The pane is dead, and shows how it ended |

**`run:`** is for things you'd type yourself: `git status`, `ssh host`, `make watch`, `htop`. The command runs in an interactive shell, so your aliases and functions work, and when it finishes (or you stop it with Ctrl-C) you're left at a prompt, still in that pane.

**`exec:`** is for things the pane exists to run: a server, a log tail, a monitor. The pane *is* the command. When it ends, the pane is flagged dead so you notice (`exited`, `✖ exit 3`, `✖ SIGKILL`…), and it can restart automatically (§4.7).

A pane has one or the other, not both. Commands are shell code, so pipes, `&&`, loops and quotes all work:

```yaml
- exec: 'while true; do date +%T; sleep 1; done'
- run: cd api && make watch
```

**Quoting tip:** YAML treats `: ` and ` #` specially. If a command contains them, or starts with `{`, `[`, `*`, `&` or quotes, put the whole command in single quotes (and double any single quote inside: `'it''s'`).

### 4.3 `title`

The label on the pane's title line. Default: the command (`run:`/`exec:`), or the shell's name (`zsh`). A pane given in a `panes:` map (tree layouts) defaults to its name. A program can set its own title (with the standard terminal escape sequence) when you haven't set one. `Ctrl-a t` renames a pane while dozer runs.

### 4.4 `shell`

The shell used for the pane: to run `run:` and `exec:` commands, and as the plain shell. Default: `$SHELL`, then `/bin/sh`. Set it at the top level for every pane, or per pane:

```yaml
shell: /bin/bash
panes:
  - {}
  - { shell: /usr/bin/fish }
```

### 4.5 `cwd`: where a pane starts

The folder a pane starts in. Set it at the top level (for every pane), per pane, or both (the pane's wins).

| Written | Resolves to |
|---|---|
| (not set) | The folder you launched dozer from |
| `~`, `~/work` | Your home folder, or a folder inside it |
| `$HOME/x`, `$PROJECTS/api` | Environment variables are expanded |
| `/srv/app` | Absolute: used as is |
| `api`, `../shared` | **Relative:** see below |

A relative `cwd` resolves against:

- **the config file's folder**, for a file loaded with `-c` or as a profile. So a `.dozer.yaml` in a project can say `cwd: frontend` and work from anywhere.
- **the folder dozer runs in**, for a config with no file behind it: `-c -` (stdin), `--config-text`, and a packaged script you run (`./tool.sh`). (Loading a package as a file, `dozer -c tool.sh`, counts as a file: its folder.)

A folder that doesn't exist makes that pane fail to start; the pane shows the error and is flagged dead.

### 4.6 `env`: environment variables

Extra variables for the pane's process, as a map. Top-level `env` applies to every pane; a pane's own `env` adds to it (and wins for the same name).

```yaml
env: { STAGE: dev }
panes:
  - { env: { PORT: "8080", LOG: "$HOME/logs/api.log" } }
```

- Values are strings. Quote numbers and anything YAML might reinterpret (`"8080"`, `"yes"`, `"on"`).
- `$VARS` in values are expanded from dozer's own environment when the config is loaded.

Every pane also gets these from dozer, which programs can use:

| Variable | Value |
|---|---|
| `TERM` | `xterm-256color` |
| `COLORTERM` | `truecolor` |
| `TERM_PROGRAM` / `TERM_PROGRAM_VERSION` | `dozer` / its version |
| `DOZER` | `1` (you're inside dozer) |
| `DOZER_PANE` | The pane's number, `1`–`9` |
| `DOZER_NAME` | The workspace's `name` |
| `DOZER_DESCRIPTION` | The workspace's `description` |

Variables that identify the *outer* terminal (`TERM_SESSION_ID`, `ITERM_SESSION_ID`, `TMUX`, `KITTY_WINDOW_ID` and similar) are removed, because inside a pane, dozer is the terminal. Without this, Terminal.app's shell integration made every pane restore the same saved session.

### 4.7 Dead panes and automatic restarts: `restart`, `max_restarts`

By default nothing restarts on its own: a dead pane waits for you (`Ctrl-a r` restarts the focused pane, `Ctrl-a R` every dead pane). An `exec:` pane can opt in:

| `restart:` | Restarts when the command… |
|---|---|
| `never` (default) | never |
| `on-failure` | fails: non-zero exit, a signal, or a lost ssh connection |
| `always` | ends, however it ended |

```yaml
- title: api
  exec: ./bin/server
  restart: on-failure
  max_restarts: 10          # default 5; or "unlimited"
```

- Restarts back off: 1 s, 2 s, 4 s, 8 s, 16 s, then every 30 s. The pane shows `↻ restarting automatically (2/5)…`.
- After `max_restarts` restarts in a row, dozer gives up and leaves the pane dead (`gave up after 5 automatic restarts`). A run that stays up for a minute resets the count, so an occasional crash never uses up the budget.
- Killing a pane on purpose (`Ctrl-a x`) never triggers a restart.
- `restart` only applies to `exec:` panes (a `run:` pane already falls back to a prompt). `max_restarts` needs `restart: on-failure` or `always`.
- An `exec: ssh …` pane that exits with 255 (connection refused or dropped) shows `✖ connection lost`.

---

## 5. Workspace settings

All optional, at the top level.

| Key | Default | What it does |
|---|---|---|
| `version` | `1` | The config format version. Only `1` exists; other values are an error. |
| `name` | none | The workspace's name: shown in the status bar, given to panes as `$DOZER_NAME`. `--package` replaces it with the package's name. |
| `description` | none | Free text: what this workspace is for. Shown by `--check`, written into package headers, and given to panes as `$DOZER_DESCRIPTION` (a pane can display it: `run: printf '%s\n' "$DOZER_DESCRIPTION"`). Use `\|` for several lines. |
| `prefix` | `C-a` | The command key: `C-a` … `C-z` (also written `^b` or `ctrl-b`). |
| `status_bar` | `bottom` | `bottom`, `top` or `off`. `Ctrl-a s` toggles it at runtime. |
| `min_pane` | `{w: 20, h: 5}` | The smallest pane content size. A window too small for the layout scrolls instead of squashing panes. |
| `quit_when_all_exited` | `false` | `true` quits dozer once every pane has ended. By default dozer stays, showing the dead panes, and quits only with `Ctrl-a q`. |
| `mouse` | `true` | `false` leaves the mouse to your terminal: no click-to-focus, border dragging, wheel history or pane-aware selection. |
| `scrollback` | `10000` | History lines kept per pane (0 to 1,000,000), for `Ctrl-a [` and the wheel. |
| `shell`, `cwd`, `env` | (see §4) | Defaults for every pane. |

---

## 6. Where configs come from, and what overrides what

**Loading.** dozer takes its config from, in order:

| You run | dozer loads |
|---|---|
| `dozer -c FILE` | That file (a YAML file, or a packaged `.sh` script) |
| `dozer -c -` | YAML or JSON from stdin |
| `dozer --config-text '…'` | YAML or JSON given as the argument |
| `dozer NAME` | A file called `NAME` if there is one, else the **profile** `~/.config/dozer/NAME.yaml` (or `.yml`) |
| `dozer` (nothing else) | `./.dozer.yaml` if the current folder has one, else `~/.config/dozer/config.yaml`, else built-in defaults |

`~/.config` is `$XDG_CONFIG_HOME` when that's set.

`dozer -l …`, `dozer -p …` or `dozer -x …` without a config ignores `.dozer.yaml` and `config.yaml`: the flags alone describe the workspace. Piped commands don't: they apply on top of an auto-found config.

**Overrides.** Command-line flags apply on top of the config:

| Flag | Effect on a loaded config |
|---|---|
| `-l 2,2,1` | Replaces the layout. Listed panes keep their settings by position. |
| `--heights`, `--widths` | Replace those sizes (shorthand layouts). |
| `-p CMD` / `-x CMD` | Set what panes 1, 2, … run (`run:` / `exec:`), keeping each pane's `cwd`, `env` and `shell`. The title and any restart policy are cleared. |
| piped lines | Like `-p`, one per line (see [`CLI.md`](CLI.md)). |
| `--prefix C-b` | Replaces `prefix`. |
| `--quit-when-all-exited` | Sets `quit_when_all_exited: true`. |
| `--no-mouse` | Sets `mouse: false`. |

`dozer --save FILE` writes the result of all this as one config file, so you can turn any command line into a config (see [`PACKAGING.md`](PACKAGING.md)).

**JSON.** JSON is valid YAML, so any config can also be written as JSON: `{"layout": "3", "panes": [{"run": "htop"}]}`. That's handy with `--config-text` and in scripts.

---

## 7. Checking a config

```sh
dozer --check api.yaml
dozer --check -c - < api.yaml
```

`--check` validates the config and prints what would launch, without launching anything:

```
config: api.yaml
name:   api
prefix: C-a   min pane: 20x5   status bar: bottom   quit when all exited: false
mouse: true   scrollback: 10000 lines
layout: rows(cols([1] [2])@70% [3]@30%)
  [1] server         exec: npm run dev
  [2] git            run:  git status
  [3] shell          shell
```

**Errors** stop dozer and name the line: a misspelled key (`line 3: field layuot not found`), a pane with both `run:` and `exec:`, more panes than the layout has, `restart:` on a `run:` pane, a bad size (`bad size "30x" (use 60, 60%, 20c, 2fr or auto)`), and so on. Unknown keys are always errors, so typos can't pass silently.

**Warnings** are for keys from later milestones (`multi_input`, `keys`; pane `readonly`, `group`, `min`). They're accepted and ignored for now, and `--check` lists them.

Every file in `examples/` is loaded by the test suite, so they're always valid starting points.

---

## 8. Recipes

**A project workspace** (`.dozer.yaml` at the project root; `dozer` in that folder opens it):

```yaml
name: shop
layout: "2,1"
heights: [65, 35]
panes:
  - { title: editor, run: vim . }
  - { title: server, cwd: backend, exec: make run, restart: on-failure }
  - { title: shell }
```

**A service you want to keep alive, with a limit:**

```yaml
name: worker
layout: "1"
panes:
  - title: queue worker
    exec: ./worker --verbose
    restart: always
    max_restarts: unlimited
```

**Watching servers** (edit the hosts; `Ctrl-a r` reconnects a dropped one):

```yaml
name: servers
layout: "2,2"
panes:
  - { title: web-1, exec: ssh web-1 }
  - { title: web-2, exec: ssh web-2 }
  - { title: db,    exec: ssh db-1 }
  - { title: local }
```

**A dashboard of live output:**

```yaml
name: monitor
layout: "2,2,1"
panes:
  - { title: top,  exec: top }
  - { title: disk, exec: 'while true; do clear; df -h; sleep 5; done' }
  - { title: net,  exec: ping 1.1.1.1 }
  - { title: logs, exec: tail -F /var/log/system.log }
  - { title: shell }
```

**A workspace that explains itself** (the pattern the manual test cases use):

```yaml
name: onboarding
description: |
  Welcome! Pane 2 runs the app, pane 3 the tests.
  Ctrl-a 2 / Ctrl-a 3 to switch; Ctrl-a q to quit.
layout: "2,1"
panes:
  - { title: guide, run: 'printf "%s\n" "$DOZER_DESCRIPTION"' }
  - { title: app,   exec: npm start }
  - { title: tests, run: npm test -- --watch }
```

More, each explained at the top of the file, in [`examples/`](../examples): `reference.yaml` (every key with its default), `dev.yaml`, `monitor.yaml`, `tree.yaml`, `lifecycle.yaml`, `ssh.yaml`.

---

## 9. Reference: every key

### Top level

| Key | Type | Default | Notes |
|---|---|---|---|
| `version` | integer | `1` | Only `1` is accepted. |
| `name` | text | none | §5 |
| `description` | text | none | §5 |
| `layout` | text or tree | `"2,1"` | §3.1, §3.4 |
| `heights` | list of sizes | equal | Shorthand only; one per row. §3.3 |
| `widths` | map: row → list of sizes | equal | Shorthand only. §3.3 |
| `panes` | list, or map by name | plain shells | §4.1 |
| `shell` | path | `$SHELL`, then `/bin/sh` | Default for every pane. §4.4 |
| `cwd` | path | launch folder | Default for every pane. §4.5 |
| `env` | map of text | none | For every pane. §4.6 |
| `prefix` | `C-a`…`C-z` | `C-a` | §5 |
| `status_bar` | `bottom` \| `top` \| `off` | `bottom` | §5 |
| `min_pane` | `{w: N, h: N}` | `{w: 20, h: 5}` | Both ≥ 1. §5 |
| `quit_when_all_exited` | boolean | `false` | §5 |
| `mouse` | boolean | `true` | §5 |
| `scrollback` | integer 0–1,000,000 | `10000` | §5 |
| `multi_input` | (reserved) | | Accepted with a warning; planned for M5. |
| `keys` | (reserved) | | Accepted with a warning; key remapping is planned. |

### Pane

| Key | Type | Default | Notes |
|---|---|---|---|
| `title` | text | the command, or the shell's name (tree: the pane's name) | §4.3 |
| `run` | shell command | none | Then a prompt. Not with `exec`. §4.2 |
| `exec` | shell command | none | The pane is the command. Not with `run`. §4.2 |
| `shell` | path | the top-level `shell` | §4.4 |
| `cwd` | path | the top-level `cwd` | §4.5 |
| `env` | map of text | none | Added to the top-level `env`. §4.6 |
| `restart` | `never` \| `on-failure` \| `always` | `never` | `exec` panes only. §4.7 |
| `max_restarts` | integer ≥ 1, or `unlimited` | `5` | Needs `restart`. §4.7 |
| `readonly`, `group`, `min` | (reserved) | | Accepted with a warning; planned for M5. |

A pane can also be a bare string, which means `run:`.

### Layout tree node

| Key | Type | Notes |
|---|---|---|
| `rows` | list of nodes | Children stacked top to bottom. |
| `cols` | list of nodes | Children side by side. |
| `pane` | name | A pane slot. Names are unique. |
| `size` | size | This node's share of its parent's direction. |

A node has exactly one of `rows`, `cols` or `pane`, plus an optional `size`.

### Sizes

`60` or `60%` (percent) · `12c` (cells) · `2fr` (weight) · `auto` (= `1fr`). §3.2
