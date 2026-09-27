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
5. `dozer_reflow.go`, `utf8.go`: soft wraps are remembered as a marker on the wrapped line's last cell (`Link.Params` suffix, stripped by `StripWrap` before drawing). Line operations carry it along for free.
6. `dozer_reflow.go`, `emulator.go` `Emulator.Resize`: the main screen and its scrollback are rewrapped at the new size (reflow, dozer DP-7), and rows that no longer fit go to scrollback instead of being cut off. Shrinking and then growing restores the text. The alternate screen is only resized.
7. `utf8.go` `handleGrapheme`: a wide glyph that doesn't fit in the last column wraps first, and a glyph ending exactly at the edge leaves the cursor pending-wrap. Before, the glyph was cut in half and the next one overwrote it.
8. `emulator.go` `NewEmulator`: the alternate screen keeps no scrollback. `Scrollback.Pushed` counts every line ever pushed, so a viewer can hold its place while old lines are evicted.
