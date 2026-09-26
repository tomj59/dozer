#!/bin/sh
# C01-run-vs-exec: a dozer workspace, packaged by dozer 849a950-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./C01-run-vs-exec.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./C01-run-vs-exec.sh --prefix C-b
#   ./C01-run-vs-exec.sh --show-config    print the embedded config
#
# To change it: ./C01-run-vs-exec.sh --show-config > C01-run-vs-exec.yaml, edit, then
#   dozer -c C01-run-vs-exec.yaml --package C01-run-vs-exec.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# C01: run: vs exec:  [M1]
#
# Steps:
#   1. Quit top in pane 2 (q).
#   2. Quit top in pane 3 (q).
#
# Expect:
#   - Pane 2 (run:) drops to a shell prompt after top exits.
#   - Pane 3 (exec:) becomes a dead pane ('exited') after top exits.
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: C01-run-vs-exec
description: |
  C01: run: vs exec:  [M1]

  Steps:
    1. Quit top in pane 2 (q).
    2. Quit top in pane 3 (q).

  Expect:
    - Pane 2 (run:) drops to a shell prompt after top exits.
    - Pane 3 (exec:) becomes a dead pane ('exited') after top exits.

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
layout: "3"
panes:
  - {title: guide, run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  - {title: 'run: top', run: top}
  - {title: 'exec: top', exec: top}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "C01-run-vs-exec: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "C01-run-vs-exec: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
