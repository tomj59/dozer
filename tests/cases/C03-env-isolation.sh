#!/bin/sh
# C03-env-isolation: a dozer workspace, packaged by dozer dc640fc-dirty on 2026-09-26.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./C03-env-isolation.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./C03-env-isolation.sh --prefix C-b
#   ./C03-env-isolation.sh --show-config    print the embedded config
#
# To change it: ./C03-env-isolation.sh --show-config > C03-env-isolation.yaml, edit, then
#   dozer -c C03-env-isolation.yaml --package C03-env-isolation.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
#
# C03: Panes don't inherit the host terminal's identity  [M2]
#
# Steps:
#   1. Launch from Terminal.app or iTerm2 (not inside tmux).
#   2. Read pane 2.
#   3. Type 'exit' in pane 3 and restart it (Ctrl-a r).
#
# Expect:
#   - No pane prints 'Restored session: …' (Terminal.app session restore).
#   - Pane 2 shows TERM=xterm-256color, TERM_PROGRAM=dozer, DOZER_PANE=2, and no TERM_SESSION_ID / ITERM_SESSION_ID / TMUX.
#   - Pane 3's shell history is its own (no cross-pane session save/restore).
#
# Record: PASS / FAIL + notes (terminal app, screenshot if visual).

yaml() {
cat <<'DOZER_YAML'
version: 1
name: C03-env-isolation
description: |
  C03: Panes don't inherit the host terminal's identity  [M2]

  Steps:
    1. Launch from Terminal.app or iTerm2 (not inside tmux).
    2. Read pane 2.
    3. Type 'exit' in pane 3 and restart it (Ctrl-a r).

  Expect:
    - No pane prints 'Restored session: …' (Terminal.app session restore).
    - Pane 2 shows TERM=xterm-256color, TERM_PROGRAM=dozer, DOZER_PANE=2, and no TERM_SESSION_ID / ITERM_SESSION_ID / TMUX.
    - Pane 3's shell history is its own (no cross-pane session save/restore).

  Record: PASS / FAIL + notes (terminal app, screenshot if visual).
layout: "3"
panes:
  - {title: guide, run: printf '%s\n' "$DOZER_DESCRIPTION" | less -FXR}
  - {title: env, run: env | grep -E '^(TERM|TERM_PROGRAM|TERM_SESSION_ID|ITERM_SESSION_ID|TMUX|DOZER_PANE)=' | sort}
  - {title: shell, run: 'echo ''[3] shell'''}
DOZER_YAML
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "C03-env-isolation: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "C03-env-isolation: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
