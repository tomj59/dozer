#!/bin/sh
# M03-mouse-select: a dozer workspace, packaged by dozer effde5c-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./M03-mouse-select.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./M03-mouse-select.sh --prefix C-b
#   ./M03-mouse-select.sh --show-config    print the embedded config
#
# To change it: ./M03-mouse-select.sh --show-config > M03-mouse-select.yaml, edit, then
#   dozer -c M03-mouse-select.yaml --package M03-mouse-select.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# M03: Mouse selection stays in its pane (re-test of E05)  [M4]
#
# Steps:
#   1. In pane 2, drag from the first line down across several lines.
#   2. Keep dragging below pane 2's bottom edge, then release.
#   3. Paste into pane 3 (Cmd-V).
#   4. Now hold your terminal's bypass (Terminal.app: toggle View → Allow Mouse Reporting, ⌘R; iTerm2: hold Option) and drag across both panes.
#
# Expect:
#   - The highlight never crosses into pane 3 or the divider; below the edge it scrolls pane 2.
#   - On release: 'copied N characters'; the paste is pane 2's text only, one line per row.
#   - With the bypass, the terminal's own selection works as before (dozer doesn't see those clicks).
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: M03-mouse-select
description: |
  M03: Mouse selection stays in its pane (re-test of E05)  [M4]

  Steps:
    1. In pane 2, drag from the first line down across several lines.
    2. Keep dragging below pane 2's bottom edge, then release.
    3. Paste into pane 3 (Cmd-V).
    4. Now hold your terminal's bypass (Terminal.app: toggle View → Allow Mouse Reporting, ⌘R; iTerm2: hold Option) and drag across both panes.

  Expect:
    - The highlight never crosses into pane 3 or the divider; below the edge it scrolls pane 2.
    - On release: 'copied N characters'; the paste is pane 2's text only, one line per row.
    - With the bypass, the terminal's own selection works as before (dozer doesn't see those clicks).

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
layout: "2,1"
panes:
  - {title: guide, run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  - {title: left text, run: for i in $(seq 1 60); do echo "left $i the quick brown fox"; done}
  - {title: 'right: paste here', run: for i in $(seq 1 5); do echo "RIGHT $i"; done}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "M03-mouse-select: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "M03-mouse-select: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
