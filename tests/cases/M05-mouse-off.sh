#!/bin/sh
# M05-mouse-off: a dozer workspace, packaged by dozer effde5c-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./M05-mouse-off.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./M05-mouse-off.sh --prefix C-b
#   ./M05-mouse-off.sh --show-config    print the embedded config
#
# To change it: ./M05-mouse-off.sh --show-config > M05-mouse-off.yaml, edit, then
#   dozer -c M05-mouse-off.yaml --package M05-mouse-off.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# M05: mouse: false  [M4]
#
# Steps:
#   1. Click in pane 2; drag across both panes; use the wheel.
#   2. Quit, then run: dozer --no-mouse (same result).
#
# Expect:
#   - dozer ignores the mouse: clicks don't change focus, the terminal's own selection works (spanning panes, as before M4).
#   - Ctrl-a [ still gives copy mode from the keyboard.
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: M05-mouse-off
description: |
  M05: mouse: false  [M4]

  Steps:
    1. Click in pane 2; drag across both panes; use the wheel.
    2. Quit, then run: dozer --no-mouse (same result).

  Expect:
    - dozer ignores the mouse: clicks don't change focus, the terminal's own selection works (spanning panes, as before M4).
    - Ctrl-a [ still gives copy mode from the keyboard.

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
mouse: false
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
	echo "M05-mouse-off: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "M05-mouse-off: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
