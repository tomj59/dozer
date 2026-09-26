#!/usr/bin/env bash
# Headless end-to-end checks: runs bin/dozer inside a detached tmux session
# and inspects the rendered screen. Linux/macOS, needs tmux and vim.
set -euo pipefail
cd "$(dirname "$0")/.."
make build >/dev/null
EMU=${EMU:-charm}
S=dozer-smoke-$$
fail=0
trap 'tmux kill-session -t $S 2>/dev/null || true' EXIT

send() { tmux send-keys -t "$S" "$@"; sleep 0.6; }
screen() { tmux capture-pane -t "$S" -p; }
check() { # name, grep pattern
  if screen | grep -q -- "$2"; then echo "ok   $1"; else echo "FAIL $1"; screen; fail=1; fi
}

tmux new-session -d -s "$S" -x 100 -y 30 "env SHELL=/bin/bash PS1='$ ' bin/dozer -emu $EMU"
sleep 1.2
check "status bar" "dozer .* emu: $EMU"

send 'printf "\e[31mred\e[0m wide:日本語 end\n"' Enter
check "output + wide chars" "red wide:日本語 end"

send 'vim -u NONE -N README.md' Enter
sleep 0.5
check "vim alt screen" "^# dozer"
send j j Down x
send Escape ':q!' Enter
check "back from alt screen" "red wide"

send C-a
check "prefix pending shown" "PREFIX"
send C-a
send 'echo literal-ok' Enter
check "C-a C-a literal" "literal-ok"

tmux resize-window -t "$S" -x 70 -y 20
sleep 0.6
send 'stty size' Enter
check "resize reaches child" "^19 70"

send 'time seq 1 200000' Enter
sleep 3
check "flood finishes" "^real"

send C-a q
sleep 0.5
if tmux has-session -t "$S" 2>/dev/null; then echo "FAIL C-a q quits"; fail=1; else echo "ok   C-a q quits"; fi
exit $fail
