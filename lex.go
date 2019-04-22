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
	tokenNumber
	tokenString
	tokenMatcher
	tokenCombinator
	tokenBracketOpen
	tokenBracketClose
	tokenParenthesisOpen
	tokenParenthesisClose
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
	r := l.next()
	l.backup()
	return r
}

func (l *lexer) backup() {
	l.index -= l.width
	_, w := utf8.DecodeRuneInString(l.input[l.index:])
	l.width = w
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
		l.emit(tokenParenthesisOpen)
		return lexSpace
	case r == ')':
		l.emit(tokenParenthesisClose)
		return lexSpace
	case r == '*':
		l.emit(tokenUniversal)
		return lexSpace
	case r == '.' && isDigit(l.peek()), isDigit(r):
		l.backup()
		return lexNumber
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
func isMatchChar(r rune) bool      { return strings.ContainsRune("~|^$*", r) }
func isCombinatorChar(r rune) bool { return strings.ContainsRune("+~>,", r) }

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

func lexClass(l *lexer) stateFn {
	err := acceptIdentifier(l)
	if err != nil {
		return l.errorf("%s", err)
	}
	l.emit(tokenClass)
	return lexSpace
}

func lexString(l *lexer) stateFn {
	quote := l.next()
	l.ignore()
	for {
		switch r := l.next(); {
		case r == eof:
			return l.errorf("invalid unterminated string: %s", string(quote))
		case r == quote:
			l.backup()
			l.emit(tokenString)
			l.next()
			l.ignore()
			return lexSpace
		}
	}
}

func lexID(l *lexer) stateFn {
	if !isNameChar(l.peek()) {
		l.errorf("invalid starting char for ID")
	}
	l.acceptRun(isNameChar)
	l.emit(tokenID)
	return lexSpace
}

func lexNumber(l *lexer) stateFn {
	l.acceptRun(isDigit)
	if l.peek() == '.' {
		l.next()
		if !isDigit(l.peek()) {
			return l.errorf("invalid number")
		}
		l.acceptRun(isDigit)
	} else if l.index == l.start {
		return l.errorf("invalid number")
	}
	l.emit(tokenNumber)
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
	l.emit(tokenIdent)
	return lexSpace
}
