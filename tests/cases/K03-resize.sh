#!/bin/sh
# K03-resize: a dozer workspace, packaged by dozer 849a950-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./K03-resize.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./K03-resize.sh --prefix C-b
#   ./K03-resize.sh --show-config    print the embedded config
#
# To change it: ./K03-resize.sh --show-config > K03-resize.yaml, edit, then
#   dozer -c K03-resize.yaml --package K03-resize.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# K03: Runtime resize (H J K L) and reset (=)  [M2]
#
# Steps:
#   1. Ctrl-a L, then L L L quickly (no prefix).
#   2. Wait a second, type L.
#   3. Ctrl-a 3, Ctrl-a K K K.
#   4. Ctrl-a 2, Ctrl-a L.
#   5. Resize the window.
#   6. Ctrl-a =
#
# Expect:
#   - Pane 1's right border moves right each press; status says RESIZE.
#   - After the pause a plain 'L' is typed into the shell.
#   - Bottom pane grows upward.
#   - Pane 2 (right edge) shrinks: its left border moves right.
#   - Your proportions survive the window resize.
#   - = restores the launch sizes.
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: K03-resize
description: |
  K03: Runtime resize (H J K L) and reset (=)  [M2]

  Steps:
    1. Ctrl-a L, then L L L quickly (no prefix).
    2. Wait a second, type L.
    3. Ctrl-a 3, Ctrl-a K K K.
    4. Ctrl-a 2, Ctrl-a L.
    5. Resize the window.
    6. Ctrl-a =

  Expect:
    - Pane 1's right border moves right each press; status says RESIZE.
    - After the pause a plain 'L' is typed into the shell.
    - Bottom pane grows upward.
    - Pane 2 (right edge) shrinks: its left border moves right.
    - Your proportions survive the window resize.
    - = restores the launch sizes.

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
layout: "2,1"
panes:
  - {title: guide, run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  - {title: shell, run: 'echo ''[2]'''}
  - {title: shell, run: 'echo ''[3]'''}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "K03-resize: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "K03-resize: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
