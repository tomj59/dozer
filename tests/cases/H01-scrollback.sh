#!/bin/sh
# H01-scrollback: a dozer workspace, packaged by dozer effde5c-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./H01-scrollback.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./H01-scrollback.sh --prefix C-b
#   ./H01-scrollback.sh --show-config    print the embedded config
#
# To change it: ./H01-scrollback.sh --show-config > H01-scrollback.yaml, edit, then
#   dozer -c H01-scrollback.yaml --package H01-scrollback.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# H01: Scrollback and copy mode (C-a [)  [M4]
#
# Steps:
#   1. Ctrl-a 2 (the seq pane), then Ctrl-a [.
#   2. k / j, Ctrl-u / Ctrl-d, Ctrl-b / Ctrl-f, g (top), G (bottom), PgUp / PgDn.
#   3. While scrolled back, look at pane 3's clock: new output keeps arriving there.
#   4. q to leave. Then Ctrl-a [ again and Esc to leave.
#
# Expect:
#   - The status bar says COPY [2] with lines-back/total; the pane shows a [n/N] tag top right.
#   - g reaches 1 (the first line); G returns to the prompt. The view doesn't jump while you read.
#   - q and Esc both return to the live pane, cursor back at the prompt.
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: H01-scrollback
description: |
  H01: Scrollback and copy mode (C-a [)  [M4]

  Steps:
    1. Ctrl-a 2 (the seq pane), then Ctrl-a [.
    2. k / j, Ctrl-u / Ctrl-d, Ctrl-b / Ctrl-f, g (top), G (bottom), PgUp / PgDn.
    3. While scrolled back, look at pane 3's clock: new output keeps arriving there.
    4. q to leave. Then Ctrl-a [ again and Esc to leave.

  Expect:
    - The status bar says COPY [2] with lines-back/total; the pane shows a [n/N] tag top right.
    - g reaches 1 (the first line); G returns to the prompt. The view doesn't jump while you read.
    - q and Esc both return to the live pane, cursor back at the prompt.

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
layout: "2,1"
panes:
  - {title: guide, run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  - {title: seq, run: seq 1 3000}
  - {title: clock, run: 'while :; do date +%H:%M:%S; sleep 1; done'}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "H01-scrollback: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "H01-scrollback: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
