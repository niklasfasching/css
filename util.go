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
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode || c.Type == html.TextNode {
			return false
		}
	}
	return true
}

func isInput(n *html.Node) bool {
	return n.Data == "input" || n.Data == "textarea"
}

func isRoot(n *html.Node) bool {
	return n.Parent != nil && n.Parent.Type == html.DocumentNode
}

func onlyChild(ofType bool) func(*html.Node) bool {
	return func(n *html.Node) bool {
		if n.Parent == nil {
			return true
		}
		count := 0
		for c := n.Parent.FirstChild; c != nil && count <= 1; c = c.NextSibling {
			if c.Type == html.ElementNode && (!ofType || c.Data == n.Data) {
				count++
			}
		}
		return count == 1
	}
}

func parseNthArgs(args string) (int, int, error) {
	if args = strings.TrimSpace(args); args == "odd" {
		return 2, 1, nil
	} else if args == "even" {
		return 2, 0, nil
	} else if m := simpleNthRegexp.FindStringSubmatch(args); m != nil {
		b, err := atoi(m[1], "0")
		return 0, b, err
	} else if m := complexNthRegexp.FindStringSubmatch(args); m != nil {
		a, err := atoi(m[1], "1")
		if err != nil {
			return 0, 0, err
		}
		b, err := atoi(m[2], "0")
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
			nth := 1
			for s := next(n); s != nil; s = next(s) {
				if s.Type == html.ElementNode && (!ofType || s.Data == n.Data) {
					nth++
				}
			}
			return isNth(a, b, nth)
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

// isNth checks whether y is a valid result for the given a and b.
// The formula is y = (a*n+b) with n being any positive integer, starting with 1.
// If a is 0 a*n is 0 and y must be b - otherwise a must fit into y-b n times, i.e. 1 or more times
// without any remainder.
func isNth(a, b, y int) bool {
	an := (y - b)
	return (a == 0 && b == y) || (a != 0 && an/a >= 0 && an%a == 0)
}

func hasAttribute(n *html.Node, key string) bool {
	for _, a := range n.Attr {
		if a.Key == key {
			return true
		}
	}
	return false
}

func attributeSelector(key, value, kind string) *AttributeSelector {
	if Matchers[kind] == nil {
		panic("invalid match type for attribute selector: " + kind)
	}
	return &AttributeSelector{key, value, kind, Matchers[kind]}
}

func includeMatch(value, sValue string) bool {
	for {
		if i := strings.IndexAny(value, " \t\r\n\f"); i == -1 {
			return value == sValue
		} else if value[:i] == sValue {
			return true
		} else {
			value = value[i+1:]
		}
	}
}

func isElementNode(n *html.Node) bool {
	return n != nil && n.Type == html.ElementNode
}
