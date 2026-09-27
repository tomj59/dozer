package app

import (
	"fmt"

	"github.com/tomj59/dozer/internal/input"
	"github.com/tomj59/dozer/internal/pane"
)

// enterCopy starts copy mode (C-a [) on the focused pane.
func (a *App) enterCopy() {
	p := a.focused()
	if !p.EnterCopy() {
		a.say("this emulator has no scrollback")
	}
}

// copyInput handles keys typed while the focused pane is in copy mode
// (main loop). Keys follow vi, as in tmux's copy mode (docs/SPEC.md §4.4).
func (a *App) copyInput(b []byte) {
	p := a.focused()
	for len(b) > 0 {
		k, n := input.NextKey(b)
		b = b[n:]
		var searchPrompt string
		var copied string
		p.WithCopy(func(c *pane.CopyMode, src pane.Source) bool {
			switch k.Name {
			case "up":
				c.Move(src, 0, -1)
			case "down":
				c.Move(src, 0, 1)
			case "left":
				c.Move(src, -1, 0)
			case "right":
				c.Move(src, 1, 0)
			case "pgup":
				c.Page(src, -2)
			case "pgdn":
				c.Page(src, 2)
			case "home":
				c.Goto(src, src.First())
			case "end":
				c.Goto(src, src.Last())
			case "esc":
				if c.Sel != pane.SelNone {
					c.Sel = pane.SelNone
					return false
				}
				return true
			case "enter":
				copied = c.Text(src)
				return true
			}
			if k.Name != "" {
				return false
			}
			switch k.Rune {
			case 'q', 0x03: // q, Ctrl-c
				return true
			case 'k':
				c.Move(src, 0, -1)
			case 'j':
				c.Move(src, 0, 1)
			case 'h':
				c.Move(src, -1, 0)
			case 'l':
				c.Move(src, 1, 0)
			case 0x15: // Ctrl-u
				c.Page(src, -1)
			case 0x04: // Ctrl-d
				c.Page(src, 1)
			case 0x02: // Ctrl-b
				c.Page(src, -2)
			case 0x06: // Ctrl-f
				c.Page(src, 2)
			case 0x19: // Ctrl-y
				c.Scroll(src, -1)
			case 0x05: // Ctrl-e
				c.Scroll(src, 1)
			case 'g':
				c.Goto(src, src.First())
			case 'G':
				c.Goto(src, src.Last())
			case 'H':
				c.ScreenRow(src, 0)
			case 'M':
				c.ScreenRow(src, src.Rows()/2)
			case 'L':
				c.ScreenRow(src, -1)
			case '0':
				c.LineStart(src)
			case '$':
				c.LineEnd(src)
			case '^':
				c.FirstNonBlank(src)
			case 'w':
				c.WordNext(src)
			case 'b':
				c.WordPrev(src)
			case 'e':
				c.WordEnd(src)
			case 'v', ' ':
				c.Select(pane.SelChar)
			case 'V':
				c.Select(pane.SelLine)
			case 'y':
				copied = c.Text(src)
				return c.Sel != pane.SelNone
			case '/':
				searchPrompt = "Search down:"
			case '?':
				searchPrompt = "Search up:"
			case 'n', 'N':
				if !c.Search(src, "", false, k.Rune == 'N') {
					a.say(notFound(c.SearchTerm()))
				}
			}
			return false
		})
		if copied != "" {
			a.clipboard(copied)
		}
		if searchPrompt != "" {
			back := searchPrompt == "Search up:"
			a.promptHint = "Enter = search · Esc = cancel · lower case ignores case"
			a.prompt(searchPrompt, "", func(q string) {
				if q == "" {
					return
				}
				p.WithCopy(func(c *pane.CopyMode, src pane.Source) bool {
					if !c.Search(src, q, back, false) {
						a.say(notFound(q))
					}
					return false
				})
			})
			if len(b) > 0 { // typed ahead: the rest belongs to the prompt
				a.modalInput(b)
			}
			return
		}
	}
}

func notFound(q string) string { return fmt.Sprintf("%q not found", q) }

// clipboard puts text on the system clipboard through the host terminal
// (OSC 52), which also works over ssh.
func (a *App) clipboard(text string) {
	a.term.SetClipboard(text)
	n := len([]rune(text))
	a.say(fmt.Sprintf("copied %d characters", n))
}
