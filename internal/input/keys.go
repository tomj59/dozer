package input

import "unicode/utf8"

// Key is one decoded keypress, for dozer's own modes (copy mode, prompts).
// Normal typing never goes through here: it passes to panes as raw bytes.
type Key struct {
	Rune rune   // printable character or control code (0x01 = Ctrl-a …)
	Name string // special keys: up down left right pgup pgdn home end esc enter backspace tab
}

// Keys decodes raw input into keys. Unknown escape sequences are dropped.
func Keys(b []byte) []Key {
	var out []Key
	for len(b) > 0 {
		c := b[0]
		switch {
		case c == 0x1b:
			k, n := escape(b)
			if k.Name != "" || k.Rune != 0 {
				out = append(out, k)
			}
			b = b[n:]
			continue
		case c == '\r' || c == '\n':
			out = append(out, Key{Name: "enter"})
		case c == 0x7f || c == 0x08:
			out = append(out, Key{Name: "backspace"})
		case c == '\t':
			out = append(out, Key{Name: "tab"})
		case c < 0x20:
			out = append(out, Key{Rune: rune(c)})
		default:
			r, n := utf8.DecodeRune(b)
			out = append(out, Key{Rune: r})
			b = b[n:]
			continue
		}
		b = b[1:]
	}
	return out
}

// escape decodes an escape sequence at the start of b.
func escape(b []byte) (Key, int) {
	if len(b) == 1 {
		return Key{Name: "esc"}, 1
	}
	switch b[1] {
	case '[', 'O':
	default:
		if b[1] == 0x1b {
			return Key{Name: "esc"}, 1
		}
		// Alt+key: report the key itself.
		r, n := utf8.DecodeRune(b[1:])
		return Key{Rune: r}, 1 + n
	}
	// CSI / SS3: parameters then a final byte.
	i := 2
	for i < len(b) && (b[i] < 0x40 || b[i] > 0x7e) {
		i++
	}
	if i >= len(b) {
		return Key{}, len(b)
	}
	params, final := string(b[2:i]), b[i]
	n := i + 1
	switch final {
	case 'A':
		return Key{Name: "up"}, n
	case 'B':
		return Key{Name: "down"}, n
	case 'C':
		return Key{Name: "right"}, n
	case 'D':
		return Key{Name: "left"}, n
	case 'H':
		return Key{Name: "home"}, n
	case 'F':
		return Key{Name: "end"}, n
	case '~':
		switch params {
		case "1", "7":
			return Key{Name: "home"}, n
		case "4", "8":
			return Key{Name: "end"}, n
		case "5":
			return Key{Name: "pgup"}, n
		case "6":
			return Key{Name: "pgdn"}, n
		}
	}
	return Key{}, n
}
