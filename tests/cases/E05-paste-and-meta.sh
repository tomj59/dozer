#!/bin/sh
# E05-paste-and-meta: a dozer workspace, packaged by dozer dc640fc-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./E05-paste-and-meta.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./E05-paste-and-meta.sh --prefix C-b
#   ./E05-paste-and-meta.sh --show-config    print the embedded config
#
# To change it: ./E05-paste-and-meta.sh --show-config > E05-paste-and-meta.yaml, edit, then
#   dozer -c E05-paste-and-meta.yaml --package E05-paste-and-meta.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# E05: Bracketed paste and Option/Meta keys  [M0]
#
# Steps:
#   1. Copy several lines of indented text.
#   2. In pane 2 (vim, insert mode) paste with Cmd-V.
#   3. In pane 1: type words, then Option-b / Option-f (needs 'Option as Meta').
#
# Expect:
#   - The paste arrives as-is: no staircase indentation.
#   - Option-b/f move by word, same as outside dozer.
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: E05-paste-and-meta
description: |
  E05: Bracketed paste and Option/Meta keys  [M0]

  Steps:
    1. Copy several lines of indented text.
    2. In pane 2 (vim, insert mode) paste with Cmd-V.
    3. In pane 1: type words, then Option-b / Option-f (needs 'Option as Meta').

  Expect:
    - The paste arrives as-is: no staircase indentation.
    - Option-b/f move by word, same as outside dozer.

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
layout: "2"
panes:
  - {title: guide, run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  - {title: vim, run: vim -u NONE -N -c 'set autoindent' -c 'startinsert'}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "E05-paste-and-meta: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "E05-paste-and-meta: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
