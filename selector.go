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
	match func(string, string) bool
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
}

type ChildSelector struct {
	Parent   Selector
	Selector Selector
}

type NextSiblingSelector struct {
	Sibling  Selector
	Selector Selector
}

type SubsequentSiblingSelector struct {
	Sibling  Selector
	Selector Selector
}

type UnionSelector struct {
	SelectorA Selector
	SelectorB Selector
}

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
	"first-child":   nthSiblingCompiled(func(n *html.Node) *html.Node { return n.PrevSibling }, "1", false),
	"first-of-type": nthSiblingCompiled(func(n *html.Node) *html.Node { return n.PrevSibling }, "1", true),
	"last-child":    nthSiblingCompiled(func(n *html.Node) *html.Node { return n.NextSibling }, "1", false),
	"last-of-type":  nthSiblingCompiled(func(n *html.Node) *html.Node { return n.NextSibling }, "1", true),
	"only-child":    onlyChild(false),
	"only-of-type":  onlyChild(true),
}

var PseudoFunctions = map[string]func(string) (func(*html.Node) bool, error){
	"not":              nil,
	"nth-child":        nthSibling(func(n *html.Node) *html.Node { return n.PrevSibling }, false),
	"nth-last-child":   nthSibling(func(n *html.Node) *html.Node { return n.NextSibling }, false),
	"nth-of-type":      nthSibling(func(n *html.Node) *html.Node { return n.PrevSibling }, true),
	"nth-last-of-type": nthSibling(func(n *html.Node) *html.Node { return n.NextSibling }, true),
}

var Matchers = map[string]func(string, string) bool{
	"~=": includeMatch,
	"|=": func(av, sv string) bool { return av == sv || strings.HasPrefix(av, sv+"-") },
	"^=": func(av, sv string) bool { return strings.HasPrefix(av, sv) },
	"$=": func(av, sv string) bool { return strings.HasSuffix(av, sv) },
	"*=": func(av, sv string) bool { return strings.Contains(av, sv) },
	"=":  func(av, sv string) bool { return av == av },
	"":   func(string, string) bool { return true },
}

var Combinators = map[string]func(Selector, Selector) Selector{
	" ": func(s1, s2 Selector) Selector { return &DescendantSelector{s1, s2} },
	">": func(s1, s2 Selector) Selector { return &ChildSelector{s1, s2} },
	"+": func(s1, s2 Selector) Selector { return &NextSiblingSelector{s1, s2} },
	"~": func(s1, s2 Selector) Selector { return &SubsequentSiblingSelector{s1, s2} },
	",": func(s1, s2 Selector) Selector { return &UnionSelector{s1, s2} },
}

func init() {
	PseudoFunctions["not"] = func(args string) (func(*html.Node) bool, error) {
		s, err := Compile(args)
		return func(n *html.Node) bool { return n.Type == html.ElementNode && !s.Match(n) }, err
	}
}

func (s *UniversalSelector) Match(n *html.Node) bool { return n.Type == html.ElementNode }

func (s *PseudoSelector) Match(n *html.Node) bool {
	return n.Type == html.ElementNode && s.match(n)
}

func (s *PseudoFunctionSelector) Match(n *html.Node) bool {
	return n.Type == html.ElementNode && s.match(n)
}

func (s *ElementSelector) Match(n *html.Node) bool {
	return n.Type == html.ElementNode && n.Data == s.Element
}

func (s *AttributeSelector) Match(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	for _, a := range n.Attr {
		if a.Key == s.Key {
			return s.match(a.Val, s.Value)
		}
	}
	return false
}

func (s *SelectorSequence) Match(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	for i := len(s.Selectors) - 1; i >= 0; i-- {
		if !s.Selectors[i].Match(n) {
			return false
		}
	}
	return true
}

func (s *DescendantSelector) Match(n *html.Node) bool {
	if n.Type == html.ElementNode && s.Selector.Match(n) {
		for n := n.Parent; n != nil; n = n.Parent {
			if n.Type == html.ElementNode && s.Ancestor.Match(n) {
				return true
			}
		}
	}
	return false
}

func (s *ChildSelector) Match(n *html.Node) bool {
	return n.Type == html.ElementNode && n.Parent != nil && s.Selector.Match(n) && s.Parent.Match(n.Parent)
}

func (s *SubsequentSiblingSelector) Match(n *html.Node) bool {
	if n.Type == html.ElementNode && s.Selector.Match(n) {
		for n := n.PrevSibling; n != nil; n = n.PrevSibling {
			if n.Type == html.ElementNode && s.Sibling.Match(n) {
				return true
			}
		}
	}
	return false
}

func (s *NextSiblingSelector) Match(n *html.Node) bool {
	return n.Type == html.ElementNode && n.PrevSibling != nil && s.Selector.Match(n) && s.Sibling.Match(n.PrevSibling)
}

func (s *UnionSelector) Match(n *html.Node) bool {
	return n.Type == html.ElementNode && (s.SelectorA.Match(n) || s.SelectorB.Match(n))
}
