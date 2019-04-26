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

	for _, path := range selectorFiles() {
		log.Println(path)
		result := readResult(path)
		selectors := strings.Split(readFileString(path), "\n\n\n")
		for _, selector := range selectors {
			selector = strings.TrimSpace(selector)
			var actual interface{}
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
	path := "testdata/benchmark.txt"
	htmlString, result := readHTML(path), readResult(path)
	for _, selector := range strings.Split(readFileString(path), "\n\n\n") {
		selector = strings.TrimSpace(selector)
		matchAll := compile(selector)
		var selection []*html.Node
		for n := 0; n < b.N; n++ {
			selection = matchAll(htmlString)
		}
		if !reflect.DeepEqual(renderHTML(selection), result.Selections[selector]) {
			log.Printf("Bad result for %s:\nGot: %v\nExpected: %v",
				selector, renderHTML(selection), result.Selections[selector])
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
	for _, path := range selectorFiles() {
		log.Println(path)
		result := Result{
			Selectors:  map[string]interface{}{},
			Selections: map[string][]string{},
		}
		html := readHTML(path)
		selectors := strings.Split(readFileString(path), "\n\n\n")
		for _, selector := range selectors {
			selector = strings.TrimSpace(selector)
			compiled, err := Compile(selector)
			if err != nil {
				result.Selectors[selector] = err.Error()
				continue
			}
			result.Selectors[selector] = compiled
			if html != nil {
				result.Selections[selector] = renderHTML(All(compiled, html))
			}
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

func selectorFiles() []string {
	dir := "./testdata"
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		panic(fmt.Sprintf("Could not read directory: %s", err))
	}
	selectorFiles := []string{}
	for _, f := range files {
		name := f.Name()
		if filepath.Ext(name) != ".txt" {
			continue
		}
		selectorFiles = append(selectorFiles, filepath.Join(dir, name))
	}
	return selectorFiles
}

func readFileString(path string) string {
	bs, err := ioutil.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(bs)
}

func readHTML(selectorFilePath string) *html.Node {
	path := selectorFilePath[:len(selectorFilePath)-len(".txt")] + ".html"
	bs, err := ioutil.ReadFile(path)
	if err != nil {
		return nil
	}
	n, err := html.Parse(bytes.NewReader(bs))
	if err != nil {
		panic(err)
	}
	return n
}

func readResult(selectorFilePath string) (result Result) {
	path := selectorFilePath[:len(selectorFilePath)-len(".txt")] + ".json"
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

func writeResult(selectorFilePath string, result Result) {
	path := selectorFilePath[:len(selectorFilePath)-len(".txt")] + ".json"
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
