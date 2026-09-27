#!/bin/sh
# E01-color-unicode: a dozer workspace, packaged by dozer dc640fc-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./E01-color-unicode.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./E01-color-unicode.sh --prefix C-b
#   ./E01-color-unicode.sh --show-config    print the embedded config
#
# To change it: ./E01-color-unicode.sh --show-config > E01-color-unicode.yaml, edit, then
#   dozer -c E01-color-unicode.yaml --package E01-color-unicode.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# E01: Colors and Unicode  [M0]
#
# Steps:
#   1. Look at pane 2 (color test) and pane 3 (Unicode).
#   2. Resize the window.
#
# Expect:
#   - 16 and 256 color blocks look right; the 24-bit gradient is smooth.
#   - Bold/italic/underline/curly underline render; CJK and emoji take two columns without shifting the text after them.
#   - The prompt after each test starts at the left edge.
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: E01-color-unicode
description: |
  E01: Colors and Unicode  [M0]

  Steps:
    1. Look at pane 2 (color test) and pane 3 (Unicode).
    2. Resize the window.

  Expect:
    - 16 and 256 color blocks look right; the 24-bit gradient is smooth.
    - Bold/italic/underline/curly underline render; CJK and emoji take two columns without shifting the text after them.
    - The prompt after each test starts at the left edge.

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
layout: "2,1"
panes:
  - {title: guide, run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  - {title: colors, run: 'for i in $(seq 0 15); do printf ''\033[48;5;%sm  '' $i; done; printf ''\033[0m\n''; for i in $(seq 16 231); do printf ''\033[48;5;%sm '' $i; done; printf ''\033[0m\n''; for i in $(seq 0 4 255); do printf ''\033[48;2;%s;80;%sm '' $i $((255-i)); done; printf ''\033[0m\n''; printf ''\033[1mbold\033[0m \033[3mitalic\033[0m \033[4munder\033[0m \033[4:3mcurly\033[0m \033[7mreverse\033[0m\n'''}
  - {title: unicode, run: "printf '[日本語] [한국어] [\U0001F44D\U0001F3FD] [e\\314\\201] [─│┌┐└┘] [→←↑↓]\\nwide|日本|next\\n'"}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "E01-color-unicode: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "E01-color-unicode: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
