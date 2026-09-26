#!/bin/sh
# D02-restore: a dozer workspace, packaged by dozer 849a950-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./D02-restore.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./D02-restore.sh --prefix C-b
#   ./D02-restore.sh --show-config    print the embedded config
#
# To change it: ./D02-restore.sh --show-config > D02-restore.yaml, edit, then
#   dozer -c D02-restore.yaml --package D02-restore.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# D02: Restoring dead panes (r / R)  [M1]
#
# Steps:
#   1. Wait until all three right-hand panes are dead.
#   2. Ctrl-a 2, Ctrl-a r.
#   3. Ctrl-a R.
#
# Expect:
#   - Only pane 2 restarts (flash 'restarted [2]'), in the same place and size.
#   - R restarts every remaining dead pane.
#   - Each restarted pane starts on a fresh screen.
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: D02-restore
description: |
  D02: Restoring dead panes (r / R)  [M1]

  Steps:
    1. Wait until all three right-hand panes are dead.
    2. Ctrl-a 2, Ctrl-a r.
    3. Ctrl-a R.

  Expect:
    - Only pane 2 restarts (flash 'restarted [2]'), in the same place and size.
    - R restarts every remaining dead pane.
    - Each restarted pane starts on a fresh screen.

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
layout: "2,1"
heights: [60%, 40%]
panes:
  - {title: guide, run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  - {title: dies in 2s, exec: date; sleep 2; exit 1}
  - {title: dies in 3s, exec: date; sleep 3; exit 2}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "D02-restore: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "D02-restore: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
