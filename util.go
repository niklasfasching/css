package css

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

var (
	complexNthRegexp = regexp.MustCompile(`^\s*([+-]?\d*)?n\s*([+-]?\s*\d+)?s*$`)
	simpleNthRegexp  = regexp.MustCompile(`^\s*([+-]?\d+)\s*$`)
	whitespaceRegexp = regexp.MustCompile(`\s`)
)

func isEmpty(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode || c.Type == html.TextNode {
			return false
		}
	}
	return true
}

func isInput(n *html.Node) bool {
	return n.Type == html.ElementNode && (n.Data == "input" || n.Data == "textarea")
}

func isRoot(n *html.Node) bool {
	return n.Type == html.ElementNode && n.Parent != nil && n.Parent.Type == html.DocumentNode
}

func parseNthArgs(args string) (a, b int, err error) {
	if args = strings.TrimSpace(args); args == "odd" {
		return 2, 1, nil
	} else if args == "even" {
		return 2, 0, nil
	} else if m := simpleNthRegexp.FindStringSubmatch(args); m != nil {
		b, err = atoi(m[1], "0")
		return 0, b, err
	} else if m := complexNthRegexp.FindStringSubmatch(args); m != nil {
		a, err = atoi(m[1], "1")
		if err != nil {
			return 0, 0, err
		}
		b, err = atoi(m[2], "0")
		if err != nil {
			return 0, 0, err
		}
		return a, b, nil
	}
	return 0, 0, fmt.Errorf("bad nth arguments: %q", args)
}

func nthSibling(next func(*html.Node) *html.Node, ofType bool) func(string) (func(*html.Node) bool, error) {
	return func(args string) (func(*html.Node) bool, error) {
		a, b, err := parseNthArgs(args)
		return func(n *html.Node) bool {
			anb := 0
			for s := next(n); s != nil; s = next(s) {
				if s.Type == html.ElementNode && (!ofType || s.Data == n.Data) {
					anb++
				}
			}
			return isNth(a, b, anb)
		}, err
	}
}

func nthSiblingCompiled(next func(*html.Node) *html.Node, args string, ofType bool) func(*html.Node) bool {
	f, err := nthSibling(next, ofType)(args)
	if err != nil {
		panic(err)
	}
	return f
}

func atoi(s, fallback string) (int, error) {
	s = whitespaceRegexp.ReplaceAllString(s, "")
	if s == "" {
		return 0, nil
	}
	if s == "+" || s == "-" {
		s = s + fallback
	}
	return strconv.Atoi(s)
}

func isNth(a, b, anb int) bool {
	an := anb - b
	return (a == 0 && b == anb) || (a != 0 && an/a >= 0 && an%a == 0)
}

func getAttribute(n *html.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val, true
		}
	}
	return "", false
}

func hasAttribute(n *html.Node, key string) bool {
	_, exists := getAttribute(n, key)
	return exists
}

func combine(a, b func(*html.Node) bool) func(*html.Node) bool {
	return func(n *html.Node) bool { return a(n) && b(n) }
}
