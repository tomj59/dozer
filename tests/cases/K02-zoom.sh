#!/bin/sh
# K02-zoom: a dozer workspace, packaged by dozer 849a950-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./K02-zoom.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./K02-zoom.sh --prefix C-b
#   ./K02-zoom.sh --show-config    print the embedded config
#
# To change it: ./K02-zoom.sh --show-config > K02-zoom.yaml, edit, then
#   dozer -c K02-zoom.yaml --package K02-zoom.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# K02: Zoom  [M1]
#
# Steps:
#   1. Ctrl-a 2, then Ctrl-a z.
#   2. Type a command; Ctrl-a z again.
#   3. Zoom again, then Ctrl-a ← .
#
# Expect:
#   - Pane 2 fills the window; its title says (zoomed); status says ZOOM.
#   - Unzoom restores the layout with the output intact.
#   - Moving focus while zoomed leaves zoom.
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: K02-zoom
description: |
  K02: Zoom  [M1]

  Steps:
    1. Ctrl-a 2, then Ctrl-a z.
    2. Type a command; Ctrl-a z again.
    3. Zoom again, then Ctrl-a ← .

  Expect:
    - Pane 2 fills the window; its title says (zoomed); status says ZOOM.
    - Unzoom restores the layout with the output intact.
    - Moving focus while zoomed leaves zoom.

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
layout: "2,1"
panes:
  - {title: guide, run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  - {title: zoom me, run: ls -la}
  - {title: shell, run: 'echo ''[3]'''}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "K02-zoom: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "K02-zoom: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
