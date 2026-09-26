#!/bin/sh
# L02-two-two-one: a dozer workspace, packaged by dozer 849a950-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./L02-two-two-one.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./L02-two-two-one.sh --prefix C-b
#   ./L02-two-two-one.sh --show-config    print the embedded config
#
# To change it: ./L02-two-two-one.sh --show-config > L02-two-two-one.yaml, edit, then
#   dozer -c L02-two-two-one.yaml --package L02-two-two-one.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# L02: Alternative 2,2,1 layout  [M1]
#
# Steps:
#   1. Launch.
#   2. Resize the window taller and shorter.
#
# Expect:
#   - Three rows: 2, 2 and 1 panes; the three rows share the height equally.
#   - Row heights adjust with the window.
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: L02-two-two-one
description: |
  L02: Alternative 2,2,1 layout  [M1]

  Steps:
    1. Launch.
    2. Resize the window taller and shorter.

  Expect:
    - Three rows: 2, 2 and 1 panes; the three rows share the height equally.
    - Row heights adjust with the window.

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
layout: "2,2,1"
panes:
  - {title: guide, run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  - {title: shell, run: 'echo ''[2]'''}
  - {title: shell, run: 'echo ''[3]'''}
  - {title: shell, run: 'echo ''[4]'''}
  - {title: bottom, run: 'echo ''[5] bottom'''}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "L02-two-two-one: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "L02-two-two-one: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
