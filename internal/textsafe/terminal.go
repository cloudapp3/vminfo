package textsafe

import "strings"

const (
	escape = '\x1b'
	csi    = '\u009b'
	st     = '\u009c'
)

// Terminal removes terminal control sequences and control characters from s.
func Terminal(s string) string {
	var out strings.Builder
	out.Grow(len(s))
	runes := []rune(s)
	for i := 0; i < len(runes); {
		r := runes[i]
		switch {
		case r == escape:
			i = skipEscapeSequence(runes, i+1)
		case r == csi:
			i = skipCSI(runes, i+1)
		case isControlStringStart(r):
			i = skipControlString(runes, i+1)
		case isControl(r):
			i++
		default:
			out.WriteRune(r)
			i++
		}
	}
	return out.String()
}

func skipEscapeSequence(runes []rune, start int) int {
	if start >= len(runes) {
		return len(runes)
	}
	switch runes[start] {
	case '[':
		return skipCSI(runes, start+1)
	case ']', 'P', 'X', '^', '_':
		return skipControlString(runes, start+1)
	}

	i := start
	for i < len(runes) && runes[i] >= 0x20 && runes[i] <= 0x2f {
		i++
	}
	if i < len(runes) {
		return i + 1
	}
	return len(runes)
}

func skipCSI(runes []rune, start int) int {
	for i := start; i < len(runes); i++ {
		if runes[i] >= 0x40 && runes[i] <= 0x7e {
			return i + 1
		}
	}
	return len(runes)
}

func skipControlString(runes []rune, start int) int {
	for i := start; i < len(runes); i++ {
		switch runes[i] {
		case '\a', st:
			return i + 1
		case escape:
			if i+1 < len(runes) && runes[i+1] == '\\' {
				return i + 2
			}
		}
	}
	return len(runes)
}

func isControlStringStart(r rune) bool {
	switch r {
	case '\u0090', '\u0098', '\u009d', '\u009e', '\u009f':
		return true
	default:
		return false
	}
}

func isControl(r rune) bool {
	return r <= 0x1f || (r >= 0x7f && r <= 0x9f)
}
