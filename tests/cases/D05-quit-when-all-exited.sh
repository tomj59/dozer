#!/bin/sh
# D05-quit-when-all-exited: a dozer workspace, packaged by dozer dc640fc-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./D05-quit-when-all-exited.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./D05-quit-when-all-exited.sh --prefix C-b
#   ./D05-quit-when-all-exited.sh --show-config    print the embedded config
#
# To change it: ./D05-quit-when-all-exited.sh --show-config > D05-quit-when-all-exited.yaml, edit, then
#   dozer -c D05-quit-when-all-exited.yaml --package D05-quit-when-all-exited.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# D05: quit_when_all_exited: true  [M1]
#
# Steps:
#   1. Launch and wait ~4 seconds without touching anything.
#
# Expect:
#   - Both panes finish; dozer then quits by itself and your terminal is back to normal.
#   - (The default is to stay open with dead panes; this config opts out.)
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: D05-quit-when-all-exited
description: |
  D05: quit_when_all_exited: true  [M1]

  Steps:
    1. Launch and wait ~4 seconds without touching anything.

  Expect:
    - Both panes finish; dozer then quits by itself and your terminal is back to normal.
    - (The default is to stay open with dead panes; this config opts out.)

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
quit_when_all_exited: true
layout: "2"
panes:
  - {title: 3s, exec: printf '%s\n\n' "$DOZER_DESCRIPTION"; sleep 3}
  - {title: 2s, exec: echo 'done in 2s'; sleep 2}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "D05-quit-when-all-exited: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "D05-quit-when-all-exited: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
