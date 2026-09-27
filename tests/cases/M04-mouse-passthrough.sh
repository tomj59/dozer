#!/bin/sh
# M04-mouse-passthrough: a dozer workspace, packaged by dozer effde5c-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./M04-mouse-passthrough.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./M04-mouse-passthrough.sh --prefix C-b
#   ./M04-mouse-passthrough.sh --show-config    print the embedded config
#
# To change it: ./M04-mouse-passthrough.sh --show-config > M04-mouse-passthrough.yaml, edit, then
#   dozer -c M04-mouse-passthrough.yaml --package M04-mouse-passthrough.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# M04: Mouse in mouse-aware programs  [M4]
#
# Steps:
#   1. In pane 2 (vim, mouse=a): click in the text, drag to select.
#   2. Shift-drag in vim.
#   3. In pane 3 (htop): click a column header or a process.
#
# Expect:
#   - vim gets the clicks: its cursor moves where you click and a drag makes a Visual selection.
#   - Shift-drag makes a dozer selection instead (copied on release).
#   - htop reacts to clicks at the right place (coordinates are pane-relative).
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: M04-mouse-passthrough
description: |
  M04: Mouse in mouse-aware programs  [M4]

  Steps:
    1. In pane 2 (vim, mouse=a): click in the text, drag to select.
    2. Shift-drag in vim.
    3. In pane 3 (htop): click a column header or a process.

  Expect:
    - vim gets the clicks: its cursor moves where you click and a drag makes a Visual selection.
    - Shift-drag makes a dozer selection instead (copied on release).
    - htop reacts to clicks at the right place (coordinates are pane-relative).

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
layout: "2,1"
panes:
  - {title: guide, run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  - {title: vim mouse=a, run: seq 1 200 > /tmp/dozer-m04.txt; vim -u NONE -N -c 'set mouse=a' /tmp/dozer-m04.txt}
  - {title: htop, run: htop || top}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "M04-mouse-passthrough: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "M04-mouse-passthrough: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
