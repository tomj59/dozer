#!/bin/sh
# K06-statusbar-top: a dozer workspace, packaged by dozer 849a950-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./K06-statusbar-top.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./K06-statusbar-top.sh --prefix C-b
#   ./K06-statusbar-top.sh --show-config    print the embedded config
#
# To change it: ./K06-statusbar-top.sh --show-config > K06-statusbar-top.yaml, edit, then
#   dozer -c K06-statusbar-top.yaml --package K06-statusbar-top.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# K06: Status bar at the top  [M2]
#
# Steps:
#   1. Launch; look at the first row.
#   2. Ctrl-a s twice.
#
# Expect:
#   - The status bar is the first row; pane titles start on row 2.
#   - Toggling hides and restores it at the top.
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: K06-statusbar-top
description: |
  K06: Status bar at the top  [M2]

  Steps:
    1. Launch; look at the first row.
    2. Ctrl-a s twice.

  Expect:
    - The status bar is the first row; pane titles start on row 2.
    - Toggling hides and restores it at the top.

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
status_bar: top
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
	echo "K06-statusbar-top: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "K06-statusbar-top: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
