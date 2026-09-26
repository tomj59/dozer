#!/bin/sh
# L03-sizes: a dozer workspace, packaged by dozer 849a950-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./L03-sizes.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./L03-sizes.sh --prefix C-b
#   ./L03-sizes.sh --show-config    print the embedded config
#
# To change it: ./L03-sizes.sh --show-config > L03-sizes.yaml, edit, then
#   dozer -c L03-sizes.yaml --package L03-sizes.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# L03: Launch sizes: percent, cells, weights  [M1]
#
# Steps:
#   1. Launch.
#   2. Compare the proportions to the expectation.
#
# Expect:
#   - Top row about 70% of the height, bottom row about 30%.
#   - Top row split 30% / 70% (guide narrow on the left).
#   - Bottom row: a fixed 24-column pane, then two panes sharing the rest 1:2.
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: L03-sizes
description: |
  L03: Launch sizes: percent, cells, weights  [M1]

  Steps:
    1. Launch.
    2. Compare the proportions to the expectation.

  Expect:
    - Top row about 70% of the height, bottom row about 30%.
    - Top row split 30% / 70% (guide narrow on the left).
    - Bottom row: a fixed 24-column pane, then two panes sharing the rest 1:2.

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
layout: "2,3"
heights: [70%, 30%]
widths: {1: [30%, 70%], 2: [24c, 1fr, 2fr]}
panes:
  - {title: guide, run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  - {title: 70% wide, run: 'echo ''[2] 70% wide'''}
  - {title: 24 cols, run: 'echo ''[3] 24 cols'''}
  - {title: 1fr, run: 'echo ''[4] 1fr'''}
  - {title: 2fr, run: 'echo ''[5] 2fr'''}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "L03-sizes: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "L03-sizes: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
