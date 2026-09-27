#!/bin/sh
# C02-cwd-env: a dozer workspace, packaged by dozer dc640fc-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./C02-cwd-env.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./C02-cwd-env.sh --prefix C-b
#   ./C02-cwd-env.sh --show-config    print the embedded config
#
# To change it: ./C02-cwd-env.sh --show-config > C02-cwd-env.yaml, edit, then
#   dozer -c C02-cwd-env.yaml --package C02-cwd-env.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# C02: Per-pane cwd and env  [M1]
#
# Steps:
#   1. Read each pane's output.
#   2. Run this script from two different folders.
#
# Expect:
#   - Pane 2 starts in your home folder (cwd: ~).
#   - Pane 3 shows GREETING=hello-from-dozer and DOZER_PANE=3.
#   - Pane 4 starts in /tmp (absolute cwd), wherever you run the script from.
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: C02-cwd-env
description: |
  C02: Per-pane cwd and env  [M1]

  Steps:
    1. Read each pane's output.
    2. Run this script from two different folders.

  Expect:
    - Pane 2 starts in your home folder (cwd: ~).
    - Pane 3 shows GREETING=hello-from-dozer and DOZER_PANE=3.
    - Pane 4 starts in /tmp (absolute cwd), wherever you run the script from.

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
layout: "2,2"
panes:
  - {title: guide, run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  - {title: cwd ~, run: pwd, cwd: "~"}
  - {title: env, run: echo GREETING=$GREETING DOZER_PANE=$DOZER_PANE, env: {GREETING: hello-from-dozer}}
  - {title: cwd /tmp, run: pwd, cwd: /tmp}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "C02-cwd-env: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "C02-cwd-env: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
