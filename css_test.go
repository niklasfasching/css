package css

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/andybalholm/cascadia"
	"github.com/ericchiang/css"
	"golang.org/x/net/html"
)

var updateTestData = flag.Bool("update-test-data", false, "update test data rather than actually running tests")

type Result struct {
	Selectors  map[string]interface{}
	Selections map[string][]string `json:",omitempty"`
}

func TestCSS(t *testing.T) {
	if *updateTestData {
		update()
		log.Println("Updated test data")
		t.Skip()
	}

	for _, path := range htmlFiles() {
		log.Println(path)
		result := readResult(path)
		document, selectors := readHTML(path)
		for _, selector := range selectors {
			actual := interface{}(nil)
			compiled, err := Compile(selector)
			if err != nil {
				actual = err.Error()
			} else {
				actual = compiled
			}
			expected := result.Selectors[selector]
			if !reflect.DeepEqual(interfacify(actual), expected) {
				t.Errorf("%s\ngot:\n\t'%s'\n\nexpected:\n\t'%s'", selector, jsonify(actual), jsonify(expected))
			}
			if compiled != nil && compiled.String() != selector {
				t.Errorf("%s: bad string conversion\ngot:\n\t'%s'\n\nexpected:\n\t'%s'", selector, compiled, selector)
			}
		}
	}
}

func BenchmarkNiklasFaschingCSS(b *testing.B) {
	benchmark(b, func(selector string) func(*html.Node) []*html.Node {
		s := MustCompile(selector)
		return func(html *html.Node) []*html.Node { return All(s, html) }
	})
}

func BenchmarkEricChiangCSS(b *testing.B) {
	benchmark(b, func(selector string) func(*html.Node) []*html.Node {
		s := css.MustCompile(selector)
		return func(html *html.Node) []*html.Node { return s.Select(html) }
	})
}

func BenchmarkAndyBalholmCSS(b *testing.B) {
	benchmark(b, func(selector string) func(*html.Node) []*html.Node {
		s := cascadia.MustCompile(selector)
		return func(html *html.Node) []*html.Node { return s.MatchAll(html) }
	})
}

func benchmark(b *testing.B, compile func(string) func(*html.Node) []*html.Node) {
	defer func() {
		if err := recover(); err != nil {
			b.Skip(err)
		}
	}()
	path := "testdata/benchmark.html"
	document, selectors := readHTML(path)
	result := readResult(path)
	for _, selector := range selectors {
		matchAll := compile(selector)
		var selection []*html.Node
		for n := 0; n < b.N; n++ {
			selection = matchAll(document)
		}
		actual, expected := renderHTML(selection), result.Selections[selector]
		if !reflect.DeepEqual(actual, expected) {
			b.Logf("%s\n\tgot:\n\t'%#v'\n\n\texpected:\n\t'%#v'", selector, actual, expected)
		}
	}
}

func interfacify(in interface{}) (out interface{}) {
	bs, err := json.Marshal(in)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(bs, &out)
	if err != nil {
		panic(err)
	}
	return out
}

func update() {
	for _, path := range htmlFiles() {
		log.Println(path)
		result := Result{
			Selectors:  map[string]interface{}{},
			Selections: map[string][]string{},
		}
		document, selectors := readHTML(path)
		for _, selector := range selectors {
			compiled, err := Compile(selector)
			if err != nil {
				result.Selectors[selector] = err.Error()
				continue
			}
			result.Selectors[selector] = compiled
			result.Selections[selector] = renderHTML(All(compiled, document))
		}
		writeResult(path, result)
	}
}

func renderHTML(ns []*html.Node) []string {
	out := make([]string, len(ns))
	for i, n := range ns {
		var s strings.Builder
		err := html.Render(&s, n)
		if err != nil {
			panic(err)
		}
		out[i] = s.String()
	}
	return out
}

func htmlFiles() (out []string) {
	dir := "./testdata"
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		panic(fmt.Sprintf("Could not read directory: %s", err))
	}
	for _, f := range files {
		name := f.Name()
		if filepath.Ext(name) != ".html" {
			continue
		}
		out = append(out, filepath.Join(dir, name))
	}
	return out
}

func readHTML(path string) (n *html.Node, selectors []string) {
	bs, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	n, err = html.Parse(bytes.NewReader(bs))
	if err != nil {
		panic(err)
	}
	selectorsText := html.UnescapeString(First(MustCompile("style"), n).FirstChild.Data)
	selectors = regexp.MustCompile(`\s*{.*}\s*`).Split(strings.TrimSpace(selectorsText), -1)
	if l := len(selectors); l > 0 && selectors[l-1] == "" {
		selectors = selectors[:l-1]
	}
	return n, selectors
}

func readResult(htmlFilePath string) (result Result) {
	path := htmlFilePath[:len(htmlFilePath)-len(".html")] + ".json"
	bs, err := ioutil.ReadFile(path)
	if err != nil {
		bs = []byte("{}")
	}
	err = json.Unmarshal(bs, &result)
	if err != nil {
		panic(err)
	}
	return result
}

func writeResult(htmlFilePath string, result Result) {
	path := htmlFilePath[:len(htmlFilePath)-len(".html")] + ".json"
	b := &bytes.Buffer{}
	encoder := json.NewEncoder(b)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(result)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(path, b.Bytes(), 0644)
	if err != nil {
		panic(err)
	}
}

func jsonify(v interface{}) string {
	bs, err := json.MarshalIndent(v, "\t", "  ")
	if err != nil {
		panic(err)
	}
	return string(bs)
}
