package css

import (
	"errors"
	"fmt"
	"strings"
)

type parser struct {
	tokens []token
	index  int
}

func (p *parser) next() token {
	if p.index == len(p.tokens) {
		return token{category: tokenEOF}
	}
	t := p.tokens[p.index]
	p.index++
	return t
}

func (p *parser) peek() token {
	t := p.next()
	p.index--
	return t
}

func (p *parser) backup() {
	if p.index == 0 {
		panic("cannot backup at start")
	}
	p.index--
}

func (p *parser) acceptRun(c tokenCategory) {
	for p.next().category == c {
	}
	p.backup()
}

func parse(tokens []token) (Selector, error) {
	p := &parser{tokens: tokens}
	s, err := p.parseSimpleSelectorSequence()
	if err != nil {
		return nil, err
	}
	for {
		if t := p.peek(); t.category == tokenEOF {
			return s, nil
		}
		s, err = p.parseComplexSelectorSequence(s)
		if err != nil {
			return nil, err
		}
	}
}

func (p *parser) parseSimpleSelectorSequence() (Selector, error) {
	s := SelectorSequence{}
	switch p.peek().category {
	case tokenIdent:
		element := strings.ToLower(p.next().string)
		s.Selectors = append(s.Selectors, &ElementSelector{element})
	case tokenUniversal:
		s.Selectors = append(s.Selectors, &UniversalSelector{p.next().string})
	}
loop:
	for {
		switch p.peek().category {
		case tokenClass:
			class := strings.ToLower(p.next().string)
			s.Selectors = append(s.Selectors, attributeSelector("class", class, "~="))
		case tokenID:
			id := strings.ToLower(p.next().string)
			s.Selectors = append(s.Selectors, attributeSelector("id", id, "="))
		case tokenBracketOpen:
			as, err := p.parseAttributeSelector()
			if err != nil {
				return nil, err
			}
			s.Selectors = append(s.Selectors, as)
		case tokenPseudoClass:
			name := p.next().string
			if PseudoClasses[name] == nil {
				return nil, errors.New("invalid pseudo selector: :" + name)
			}
			s.Selectors = append(s.Selectors, &PseudoSelector{name, PseudoClasses[name]})
		case tokenPseudoFunction:
			ps, err := p.parsePseudoFunctionSelector()
			if err != nil {
				return nil, err
			}
			s.Selectors = append(s.Selectors, ps)
		default:
			break loop
		}
	}
	if len(s.Selectors) == 0 {
		return nil, errors.New("empty simple selector sequence")
	}
	return &s, nil
}

func (p *parser) parseComplexSelectorSequence(s1 Selector) (Selector, error) {
	combinator := p.parseCombinator()
	s2, err := p.parseSimpleSelectorSequence()
	if err != nil {
		return nil, err
	}
	switch combinator {
	case " ":
		return &DescendantSelector{s1, s2}, nil
	case ">":
		return &ChildSelector{s1, s2}, nil
	case "+":
		return &NextSiblingSelector{s1, s2}, nil
	case "~":
		return &SubsequentSiblingSelector{s1, s2}, nil
	case ",":
		return &UnionSelector{s1, s2}, nil
	default:
		return nil, fmt.Errorf("bad combinator: '%s'", combinator)
	}
}

func (p *parser) parseAttributeSelector() (Selector, error) {
	if p.next().category != tokenBracketOpen {
		return nil, errors.New("invalid attribute selector")
	}
	t := p.next()
	if t.category != tokenIdent {
		return nil, errors.New("invalid attribute selector")
	}
	key, matcher := strings.ToLower(t.string), p.parseMatcher()
	if t := p.next(); matcher == "" && t.category == tokenBracketClose {
		return attributeSelector(key, "", ""), nil
	} else if matcher != "" && (t.category == tokenString || t.category == tokenIdent) {
		if p.next().category == tokenBracketClose {
			value := t.string
			if t.category == tokenString {
				value = value[1 : len(value)-2]
			}
			return attributeSelector(key, value, matcher), nil
		}
	}
	return nil, errors.New("invalid attribute selector")
}

func (p *parser) parsePseudoFunctionSelector() (Selector, error) {
	if p.peek().category != tokenPseudoFunction {
		return nil, errors.New("expected pseudo function")
	}
	name := strings.ToLower(p.next().string)
	f := PseudoFunctions[name]
	if p.peek().category != tokenFunctionArguments {
		return nil, errors.New("expected pseudo function arguments")
	}
	args := p.next().string
	match, err := f(args[1 : len(args)-1])
	if err != nil {
		return nil, err
	}
	return &PseudoFunctionSelector{name, args, match}, nil
}

func (p *parser) parseCombinator() string {
	combinator, space := "", p.peek().category == tokenSpace
	p.acceptRun(tokenSpace)
	if p.peek().category == tokenCombinator {
		combinator = p.next().string
	} else if space {
		combinator = " "
	} else {
		return ""
	}
	p.acceptRun(tokenSpace)
	return combinator
}

func (p *parser) parseMatcher() string {
	matcher := ""
	p.acceptRun(tokenSpace)
	if p.peek().category == tokenMatcher {
		matcher = p.next().string
	}
	p.acceptRun(tokenSpace)
	return matcher
}
