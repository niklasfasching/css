// https://drafts.csswg.org/cssom/#common-serializing-idioms
package css

import (
	"strconv"
	"unicode"
	"unicode/utf8"
)

func EscapeIdentifier(unescaped string) (escaped string) {
	for i := 0; i < len(unescaped); {
		r, w := utf8.DecodeRuneInString(unescaped[i:])
		switch {
		case r == '\u0000':
			escaped += string('\uFFFD')
		case r >= '\u0001' && r <= '\u001F', r == '\u007F',
			i == 0 && r >= '0' && r <= '9',
			i == 1 && r >= '\u0030' && r <= '\u0039' && unescaped[0] == '\u002D':
			escaped += `\` + strconv.FormatInt(int64(r), 16) + " "
		case i == 0 && len(unescaped) == 1 && r == '-':
			escaped += `\-`
		case r == '\u002D' || r == '\u005F' || r >= '\u0080' ||
			r >= '\u0030' && r <= '\u0039' ||
			r >= '\u0041' && r <= '\u005A' ||
			r >= '\u0061' && r <= '\u007A':
			escaped += string(r)
		default:
			escaped += `\` + string(r)
		}
		i += w
	}
	return escaped
}

func EscapeString(unescaped string) (escaped string) {
	for i := 0; i < len(unescaped); {
		r, w := utf8.DecodeRuneInString(unescaped[i:])
		switch {
		case r == '\u0000':
			escaped += string('\uFFFD')
		case r >= '\u0001' && r <= '\u001F', r == '\u007F':
			escaped += `\` + strconv.FormatInt(int64(r), 16) + " "
		case r == '"' || r == '\\':
			escaped += `\` + string(r)
		default:
			escaped += string(r)
		}
		i += w
	}
	return escaped
}

func Unescape(escaped string) (unescaped string) {
	for i := 0; i < len(escaped); {
		r, w := utf8.DecodeRuneInString(escaped[i:])
		i += w
		switch {
		case r == '\uFFFD':
			unescaped += string('\u0000')
		case r == '\\' && i < len(escaped) && !isHexDigit(rune(escaped[i])):
			unescaped += string(escaped[i])
			i++
		case r == '\\' && i < len(escaped):
			j := i
			for ; j < i+6 && j < len(escaped) && isHexDigit(rune(escaped[j])); j++ {
			}
			r, err := strconv.ParseUint(escaped[i:j], 16, 64)
			if err != nil {
				panic(err)
			}
			unescaped, i = unescaped+string(r), j
			if i < len(escaped) && unicode.IsSpace(rune(escaped[i])) {
				i++
			}
		default:
			unescaped += string(r)
		}
	}
	return unescaped
}
