package css

import (
	"golang.org/x/net/html"
)

func getAttribute(n *html.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val, true
		}
	}
	return "", false
}

func negatedMatch(f func(*html.Node) bool) func(*html.Node) bool {
	return func(n *html.Node) bool { return !f(n) }
}
