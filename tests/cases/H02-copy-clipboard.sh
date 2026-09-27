#!/bin/sh
# H02-copy-clipboard: a dozer workspace, packaged by dozer d966d1e on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./H02-copy-clipboard.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./H02-copy-clipboard.sh --prefix C-b
#   ./H02-copy-clipboard.sh --show-config    print the embedded config
#
# To change it: ./H02-copy-clipboard.sh --show-config > H02-copy-clipboard.yaml, edit, then
#   dozer -c H02-copy-clipboard.yaml --package H02-copy-clipboard.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# H02: Copy to the clipboard (v, V, y)  [M4]
#
# Steps:
#   1. Ctrl-a 2, Ctrl-a [, move onto the long wrapped line, press V then y.
#   2. Ctrl-a 3 and paste (Cmd-V).
#   3. Ctrl-a 2, Ctrl-a [, put the cursor on 'alpha', v, then w w e (end of gamma), y. Paste in pane 3.
#
# Expect:
#   - Status bar says 'copied N characters'; copy mode ends.
#   - The long line pastes as ONE line (soft wraps joined), with no trailing spaces.
#   - The v selection pastes exactly 'alpha beta gamma' (inclusive of the cursor character).
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: H02-copy-clipboard
description: |
  H02: Copy to the clipboard (v, V, y)  [M4]

  Steps:
    1. Ctrl-a 2, Ctrl-a [, move onto the long wrapped line, press V then y.
    2. Ctrl-a 3 and paste (Cmd-V).
    3. Ctrl-a 2, Ctrl-a [, put the cursor on 'alpha', v, then w w e (end of gamma), y. Paste in pane 3.

  Expect:
    - Status bar says 'copied N characters'; copy mode ends.
    - The long line pastes as ONE line (soft wraps joined), with no trailing spaces.
    - The v selection pastes exactly 'alpha beta gamma' (inclusive of the cursor character).

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
layout: "2,1"
panes:
  - {title: guide, run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  - {title: text, run: 'printf ''%s\n'' ''This is one long logical line that is much wider than the pane, so the terminal wraps it onto several rows; copying it must give back ONE line, and resizing must rewrap it.''; echo ''alpha beta gamma delta''; echo ''last line'''}
  - {title: paste here, run: 'echo ''[3] paste here'''}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "H02-copy-clipboard: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "H02-copy-clipboard: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
