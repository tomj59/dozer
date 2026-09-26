# Local patches to charmbracelet/x/vt

Upstream: github.com/charmbracelet/x/vt at the version in UPSTREAM_VERSION (MIT, see LICENSE).
Wired in through a `replace` directive in dozer's go.mod. Goal: upstream these and drop the fork.

1. `screen.go` `Screen.DeleteLine`: full-width scroll regions rotate line slices
   (`rotateUp`) instead of copying every cell. Scrolling was O(width×height) per
   line feed, about 0.2 MB/s of output on a 120×40 pane.
2. `scrollback.go` `Scrollback.Push`: drop the oldest line by reslicing instead of
   `slices.Delete`, which shifted the whole scrollback on every push once it was full.
3. `scrollback.go` `Scrollback.Push`: trailing-blank trimming uses cheap field checks (`isBlankCell`) instead of `Cell.Equal`.
4. `emulator.go` `Emulator.Read`: drop the unsynchronized `closed` check, which raced with `Close` on another goroutine. The pipe already returns `io.EOF` after `Close`.
