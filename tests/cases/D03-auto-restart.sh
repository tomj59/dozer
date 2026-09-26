#!/bin/sh
# D03-auto-restart: a dozer workspace, packaged by dozer 792ade6-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./D03-auto-restart.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./D03-auto-restart.sh --prefix C-b
#   ./D03-auto-restart.sh --show-config    print the embedded config
#
# To change it: ./D03-auto-restart.sh --show-config > D03-auto-restart.yaml, edit, then
#   dozer -c D03-auto-restart.yaml --package D03-auto-restart.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# D03: Automatic restart: back-off and the retry limit  [M2]
#
# Steps:
#   1. Watch pane 2 for ~35 seconds.
#   2. Watch pane 3 (max_restarts: unlimited) for the same time.
#   3. When pane 2 has given up, Ctrl-a 2, Ctrl-a r.
#
# Expect:
#   - Pane 2 fails and shows '↻ restarting automatically (1/5)…'; the gaps grow 1s, 2s, 4s, 8s, 16s.
#   - After 5 restarts it stops: '✖ exit 1 · gave up after 5 automatic restarts · C-a r to try again'.
#   - Pane 3 keeps restarting (#1, #2, …) with the same growing gaps (capped at 30s).
#   - C-a r on pane 2 starts a fresh streak of 5.
#   - A run that stays up for a minute resets its count (not shown here).
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: D03-auto-restart
description: |
  D03: Automatic restart: back-off and the retry limit  [M2]

  Steps:
    1. Watch pane 2 for ~35 seconds.
    2. Watch pane 3 (max_restarts: unlimited) for the same time.
    3. When pane 2 has given up, Ctrl-a 2, Ctrl-a r.

  Expect:
    - Pane 2 fails and shows '↻ restarting automatically (1/5)…'; the gaps grow 1s, 2s, 4s, 8s, 16s.
    - After 5 restarts it stops: '✖ exit 1 · gave up after 5 automatic restarts · C-a r to try again'.
    - Pane 3 keeps restarting (#1, #2, …) with the same growing gaps (capped at 30s).
    - C-a r on pane 2 starts a fresh streak of 5.
    - A run that stays up for a minute resets its count (not shown here).

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
layout: "2,1"
panes:
  - {title: guide, run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  - {title: 'on-failure, default max 5', exec: 'echo "started $(date +%H:%M:%S)"; sleep 1; exit 1', restart: on-failure}
  - {title: 'always, unlimited', exec: 'echo "started $(date +%H:%M:%S)"; sleep 1; exit 1', restart: always, max_restarts: unlimited}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "D03-auto-restart: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "D03-auto-restart: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
