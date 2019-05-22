package css

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

type Selector interface {
	Match(*html.Node) bool
	String() string
}

type AttributeSelector struct {
	Key   string
	Value string
	Type  string
	match func(string, string) bool
}

type ClassSelector struct{ *AttributeSelector }

type IDSelector struct{ *AttributeSelector }

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
	"contains":         contains,
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
		return func(n *html.Node) bool { return isElementNode(n) && !s.Match(n) }, err
	}
}

func (s *UniversalSelector) Match(n *html.Node) bool      { return true }
func (s *PseudoSelector) Match(n *html.Node) bool         { return s.match(n) }
func (s *PseudoFunctionSelector) Match(n *html.Node) bool { return s.match(n) }
func (s *ElementSelector) Match(n *html.Node) bool        { return n.Data == s.Element }
func (s *AttributeSelector) Match(n *html.Node) bool {
	for _, a := range n.Attr {
		if a.Key == s.Key {
			return s.match(a.Val, s.Value)
		}
	}
	return false
}

func (s *UnionSelector) Match(n *html.Node) bool {
	return s.SelectorA.Match(n) || s.SelectorB.Match(n)
}

func (s *SelectorSequence) Match(n *html.Node) bool {
	for i := len(s.Selectors) - 1; i >= 0; i-- {
		if !s.Selectors[i].Match(n) {
			return false
		}
	}
	return true
}

func (s *DescendantSelector) Match(n *html.Node) bool {
	if !s.Selector.Match(n) {
		return false
	}
	for n := n.Parent; n != nil; n = n.Parent {
		if n.Type == html.ElementNode && s.Ancestor.Match(n) {
			return true
		}
	}
	return false
}

func (s *ChildSelector) Match(n *html.Node) bool {
	return s.Selector.Match(n) && isElementNode(n.Parent) && s.Parent.Match(n.Parent)
}

func (s *SubsequentSiblingSelector) Match(n *html.Node) bool {
	if !s.Selector.Match(n) {
		return false
	}
	for n := n.PrevSibling; n != nil; n = n.PrevSibling {
		if n.Type == html.ElementNode && s.Sibling.Match(n) {
			return true
		}
	}
	return false
}

func (s *NextSiblingSelector) Match(n *html.Node) bool {
	return s.Selector.Match(n) && isElementNode(n.PrevSibling) && s.Sibling.Match(n.PrevSibling)
}

func (s *UniversalSelector) String() string      { return "*" }
func (s *ClassSelector) String() string          { return "." + s.Value }
func (s *IDSelector) String() string             { return "#" + s.Value }
func (s *PseudoSelector) String() string         { return ":" + s.Name }
func (s *PseudoFunctionSelector) String() string { return fmt.Sprintf(":%s%s", s.Name, s.Args) }
func (s *ElementSelector) String() string        { return s.Element }
func (s *UnionSelector) String() string          { return fmt.Sprintf("%s, %s", s.SelectorA, s.SelectorB) }
func (s *DescendantSelector) String() string     { return fmt.Sprintf("%s %s", s.Ancestor, s.Selector) }
func (s *ChildSelector) String() string          { return fmt.Sprintf("%s > %s", s.Parent, s.Selector) }
func (s *NextSiblingSelector) String() string    { return fmt.Sprintf("%s + %s", s.Sibling, s.Selector) }
func (s *SubsequentSiblingSelector) String() string {
	return fmt.Sprintf("%s ~ %s", s.Sibling, s.Selector)
}

func (s *AttributeSelector) String() string {
	if s.Type == "" {
		return fmt.Sprintf("[%s]", s.Key)
	}
	return fmt.Sprintf("[%s%s%q]", s.Key, s.Type, s.Value)
}

func (s *SelectorSequence) String() string {
	out := s.Selectors[0].String()
	for _, s := range s.Selectors[1:] {
		out += s.String()
	}
	return out
}
