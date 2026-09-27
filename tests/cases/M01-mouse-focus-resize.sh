#!/bin/sh
# M01-mouse-focus-resize: a dozer workspace, packaged by dozer effde5c-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./M01-mouse-focus-resize.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./M01-mouse-focus-resize.sh --prefix C-b
#   ./M01-mouse-focus-resize.sh --show-config    print the embedded config
#
# To change it: ./M01-mouse-focus-resize.sh --show-config > M01-mouse-focus-resize.yaml, edit, then
#   dozer -c M01-mouse-focus-resize.yaml --package M01-mouse-focus-resize.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# M01: Mouse: click to focus, drag borders  [M4]
#
# Steps:
#   1. Click inside each pane.
#   2. Click a pane's title line.
#   3. Drag the vertical divider left and right.
#   4. Drag the title line of pane 3 up and down.
#   5. Ctrl-a = to reset.
#
# Expect:
#   - A click focuses that pane (cyan title) and does nothing else.
#   - Dragging the divider or a title line resizes live, never below the minimum pane size.
#   - The sizes survive a window resize; Ctrl-a = restores the launch layout.
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: M01-mouse-focus-resize
description: |
  M01: Mouse: click to focus, drag borders  [M4]

  Steps:
    1. Click inside each pane.
    2. Click a pane's title line.
    3. Drag the vertical divider left and right.
    4. Drag the title line of pane 3 up and down.
    5. Ctrl-a = to reset.

  Expect:
    - A click focuses that pane (cyan title) and does nothing else.
    - Dragging the divider or a title line resizes live, never below the minimum pane size.
    - The sizes survive a window resize; Ctrl-a = restores the launch layout.

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
layout: "2,1"
panes:
  - {title: guide, run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  - {title: shell, run: 'echo ''[2]'''}
  - {title: drag my title line, run: 'echo ''[3] drag my title line'''}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "M01-mouse-focus-resize: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "M01-mouse-focus-resize: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
