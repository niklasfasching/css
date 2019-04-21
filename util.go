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

func hasAttribute(n *html.Node, key string) bool {
	_, exists := getAttribute(n, key)
	return exists
}

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
