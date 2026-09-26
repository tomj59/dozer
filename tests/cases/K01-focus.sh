#!/bin/sh
# K01-focus: a dozer workspace, packaged by dozer 849a950-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./K01-focus.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./K01-focus.sh --prefix C-b
#   ./K01-focus.sh --show-config    print the embedded config
#
# To change it: ./K01-focus.sh --show-config > K01-focus.yaml, edit, then
#   dozer -c K01-focus.yaml --package K01-focus.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# K01: Focus navigation  [M1]
#
# Steps:
#   1. Ctrl-a → ↓ ← ↑ and watch which title turns cyan.
#   2. Ctrl-a h j k l (same as arrows).
#   3. Ctrl-a 1…5, Ctrl-a o (next).
#   4. Type 'echo hi' in each pane you visit.
#
# Expect:
#   - Arrows move to the geometric neighbor; from [5] (bottom) ↑ goes to the row above.
#   - Typing always lands in the cyan-titled pane only.
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: K01-focus
description: |
  K01: Focus navigation  [M1]

  Steps:
    1. Ctrl-a → ↓ ← ↑ and watch which title turns cyan.
    2. Ctrl-a h j k l (same as arrows).
    3. Ctrl-a 1…5, Ctrl-a o (next).
    4. Type 'echo hi' in each pane you visit.

  Expect:
    - Arrows move to the geometric neighbor; from [5] (bottom) ↑ goes to the row above.
    - Typing always lands in the cyan-titled pane only.

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
layout: "2,2,1"
panes:
  - {title: guide, run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  - {title: top-right, run: 'echo ''[2] top-right'''}
  - {title: mid-left, run: 'echo ''[3] mid-left'''}
  - {title: mid-right, run: 'echo ''[4] mid-right'''}
  - {title: bottom, run: 'echo ''[5] bottom'''}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "K01-focus: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "K01-focus: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
