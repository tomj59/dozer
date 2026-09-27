#!/bin/sh
# H03-search: a dozer workspace, packaged by dozer effde5c-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./H03-search.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./H03-search.sh --prefix C-b
#   ./H03-search.sh --show-config    print the embedded config
#
# To change it: ./H03-search.sh --show-config > H03-search.yaml, edit, then
#   dozer -c H03-search.yaml --package H03-search.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# H03: Search in copy mode (/ ? n N)  [M4]
#
# Steps:
#   1. Ctrl-a 2, Ctrl-a [, ? error Enter.
#   2. n several times, then N.
#   3. ? ERROR Enter.
#   4. / nomatch Enter.
#
# Expect:
#   - ? finds the nearest 'error' above the cursor; lower case also matches ERROR (smart case).
#   - n keeps going up, N goes back down; the search wraps at the ends.
#   - Upper case in the query makes it case-sensitive: only ERROR lines match.
#   - A miss says "nomatch" not found and leaves the cursor put.
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: H03-search
description: |
  H03: Search in copy mode (/ ? n N)  [M4]

  Steps:
    1. Ctrl-a 2, Ctrl-a [, ? error Enter.
    2. n several times, then N.
    3. ? ERROR Enter.
    4. / nomatch Enter.

  Expect:
    - ? finds the nearest 'error' above the cursor; lower case also matches ERROR (smart case).
    - n keeps going up, N goes back down; the search wraps at the ends.
    - Upper case in the query makes it case-sensitive: only ERROR lines match.
    - A miss says "nomatch" not found and leaves the cursor put.

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
layout: "2,1"
panes:
  - {title: guide, run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  - {title: log, run: 'for i in $(seq 1 300); do case $((i % 37)) in 0) echo "$i ERROR disk full";; 11) echo "$i error: timeout";; *) echo "$i ok";; esac; done'}
  - {title: shell, run: 'echo ''[3]'''}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "H03-search: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "H03-search: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
