#!/bin/sh
# K05-prefix-ctrl-b: a dozer workspace, packaged by dozer dc640fc-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./K05-prefix-ctrl-b.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./K05-prefix-ctrl-b.sh --prefix C-b
#   ./K05-prefix-ctrl-b.sh --show-config    print the embedded config
#
# To change it: ./K05-prefix-ctrl-b.sh --show-config > K05-prefix-ctrl-b.yaml, edit, then
#   dozer -c K05-prefix-ctrl-b.yaml --package K05-prefix-ctrl-b.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# K05: Remapped prefix (C-b)  [M1]
#
# Steps:
#   1. Try Ctrl-a q: nothing should happen (Ctrl-a goes to the shell).
#   2. Ctrl-b →, Ctrl-b z, Ctrl-b z.
#   3. Ctrl-b Ctrl-b in a shell.
#   4. Ctrl-b q, y.
#
# Expect:
#   - Ctrl-a behaves as in a plain shell (start of line).
#   - Ctrl-b is the prefix: focus and zoom work; the status bar hints say C-b.
#   - Ctrl-b Ctrl-b sends one Ctrl-b (cursor back one char).
#   - Quits.
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: K05-prefix-ctrl-b
description: |
  K05: Remapped prefix (C-b)  [M1]

  Steps:
    1. Try Ctrl-a q: nothing should happen (Ctrl-a goes to the shell).
    2. Ctrl-b →, Ctrl-b z, Ctrl-b z.
    3. Ctrl-b Ctrl-b in a shell.
    4. Ctrl-b q, y.

  Expect:
    - Ctrl-a behaves as in a plain shell (start of line).
    - Ctrl-b is the prefix: focus and zoom work; the status bar hints say C-b.
    - Ctrl-b Ctrl-b sends one Ctrl-b (cursor back one char).
    - Quits.

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
prefix: C-b
layout: "2"
panes:
  - {title: guide, run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  - {title: shell, run: 'echo ''[2]'''}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "K05-prefix-ctrl-b: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "K05-prefix-ctrl-b: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
