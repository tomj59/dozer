#!/bin/sh
# L06-small-window: a dozer workspace, packaged by dozer 849a950-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./L06-small-window.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./L06-small-window.sh --prefix C-b
#   ./L06-small-window.sh --show-config    print the embedded config
#
# To change it: ./L06-small-window.sh --show-config > L06-small-window.yaml, edit, then
#   dozer -c L06-small-window.yaml --package L06-small-window.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# L06: Window too small: minimum size + scrolling view (DP-1)  [M1]
#
# Steps:
#   1. Launch, then shrink the window narrower than about 100 columns.
#   2. Ctrl-a 3, then Ctrl-a 1.
#   3. Widen the window again.
#
# Expect:
#   - Panes stop shrinking at 32 columns; the status bar shows 'more ▶'.
#   - Focusing pane 3 scrolls the view right ('more ◀'); pane 1 scrolls back.
#   - Judgment call: is scrolling right, or would auto-zoom be better?
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: L06-small-window
description: |
  L06: Window too small: minimum size + scrolling view (DP-1)  [M1]

  Steps:
    1. Launch, then shrink the window narrower than about 100 columns.
    2. Ctrl-a 3, then Ctrl-a 1.
    3. Widen the window again.

  Expect:
    - Panes stop shrinking at 32 columns; the status bar shows 'more ▶'.
    - Focusing pane 3 scrolls the view right ('more ◀'); pane 1 scrolls back.
    - Judgment call: is scrolling right, or would auto-zoom be better?

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
min_pane: {w: 32, h: 5}
layout: "3"
panes:
  - {title: guide, run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  - {title: middle, run: 'echo ''[2] middle'''}
  - {title: right, run: 'echo ''[3] right'''}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "L06-small-window: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "L06-small-window: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
