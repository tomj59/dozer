#!/bin/sh
# E02-fullscreen-apps: a dozer workspace, packaged by dozer dc640fc-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./E02-fullscreen-apps.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./E02-fullscreen-apps.sh --prefix C-b
#   ./E02-fullscreen-apps.sh --show-config    print the embedded config
#
# To change it: ./E02-fullscreen-apps.sh --show-config > E02-fullscreen-apps.yaml, edit, then
#   dozer -c E02-fullscreen-apps.yaml --package E02-fullscreen-apps.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# E02: Full-screen programs side by side  [M0]
#
# Steps:
#   1. Pane 2 runs top; pane 3 runs less; pane 4 is a shell: run vim there.
#   2. Focus each and use it (q quits top/less; arrows in vim).
#   3. Resize the window while they run.
#
# Expect:
#   - Each program draws only inside its pane; arrows work (no stray A/B/C/D).
#   - They redraw correctly after the resize.
#   - Quitting each returns to a prompt with the screen restored.
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: E02-fullscreen-apps
description: |
  E02: Full-screen programs side by side  [M0]

  Steps:
    1. Pane 2 runs top; pane 3 runs less; pane 4 is a shell: run vim there.
    2. Focus each and use it (q quits top/less; arrows in vim).
    3. Resize the window while they run.

  Expect:
    - Each program draws only inside its pane; arrows work (no stray A/B/C/D).
    - They redraw correctly after the resize.
    - Quitting each returns to a prompt with the screen restored.

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
layout: "2,2"
panes:
  - {title: guide, run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  - {title: top, run: top}
  - {title: less, run: seq 1 500 | less -N}
  - {title: run vim here, run: 'echo ''[4] run vim here'''}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "E02-fullscreen-apps: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "E02-fullscreen-apps: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
