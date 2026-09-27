#!/bin/sh
# D06-quit-confirm: a dozer workspace, packaged by dozer dc640fc-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./D06-quit-confirm.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./D06-quit-confirm.sh --prefix C-b
#   ./D06-quit-confirm.sh --show-config    print the embedded config
#
# To change it: ./D06-quit-confirm.sh --show-config > D06-quit-confirm.yaml, edit, then
#   dozer -c D06-quit-confirm.yaml --package D06-quit-confirm.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# D06: Quit confirmation  [M1]
#
# Steps:
#   1. Ctrl-a q, n.
#   2. Ctrl-a q, y.
#
# Expect:
#   - The first asks 'Quit dozer? 2 pane(s) still running…' and stays after 'n'.
#   - The second quits; the terminal is exactly as before (echo, cursor, colors).
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: D06-quit-confirm
description: |
  D06: Quit confirmation  [M1]

  Steps:
    1. Ctrl-a q, n.
    2. Ctrl-a q, y.

  Expect:
    - The first asks 'Quit dozer? 2 pane(s) still running…' and stays after 'n'.
    - The second quits; the terminal is exactly as before (echo, cursor, colors).

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
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
	echo "D06-quit-confirm: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "D06-quit-confirm: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
