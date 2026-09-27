#!/bin/sh
# P01-passthrough: a dozer workspace, packaged by dozer dc640fc-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./P01-passthrough.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./P01-passthrough.sh --prefix C-b
#   ./P01-passthrough.sh --show-config    print the embedded config
#
# To change it: ./P01-passthrough.sh --show-config > P01-passthrough.yaml, edit, then
#   dozer -c P01-passthrough.yaml --package P01-passthrough.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# P01: Package: flags pass through  [M2]
#
# Steps:
#   1. Quit, then run this script again as: ./P01-passthrough.sh --prefix C-b --heights 30,70
#   2. Then: ./P01-passthrough.sh --show-config
#
# Expect:
#   - With the flags, the prefix is Ctrl-b and the bottom row is taller.
#   - --show-config prints this case's YAML without launching.
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: P01-passthrough
description: |
  P01: Package: flags pass through  [M2]

  Steps:
    1. Quit, then run this script again as: ./P01-passthrough.sh --prefix C-b --heights 30,70
    2. Then: ./P01-passthrough.sh --show-config

  Expect:
    - With the flags, the prefix is Ctrl-b and the bottom row is taller.
    - --show-config prints this case's YAML without launching.

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
layout: "2,1"
panes:
  - {title: guide, run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  - {title: shell, run: 'echo ''[2]'''}
  - {title: shell, run: 'echo ''[3]'''}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "P01-passthrough: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "P01-passthrough: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
