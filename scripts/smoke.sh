#!/usr/bin/env bash
# Headless end-to-end checks: runs bin/dozer inside detached tmux sessions
# and inspects the rendered screen. Linux/macOS; needs tmux and vim.
set -uo pipefail
cd "$(dirname "$0")/.."
make build >/dev/null || exit 1
EMU=${EMU:-charm}
export LANG=${LANG:-C.UTF-8}
BASE=dozer-smoke-$$
PKGDIR=$(mktemp -d)
S=$BASE
fail=0
cleanup() { rm -rf "$PKGDIR"; for s in $BASE $BASE-1 $BASE-q $BASE-p $BASE-h $BASE-m; do tmux kill-session -t "$s" 2>/dev/null; done; }
trap cleanup EXIT

start() { # session, width, height, dozer args...
  local s=$1 w=$2 h=$3; shift 3
  tmux new-session -d -s "$s" -x "$w" -y "$h" "env SHELL=/bin/bash PS1='$ ' bin/dozer -emu $EMU $*"
  tmux set -g set-clipboard on # accept OSC 52 copies into tmux buffers
  sleep 1.5
}
send() { tmux send-keys -t "$S" "$@"; sleep 0.5; }
screen() { tmux capture-pane -t "$S" -p; }
ok() { echo "ok   $1"; }
bad() { echo "FAIL $1"; fail=1; }
check() { # name, grep pattern
  if screen | grep -q -- "$2"; then ok "$1"; else bad "$1"; screen; fi
}
cursor_is() { # name, expected column
  local got; got=$(tmux display -p -t "$S" '#{cursor_x}')
  if [ "$got" = "$2" ]; then ok "$1"; else bad "$1: cursor_x=$got want $2"; fi
}
quit() { send C-a q; send y; sleep 0.3; }

# --- default layout, 3 panes --------------------------------------------
start $S 110 32
check "three pane titles" "─ \[3\]"
check "status bar" "│ \[1\] bash"

send 'printf "\e[31mred\e[0m wide:日本語 end\n"' Enter
check "output + wide chars" "red wide:日本語 end"

send 'vim -u NONE -N README.md' Enter
sleep 0.5
check "vim alt screen" "# dozer"
send j j Down x Escape ':q!' Enter
check "back from alt screen" "red wide"

send C-u 'echo abcdef'
end=$(tmux display -p -t "$S" '#{cursor_x}')
send Left
cursor_is "cursor follows 1st Left" $((end - 1))
send Left
cursor_is "cursor follows 2nd Left" $((end - 2))
send BSpace X
check "edit lands at cursor" "echo abcXef"
cursor_is "cursor after edit" $((end - 2))
send C-u

send C-a
check "prefix pending shown" "PREFIX"
send Escape
check "Esc cancels prefix" "│ \[1\] bash"
send C-a C-a
send 'echo literal-ok' Enter
check "C-a C-a literal" "literal-ok"

send C-a Right
send 'echo marker-two' Enter
if screen | grep -q "│.*marker-two"; then ok "C-a → focuses right pane"; else bad "C-a → focuses right pane"; screen; fi
send C-a Down
send 'echo marker-three' Enter
if screen | grep -q "^marker-three"; then ok "C-a ↓ focuses bottom pane"; else bad "C-a ↓ focuses bottom pane"; screen; fi
send C-a 1
check "C-a 1 focuses pane 1" "│ \[1\] bash"

send C-a z
check "zoom" "(zoomed)"
send C-a z
check "unzoom" "─ \[3\]"

send C-a C-l
check "C-a C-l redraw keeps screen" "marker-two"

# Runtime resize: C-a L then L again (repeat window) moves the border right.
border() { screen | head -1 | python3 -c 'import sys; print(sys.stdin.readline().index("┬"))'; }
before=$(border)
send C-a L
check "resize mode shown" "RESIZE"
send L
after=$(border)
if [ "$after" -eq $((before + 4)) ]; then ok "C-a L L grows pane 1 by 4"; else bad "C-a L L: border $before → $after, want +4"; fi
send C-a =
after=$(border)
if [ "$after" -eq "$before" ]; then ok "C-a = resets sizes"; else bad "C-a =: border $after, want $before"; fi

send C-a t
check "title prompt" "Title for \[1\]"
send C-u 'my-title' Enter
check "pane renamed" "─ \[1\] my-title"

send C-a s
if screen | tail -1 | grep -q "my-title"; then bad "C-a s hides status bar"; else ok "C-a s hides status bar"; fi
send C-a s
check "C-a s shows it again" "│ \[1\] my-title"

send 'time seq 1 200000' Enter
sleep 3
check "flood finishes" "^real"

send C-a q
check "quit asks when panes run" "Quit dozer?"
send n
check "declining keeps dozer" "│ \[1\] "
quit
if tmux has-session -t "$S" 2>/dev/null; then bad "C-a q y quits"; else ok "C-a q y quits"; fi

