#!/bin/sh
# E03-flood: a dozer workspace, packaged by dozer 849a950-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./E03-flood.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./E03-flood.sh --prefix C-b
#   ./E03-flood.sh --show-config    print the embedded config
#
# To change it: ./E03-flood.sh --show-config > E03-flood.yaml, edit, then
#   dozer -c E03-flood.yaml --package E03-flood.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# E03: Heavy output in several panes  [M0]
#
# Steps:
#   1. Watch the three flood panes.
#   2. While they run, type in the guide pane and try Ctrl-a z.
#
# Expect:
#   - Each flood finishes (see 'real' timings) within a few seconds.
#   - The guide pane stays responsive during the flood.
#   - Compare: `time seq 1 1000000` outside dozer.
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: E03-flood
description: |
  E03: Heavy output in several panes  [M0]

  Steps:
    1. Watch the three flood panes.
    2. While they run, type in the guide pane and try Ctrl-a z.

  Expect:
    - Each flood finishes (see 'real' timings) within a few seconds.
    - The guide pane stays responsive during the flood.
    - Compare: `time seq 1 1000000` outside dozer.

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
layout: "2,2"
panes:
  - {title: guide, run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  - {title: flood 1, run: time seq 1 1000000}
  - {title: flood 2, run: time seq 1 1000000}
  - {title: flood 3, run: time seq 1 1000000}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "E03-flood: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "E03-flood: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
