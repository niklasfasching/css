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
	Name string
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

var PseudoClasses = map[string]func(*html.Node) bool{
	"checked":       (&AttributeSelector{"checked", "", ""}).Match,
	"disabled":      (&AttributeSelector{"disabled", "", ""}).Match,
	"empty":         nil,
	"enabled":       negatedMatch((&AttributeSelector{"disabled", "", ""}).Match),
	"first":         nil,
	"first-child":   nil,
	"first-of-type": nil,
	"fullscreen":    nil,
	"focus":         nil,
	"hover":         nil,
	"indeterminate": nil,
	"in-range":      nil,
	"invalid":       nil,
	"last-child":    nil,
	"last-of-type":  nil,
	"left":          nil,
	"link":          nil,
	"only-child":    nil,
	"only-of-type":  nil,
	"optional":      nil,
	"out-of-range":  nil,
	"read-only":     nil,
	"read-write":    nil,
	"required":      nil,
	"right":         nil,
	"root":          nil,
	"scope":         nil,
	"target":        nil,
	"valid":         nil,
}

var PseudoFunctions = map[string]func(*html.Node) bool{
	"dir":              nil,
	"lang":             nil,
	"not":              nil,
	"nth-child":        nil,
	"nth-last-child":   nil,
	"nth-last-of-type": nil,
	"nth-of-type":      nil,
}

func (s *UniversalSelector) Match(*html.Node) bool { return true }

func (s *PseudoSelector) Match(n *html.Node) bool { return PseudoClasses[s.Name](n) }

func (s *ElementSelector) Match(n *html.Node) bool {
	return n.Type == html.ElementNode && n.Data == s.Element // TODO: where is element name stored
}

func (s *AttributeSelector) Match(n *html.Node) bool {
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
