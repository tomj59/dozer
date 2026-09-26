#!/bin/sh
# L01-default-layout: a dozer workspace, packaged by dozer 849a950-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./L01-default-layout.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./L01-default-layout.sh --prefix C-b
#   ./L01-default-layout.sh --show-config    print the embedded config
#
# To change it: ./L01-default-layout.sh --show-config > L01-default-layout.yaml, edit, then
#   dozer -c L01-default-layout.yaml --package L01-default-layout.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# L01: Default 2,1 layout  [M1]
#
# Steps:
#   1. Launch.
#   2. Look at the arrangement, dividers and titles.
#   3. Resize the window a few times.
#
# Expect:
#   - Two panes on top, one full-width pane below, split 50/50 vertically.
#   - Thin dividers; the ┬ joins the title line; pane [1] title is cyan.
#   - Panes re-proportion on resize; prompts redraw.
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: L01-default-layout
description: |
  L01: Default 2,1 layout  [M1]

  Steps:
    1. Launch.
    2. Look at the arrangement, dividers and titles.
    3. Resize the window a few times.

  Expect:
    - Two panes on top, one full-width pane below, split 50/50 vertically.
    - Thin dividers; the ┬ joins the title line; pane [1] title is cyan.
    - Panes re-proportion on resize; prompts redraw.

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
layout: "2,1"
panes:
  - {title: guide, run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  - {title: top-right, run: 'echo ''[2] top-right'''}
  - {title: bottom, run: 'echo ''[3] bottom'''}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "L01-default-layout: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "L01-default-layout: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
