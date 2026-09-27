#!/bin/sh
# L05-nine-panes: a dozer workspace, packaged by dozer dc640fc-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./L05-nine-panes.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./L05-nine-panes.sh --prefix C-b
#   ./L05-nine-panes.sh --show-config    print the embedded config
#
# To change it: ./L05-nine-panes.sh --show-config > L05-nine-panes.yaml, edit, then
#   dozer -c L05-nine-panes.yaml --package L05-nine-panes.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# L05: Maximum: nine panes (3,3,3)  [M1]
#
# Steps:
#   1. Launch in a large window.
#   2. Ctrl-a 9, then Ctrl-a 1.
#
# Expect:
#   - Nine panes, three by three, each with its own prompt.
#   - Focus jumps to the bottom-right, then back to the top-left.
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: L05-nine-panes
description: |
  L05: Maximum: nine panes (3,3,3)  [M1]

  Steps:
    1. Launch in a large window.
    2. Ctrl-a 9, then Ctrl-a 1.

  Expect:
    - Nine panes, three by three, each with its own prompt.
    - Focus jumps to the bottom-right, then back to the top-left.

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
layout: "3,3,3"
panes:
  - {title: guide, run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  - {title: shell, run: 'echo ''[2]'''}
  - {title: shell, run: 'echo ''[3]'''}
  - {title: shell, run: 'echo ''[4]'''}
  - {title: shell, run: 'echo ''[5]'''}
  - {title: shell, run: 'echo ''[6]'''}
  - {title: shell, run: 'echo ''[7]'''}
  - {title: shell, run: 'echo ''[8]'''}
  - {title: shell, run: 'echo ''[9]'''}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "L05-nine-panes: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "L05-nine-panes: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
