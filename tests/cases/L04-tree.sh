#!/bin/sh
# L04-tree: a dozer workspace, packaged by dozer dc640fc-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./L04-tree.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./L04-tree.sh --prefix C-b
#   ./L04-tree.sh --show-config    print the embedded config
#
# To change it: ./L04-tree.sh --show-config > L04-tree.yaml, edit, then
#   dozer -c L04-tree.yaml --package L04-tree.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# L04: Tree layout: tall left, stacked right  [M1]
#
# Steps:
#   1. Launch.
#   2. Look at where the divider meets the stacked titles.
#
# Expect:
#   - Left pane 60% wide and full height.
#   - Right column: two panes; the lower one is 10 rows tall.
#   - The divider shows ├ where the lower-right title line meets it.
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: L04-tree
description: |
  L04: Tree layout: tall left, stacked right  [M1]

  Steps:
    1. Launch.
    2. Look at where the divider meets the stacked titles.

  Expect:
    - Left pane 60% wide and full height.
    - Right column: two panes; the lower one is 10 rows tall.
    - The divider shows ├ where the lower-right title line meets it.

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
layout:
  cols:
    - {pane: guide, size: 60%}
    - rows:
        - {pane: upper}
        - {pane: lower, size: 10c}
panes:
  guide: {run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  upper: {run: echo upper}
  lower: {run: echo 'lower (10 rows)'}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "L04-tree: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "L04-tree: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
