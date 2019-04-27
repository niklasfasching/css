/*
https://www.w3.org/TR/2018/CR-selectors-3-20180130/#w3cselgrammar
*/
package css

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

type token struct {
	category tokenCategory
	string   string
	index    int
}

type tokenCategory int

const (
	tokenEOF tokenCategory = iota
	tokenSpace
	tokenUniversal
	tokenClass
	tokenIdent
	tokenID
	tokenPseudoClass
	tokenPseudoFunction
	tokenFunctionArguments
	tokenString
	tokenMatcher
	tokenCombinator
	tokenBracketOpen
	tokenBracketClose
)

const eof = -1

type stateFn func(*lexer) stateFn

type lexer struct {
	input  string
	index  int
	start  int
	width  int
	tokens []token
	error  error
}

func lex(input string) ([]token, error) {
	l := &lexer{input: strings.TrimSpace(input)}
	for state := lexSpace; state != nil; state = state(l) {
	}
	return l.tokens, l.error
}

func (l *lexer) next() rune {
	if l.index >= len(l.input) {
		l.width = 0
		return eof
	}
	r, w := utf8.DecodeRuneInString(l.input[l.index:])
	l.width = w
	l.index += l.width
	return r
}

func (l *lexer) peek() rune {
	if l.index >= len(l.input) {
		return eof
	}
	r, _ := utf8.DecodeRuneInString(l.input[l.index:])
	return r
}

func (l *lexer) backup() {
	l.index -= l.width
}

func (l *lexer) emit(c tokenCategory) {
	l.tokens = append(l.tokens, token{c, l.input[l.start:l.index], l.start})
	l.start = l.index
}

func (l *lexer) ignore() {
	l.start = l.index
}

func (l *lexer) acceptRun(f func(rune) bool) {
	for f(l.next()) {
	}
	l.backup()
}

func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.error = fmt.Errorf(format, args...)
	return nil
}

func lexSpace(l *lexer) stateFn {
	if isWhitespace(l.peek()) {
		l.acceptRun(isWhitespace)
		l.emit(tokenSpace)
	}
	switch r := l.next(); {
	case isMatchChar(r) && l.peek() == '=':
		l.next()
		l.emit(tokenMatcher)
		return lexSpace
	case r == '=':
		l.emit(tokenMatcher)
		return lexSpace
	case isCombinatorChar(r):
		l.emit(tokenCombinator)
		return lexSpace
	case r == '[':
		l.emit(tokenBracketOpen)
		return lexSpace
	case r == ']':
		l.emit(tokenBracketClose)
		return lexSpace
	case r == '(':
		l.backup()
		return lexFunctionArguments
	case r == '*':
		l.emit(tokenUniversal)
		return lexSpace
	case r == '.':
		l.ignore()
		return lexClass
	case r == '#':
		l.ignore()
		return lexID
	case r == ':':
		l.ignore()
		return lexPseudo
	case r == '\'', r == '"':
		l.backup()
		return lexString
	case r == eof:
		l.emit(tokenEOF)
		return nil
	default:
		l.backup()
		return lexIdent
	}
}

// isNameStart checks whether rune r is a valid character as the start of a name
// [_a-z]|{nonascii}|{escape}
func isNameStart(r rune) bool {
	return 'a' <= r && r <= 'z' || 'A' <= r && r <= 'Z' || r == '_' || r > 127
}

// isNameChar checks whether rune r is a valid character as a part of a name
// [_a-z0-9-]|{nonascii}|{escape}
func isNameChar(r rune) bool {
	return 'a' <= r && r <= 'z' || 'A' <= r && r <= 'Z' || '0' <= r && r <= '9' ||
		r == '_' || r == '-' || r > 127
}

func isWhitespace(r rune) bool     { return strings.ContainsRune(" \t\f\r\n", r) }
func isDigit(r rune) bool          { return '0' <= r && r <= '9' }
func isMatchChar(r rune) bool      { return Matchers[string(r)+"="] != nil }
func isCombinatorChar(r rune) bool { return Combinators[string(r)] != nil }

func acceptIdentifier(l *lexer) error {
	if l.peek() == '-' {
		l.next()
	}
	if !isNameStart(l.peek()) {
		return errors.New("invalid starting char for identifier")
	}
	l.acceptRun(isNameChar)
	return nil
}

func acceptString(l *lexer) error {
	quote := l.next()
	if !strings.ContainsRune(`'"`, quote) {
		return fmt.Errorf("invalid quoting char for string: %s", string(quote))
	}
	for r := l.next(); r != quote; r = l.next() {
		switch {
		case r == eof:
			return fmt.Errorf("unterminated quoted string")
		case r == '\n', r == '\r', r == '\f':
			return fmt.Errorf("unescaped %q", string(r))
		case r == '\\':
			l.next()
		}
	}
	return nil
}

func lexClass(l *lexer) stateFn {
	err := acceptIdentifier(l)
	if err != nil {
		return l.errorf("%s", err)
	}
	l.emit(tokenClass)
	return lexSpace
}

func lexString(l *lexer) stateFn {
	if err := acceptString(l); err != nil {
		l.errorf("%s", err)
	}
	l.emit(tokenString)
	return lexSpace
}

func lexID(l *lexer) stateFn {
	if !isNameChar(l.peek()) {
		l.errorf("invalid starting char for ID")
	}
	l.acceptRun(isNameChar)
	l.emit(tokenID)
	return lexSpace
}

func lexPseudo(l *lexer) stateFn {
	if l.peek() == ':' {
		return l.errorf("invalid use of pseudo element")
	}
	err := acceptIdentifier(l)
	if err != nil {
		return l.errorf("%s", err)
	}
	if l.peek() == '(' {
		l.emit(tokenPseudoFunction)
	} else {
		l.emit(tokenPseudoClass)
	}
	return lexSpace
}

func lexIdent(l *lexer) stateFn {
	err := acceptIdentifier(l)
	if err != nil {
		return l.errorf("%s", err)
	}
	if l.start == l.index {
		return l.errorf("invalid identifier")
	}
	l.emit(tokenIdent)
	return lexSpace
}

func lexFunctionArguments(l *lexer) stateFn {
	if l.next() != '(' {
		return l.errorf("invalid start of function arguments")
	}
	for r, lvl := l.next(), 1; lvl != 0; r = l.next() {
		switch r {
		case eof:
			return l.errorf("unterminated function arguments")
		case '(':
			lvl++
		case ')':
			lvl--
		case '"', '\'':
			if err := acceptString(l); err != nil {
				return l.errorf("%s", err)
			}
		}
	}
	l.emit(tokenFunctionArguments)
	return lexSpace
}