# --- one pane: sizes reach the child -------------------------------------
S=$BASE-1
start $S 70 20 -l 1
send 'stty size' Enter
check "pty size = window - title - status" "^18 70"
tmux resize-window -t "$S" -x 60 -y 15
sleep 0.6
send 'stty size' Enter
check "resize reaches child" "^13 60"
quit

# --- dead panes ----------------------------------------------------------
S=$BASE-q
start $S 90 24 -l 2 -x "'echo bye; exit 3'"
sleep 0.5
check "failed pane flagged" "✖ exit 3"
check "dead count in status bar" "1 dead"
send C-a 2
send C-a r
check "restart flash" "restarted \[2\]"
send C-a x
check "kill asks" "Kill \[2\]"
send y
sleep 0.5
check "killed pane flagged" "✖ killed"
send C-a r
sleep 0.5
send 'exit' Enter
sleep 0.5
if tmux has-session -t "$S" 2>/dev/null; then ok "dozer stays when panes exit"; else bad "dozer stays when panes exit"; fi
check "exited pane flagged" "exited"
quit

# --- history: reflow, copy mode, clipboard (M4) -----------------------
S=$BASE-h
start $S 60 12 -l 1
send 'printf "%s\n" "alpha-0123456789-beta-0123456789-gamma-0123456789-delta"' Enter
tmux resize-window -t "$S" -x 30 -y 12; sleep 0.6
tmux resize-window -t "$S" -x 60 -y 12; sleep 0.6
check "reflow: shrink then grow restores the line" "^alpha-0123456789-beta-0123456789-gamma-0123456789-delta"
send 'seq 1 40' Enter
send C-a '['
check "copy mode: status and position tag" "COPY \[1\]"
send k k 0 v '$' y
if tmux show-buffer 2>/dev/null | grep -qx 39; then ok "copy mode: y copies via OSC 52"; else bad "copy mode: y copies via OSC 52"; tmux show-buffer | head -3; fi
send C-a '[' '?' 1 7 Enter
check "copy mode: search moves the view" "^17 "
send q
if screen | grep -q "COPY"; then bad "copy mode: q exits"; else ok "copy mode: q exits"; fi
quit

# --- mouse (M4): events typed as raw SGR reports ------------------------
S=$BASE-m
start $S 80 16 -l 2
mouse() { tmux send-keys -t "$S" -l "$(printf '\033[<%s' "$1")"; sleep 0.3; }
send 'seq 1 50' Enter
mouse '0;60;5M'; mouse '0;60;5m'
check "mouse: click focuses the pane" "│ \[2\] bash"
before=$(border)
mouse '0;41;8M'; mouse '32;37;8M'; mouse '0;37;8m'
after=$(border)
if [ "$after" -eq $((before - 4)) ]; then ok "mouse: dragging a divider resizes"; else bad "mouse: divider drag $before → $after, want -4"; fi
mouse '64;10;8M'
check "mouse: wheel scrolls history" "\[3/"
mouse '65;10;8M'
if screen | grep -q "\[0/\|\[3/"; then bad "mouse: wheel back down returns to live"; else ok "mouse: wheel back down returns to live"; fi
# A wheel flood (a trackpad flick) must not lock up input (H01 note).
flood=$(for i in $(seq 1 150); do printf '\033[<64;10;8M'; done)
t0=$(date +%s)
for k in $(seq 1 20); do tmux send-keys -t "$S" -l "$flood"; done
tmux send-keys -t "$S" C-a 2; sleep 0.5
for i in $(seq 1 20); do screen | grep -q "│ \[2\] bash" && break; sleep 0.25; done
if [ $(( $(date +%s) - t0 )) -le 5 ] && screen | grep -q "│ \[2\] bash"; then ok "mouse: 3000 wheel events don't block keys"; else bad "mouse: wheel flood blocked input ($(( $(date +%s) - t0 ))s)"; fi
send C-a 1
send q
mouse '0;1;3M'; mouse '32;2;4M'; mouse '0;2;4m'
sleep 0.3
if [ "$( (tmux show-buffer 2>/dev/null; echo) | grep -Ec '^[0-9]+$')" = 2 ]; then ok "mouse: drag selects and copies"; else bad "mouse: drag selects and copies"; tmux list-buffers; fi
quit

# --- packaged workspace: config piped on stdin, keyboard from the tty ----
printf 'layout: "2"\npanes: [{title: one, cwd: sub}, {title: two}]\n' | bin/dozer -c - --package "$PKGDIR/ws" >/dev/null 2>&1
mkdir -p "$PKGDIR/run/sub"
S=$BASE-p
tmux new-session -d -s "$S" -x 90 -y 20 "cd $PKGDIR/run && env DOZER_BIN=$PWD/bin/dozer SHELL=/bin/bash PS1='$ ' sh $PKGDIR/ws.sh"
sleep 1.5
check "package launches with its name" " ws │ \[1\] one"
send 'pwd' Enter
check "relative cwd resolves where it runs" "$PKGDIR/run/sub"
quit
if tmux has-session -t "$S" 2>/dev/null; then bad "package: keyboard works (C-a q y)"; else ok "package: keyboard works (C-a q y)"; fi

exit $fail
