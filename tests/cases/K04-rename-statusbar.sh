#!/bin/sh
# K04-rename-statusbar: a dozer workspace, packaged by dozer 849a950-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./K04-rename-statusbar.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./K04-rename-statusbar.sh --prefix C-b
#   ./K04-rename-statusbar.sh --show-config    print the embedded config
#
# To change it: ./K04-rename-statusbar.sh --show-config > K04-rename-statusbar.yaml, edit, then
#   dozer -c K04-rename-statusbar.yaml --package K04-rename-statusbar.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# K04: Rename (t) and status bar toggle (s)  [M2]
#
# Steps:
#   1. Ctrl-a t, type a new name, Enter.
#   2. Ctrl-a t, Ctrl-u, Enter.
#   3. Ctrl-a s, then Ctrl-a (wait), Esc, Ctrl-a s.
#
# Expect:
#   - The title and status bar show the new name; the cursor sits in the prompt while typing.
#   - An empty name restores the default label.
#   - The bar hides (panes gain a row), reappears while PREFIX is pending, then returns.
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: K04-rename-statusbar
description: |
  K04: Rename (t) and status bar toggle (s)  [M2]

  Steps:
    1. Ctrl-a t, type a new name, Enter.
    2. Ctrl-a t, Ctrl-u, Enter.
    3. Ctrl-a s, then Ctrl-a (wait), Esc, Ctrl-a s.

  Expect:
    - The title and status bar show the new name; the cursor sits in the prompt while typing.
    - An empty name restores the default label.
    - The bar hides (panes gain a row), reappears while PREFIX is pending, then returns.

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
layout: "2"
panes:
  - {title: guide, run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  - {title: rename me, run: echo 'rename me'}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "K04-rename-statusbar: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "K04-rename-statusbar: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
