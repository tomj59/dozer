#!/bin/sh
# D04-kill: a dozer workspace, packaged by dozer dc640fc-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./D04-kill.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./D04-kill.sh --prefix C-b
#   ./D04-kill.sh --show-config    print the embedded config
#
# To change it: ./D04-kill.sh --show-config > D04-kill.yaml, edit, then
#   dozer -c D04-kill.yaml --package D04-kill.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# D04: Kill a pane (x)  [M2]
#
# Steps:
#   1. Ctrl-a 2, Ctrl-a x, n.
#   2. Ctrl-a x, y.
#   3. Ctrl-a r.
#
# Expect:
#   - 'n' leaves it running.
#   - 'y' kills it: the title says '✖ killed (…)' with how it ended (zsh catches the hang-up and exits 1; others show SIGHUP or SIGKILL).
#   - No automatic restart, even though the pane has restart: always.
#   - r brings it back.
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: D04-kill
description: |
  D04: Kill a pane (x)  [M2]

  Steps:
    1. Ctrl-a 2, Ctrl-a x, n.
    2. Ctrl-a x, y.
    3. Ctrl-a r.

  Expect:
    - 'n' leaves it running.
    - 'y' kills it: the title says '✖ killed (…)' with how it ended (zsh catches the hang-up and exits 1; others show SIGHUP or SIGKILL).
    - No automatic restart, even though the pane has restart: always.
    - r brings it back.

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
layout: "2"
panes:
  - {title: guide, run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  - {title: victim, exec: while true; do date; sleep 1; done, restart: always}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "D04-kill: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "D04-kill: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
