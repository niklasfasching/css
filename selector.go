package css

import (
	"strings"

	"golang.org/x/net/html"
)

type Selector interface {
	Match(*html.Node) bool
}

type AttributeSelector struct {
	Key   string
	Value string
	Type  string
}

type UniversalSelector struct {
	Element string
}

type PseudoSelector struct {
	Name  string
	match func(*html.Node) bool
}

type PseudoFunctionSelector struct {
	Name  string
	Args  string
	match func(*html.Node) bool
}

type ElementSelector struct {
	Element string
}

type SelectorSequence struct {
	Selectors []Selector
}

type DescendantSelector struct {
	Ancestor Selector
	Selector Selector
	Direct   bool
}

type SiblingSelector struct {
	Sibling   Selector
	Selector  Selector
	Immediate bool
}

type UnionSelector struct {
	SelectorA Selector
	SelectorB Selector
}

var (
	firstChild = nthSiblingCompiled(func(n *html.Node) *html.Node { return n.PrevSibling }, "1", false)
	firstType  = nthSiblingCompiled(func(n *html.Node) *html.Node { return n.PrevSibling }, "1", true)
	lastChild  = nthSiblingCompiled(func(n *html.Node) *html.Node { return n.NextSibling }, "1", false)
	lastType   = nthSiblingCompiled(func(n *html.Node) *html.Node { return n.NextSibling }, "1", true)
	onlyChild  = combine(firstChild, lastChild)
	onlyType   = combine(firstType, lastType)
)

var PseudoClasses = map[string]func(*html.Node) bool{
	"root":          isRoot,
	"empty":         isEmpty,
	"checked":       func(n *html.Node) bool { return isInput(n) && hasAttribute(n, "checked") },
	"disabled":      func(n *html.Node) bool { return isInput(n) && hasAttribute(n, "disabled") },
	"enabled":       func(n *html.Node) bool { return isInput(n) && !hasAttribute(n, "disabled") },
	"optional":      func(n *html.Node) bool { return isInput(n) && !hasAttribute(n, "required") },
	"required":      func(n *html.Node) bool { return isInput(n) && hasAttribute(n, "required") },
	"read-only":     func(n *html.Node) bool { return isInput(n) && hasAttribute(n, "readonly") },
	"read-write":    func(n *html.Node) bool { return isInput(n) && !hasAttribute(n, "readonly") },
	"first-child":   firstChild,
	"first-of-type": firstType,
	"last-child":    lastChild,
	"last-of-type":  lastType,
	"only-child":    onlyChild,
	"only-of-type":  onlyType,
}

var PseudoFunctions = map[string]func(string) (func(*html.Node) bool, error){
	"not":              nil,
	"nth-child":        nthSibling(func(n *html.Node) *html.Node { return n.PrevSibling }, false),
	"nth-last-child":   nthSibling(func(n *html.Node) *html.Node { return n.NextSibling }, false),
	"nth-of-type":      nthSibling(func(n *html.Node) *html.Node { return n.PrevSibling }, true),
	"nth-last-of-type": nthSibling(func(n *html.Node) *html.Node { return n.NextSibling }, true),
}

func init() {
	PseudoFunctions["not"] = func(args string) (func(*html.Node) bool, error) {
		s, err := Compile(args)
		return func(n *html.Node) bool { return n.Type == html.ElementNode && !s.Match(n) }, err
	}
}

func (s *UniversalSelector) Match(*html.Node) bool { return true }

func (s *PseudoSelector) Match(n *html.Node) bool { return s.match(n) }

func (s *PseudoFunctionSelector) Match(n *html.Node) bool { return s.match(n) }

func (s *ElementSelector) Match(n *html.Node) bool {
	return n.Type == html.ElementNode && n.Data == s.Element // TODO: where is element name stored
}

func (s *AttributeSelector) Match(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	value, exists := getAttribute(n, s.Key)
	switch s.Type {
	case "~=":
		for _, v := range strings.Fields(value) {
			if s.Value == v {
				return true
			}
		}
		return false
	case "|=":
		return s.Value == value || strings.HasPrefix(value, s.Value+"-")
	case "^=":
		return strings.HasPrefix(value, s.Value)
	case "$=":
		return strings.HasSuffix(value, s.Value)
	case "*=":
		return strings.Contains(value, s.Value)
	case "=":
		return s.Value == value
	case "":
		return exists
	default:
		panic("invalid match type for attribute selector: " + s.Type)
	}
}

func (s *SelectorSequence) Match(n *html.Node) bool {
	for _, cs := range s.Selectors {
		if !cs.Match(n) {
			return false
		}
	}
	return true
}

func (s *DescendantSelector) Match(n *html.Node) bool {
	if s.Selector.Match(n) {
		for n, direct := n.Parent, true; n != nil && (!s.Direct || direct); n, direct = n.Parent, false {
			if s.Ancestor.Match(n) {
				return true
			}
		}
	}
	return false
}

func (s *SiblingSelector) Match(n *html.Node) bool {
	if s.Selector.Match(n) {
		for n, immediate := n.PrevSibling, true; n != nil && (!s.Immediate || immediate); n, immediate = n.PrevSibling, false {
			if s.Sibling.Match(n) {
				return true
			}
		}
	}
	return false
}

func (s *UnionSelector) Match(n *html.Node) bool {
	return s.SelectorA.Match(n) || s.SelectorB.Match(n)
}
