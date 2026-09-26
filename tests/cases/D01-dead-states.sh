#!/bin/sh
# D01-dead-states: a dozer workspace, packaged by dozer 849a950-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./D01-dead-states.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./D01-dead-states.sh --prefix C-b
#   ./D01-dead-states.sh --show-config    print the embedded config
#
# To change it: ./D01-dead-states.sh --show-config > D01-dead-states.yaml, edit, then
#   dozer -c D01-dead-states.yaml --package D01-dead-states.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# D01: Dead-pane states and chrome  [M1]
#
# Steps:
#   1. Launch and wait ~6 seconds.
#   2. Judge: could any of these be overlooked?
#
# Expect:
#   - [2] dim title 'exited' (status 0).
#   - [3] red '✖ exit 3'.
#   - [4] red '✖ SIGKILL'.
#   - Each dead pane: dimmed last output + banner at its bottom with the time and 'C-a r restart'.
#   - Status bar: red '✖ 3 dead · C-a R'. Layout never changes.
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: D01-dead-states
description: |
  D01: Dead-pane states and chrome  [M1]

  Steps:
    1. Launch and wait ~6 seconds.
    2. Judge: could any of these be overlooked?

  Expect:
    - [2] dim title 'exited' (status 0).
    - [3] red '✖ exit 3'.
    - [4] red '✖ SIGKILL'.
    - Each dead pane: dimmed last output + banner at its bottom with the time and 'C-a r restart'.
    - Status bar: red '✖ 3 dead · C-a R'. Layout never changes.

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
layout: "2,2"
panes:
  - {title: guide, run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  - {title: clean exit, exec: echo 'exit 0 in 3s'; sleep 3; exit 0}
  - {title: failure, exec: echo 'exit 3 in 4s'; sleep 4; exit 3}
  - {title: killed, exec: echo 'SIGKILL in 5s'; sleep 5; kill -KILL $$}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "D01-dead-states: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "D01-dead-states: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
