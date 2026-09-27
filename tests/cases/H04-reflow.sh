#!/bin/sh
# H04-reflow: a dozer workspace, packaged by dozer effde5c-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./H04-reflow.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./H04-reflow.sh --prefix C-b
#   ./H04-reflow.sh --show-config    print the embedded config
#
# To change it: ./H04-reflow.sh --show-config > H04-reflow.yaml, edit, then
#   dozer -c H04-reflow.yaml --package H04-reflow.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# H04: Resize reflow (re-test of E01)  [M4]
#
# Steps:
#   1. Make the window narrow (about half width), then wide again.
#   2. Make it short, then tall again.
#   3. Ctrl-a 2, Ctrl-a [ and scroll up.
#
# Expect:
#   - Narrow: the long line and the colored text rewrap instead of being cut off.
#   - Wide again: everything is back exactly as it was (the E01 bug is gone).
#   - Short: rows that don't fit move into history (visible in copy mode), then come back when tall.
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: H04-reflow
description: |
  H04: Resize reflow (re-test of E01)  [M4]

  Steps:
    1. Make the window narrow (about half width), then wide again.
    2. Make it short, then tall again.
    3. Ctrl-a 2, Ctrl-a [ and scroll up.

  Expect:
    - Narrow: the long line and the colored text rewrap instead of being cut off.
    - Wide again: everything is back exactly as it was (the E01 bug is gone).
    - Short: rows that don't fit move into history (visible in copy mode), then come back when tall.

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
layout: "2,1"
panes:
  - {title: guide, run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  - {title: long lines, run: 'printf ''%s\n'' ''This is one long logical line that is much wider than the pane, so the terminal wraps it onto several rows; copying it must give back ONE line, and resizing must rewrap it.''; printf ''\033[1;31mred bold\033[0m then a wide word list: 日本語テキスト alpha beta gamma delta epsilon zeta eta theta\n'''}
  - {title: numbers, run: seq 1 40}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "H04-reflow: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "H04-reflow: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
