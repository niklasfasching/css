//css implements a css selector compiler for the net/html dom.

package css

import (
	"golang.org/x/net/html"
)

func Compile(selector string) (Selector, error) {
	tokens, err := lex(selector)
	if err != nil {
		return nil, err
	}
	return parse(tokens)
}

func MustCompile(selector string) Selector {
	s, err := Compile(selector)
	if err != nil {
		panic(err)
	}
	return s
}

func First(s Selector, n *html.Node) *html.Node {
	if s.Match(n) {
		return n
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if n := First(s, c); n != nil {
			return n
		}
	}
	return nil
}

func All(s Selector, n *html.Node) []*html.Node {
	var ns []*html.Node
	if s.Match(n) {
		ns = append(ns, n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		ns = append(ns, All(s, c)...)
	}
	return ns
}
