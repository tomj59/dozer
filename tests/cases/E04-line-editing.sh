#!/bin/sh
# E04-line-editing: a dozer workspace, packaged by dozer 849a950-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./E04-line-editing.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./E04-line-editing.sh --prefix C-b
#   ./E04-line-editing.sh --show-config    print the embedded config
#
# To change it: ./E04-line-editing.sh --show-config > E04-line-editing.yaml, edit, then
#   dozer -c E04-line-editing.yaml --package E04-line-editing.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# E04: Shell line editing and history  [M0]
#
# Steps:
#   1. In pane 1: type 'echo one two three', Enter.
#   2. ↑ to recall it; ← to the middle; delete and insert text.
#   3. Ctrl-a Ctrl-a (start of line), then Ctrl-e.
#   4. Repeat in pane 2.
#
# Expect:
#   - The cursor is always exactly where edits land (no off-by-one).
#   - Ctrl-a Ctrl-a sends one Ctrl-a to the shell.
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: E04-line-editing
description: |
  E04: Shell line editing and history  [M0]

  Steps:
    1. In pane 1: type 'echo one two three', Enter.
    2. ↑ to recall it; ← to the middle; delete and insert text.
    3. Ctrl-a Ctrl-a (start of line), then Ctrl-e.
    4. Repeat in pane 2.

  Expect:
    - The cursor is always exactly where edits land (no off-by-one).
    - Ctrl-a Ctrl-a sends one Ctrl-a to the shell.

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
layout: "2"
panes:
  - {title: guide, run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  - {title: edit here too, run: 'echo ''[2] edit here too'''}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "E04-line-editing: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "E04-line-editing: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
