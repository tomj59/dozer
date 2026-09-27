#!/bin/sh
# M02-mouse-wheel: a dozer workspace, packaged by dozer effde5c-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./M02-mouse-wheel.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./M02-mouse-wheel.sh --prefix C-b
#   ./M02-mouse-wheel.sh --show-config    print the embedded config
#
# To change it: ./M02-mouse-wheel.sh --show-config > M02-mouse-wheel.yaml, edit, then
#   dozer -c M02-mouse-wheel.yaml --package M02-mouse-wheel.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# M02: Mouse wheel  [M4]
#
# Steps:
#   1. Wheel up over pane 2 (a shell with history), then back down.
#   2. Wheel over pane 3 (less).
#   3. Wheel over pane 4 (vim with mouse on).
#
# Expect:
#   - Pane 2 enters its history ([n/N] tag); wheeling back to the bottom leaves it by itself. Focus doesn't change.
#   - less scrolls its file (dozer sends it arrow keys).
#   - vim scrolls its buffer (vim gets the wheel itself).
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: M02-mouse-wheel
description: |
  M02: Mouse wheel  [M4]

  Steps:
    1. Wheel up over pane 2 (a shell with history), then back down.
    2. Wheel over pane 3 (less).
    3. Wheel over pane 4 (vim with mouse on).

  Expect:
    - Pane 2 enters its history ([n/N] tag); wheeling back to the bottom leaves it by itself. Focus doesn't change.
    - less scrolls its file (dozer sends it arrow keys).
    - vim scrolls its buffer (vim gets the wheel itself).

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
layout: "2,2"
panes:
  - {title: guide, run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  - {title: shell + history, run: seq 1 500}
  - {title: less, run: seq 1 500 | less -N}
  - {title: vim mouse=a, run: seq 1 500 > /tmp/dozer-m02.txt; vim -u NONE -N -c 'set mouse=a' /tmp/dozer-m02.txt}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "M02-mouse-wheel: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "M02-mouse-wheel: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
