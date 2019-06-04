package css

import (
	"testing"
)

type escapeTest struct{ unescaped, escapedID, escapedString string }

var escapeTests = []escapeTest{
	{"0123abc", "\\30 123abc", "0123abc"},
	{"-0123abc", "-\\30 123abc", "-0123abc"},
	{"-", "\\-", "-"},
	{"#foo.bar", "\\#foo\\.bar", "#foo.bar"},
	{"\000", "\uFFFD", "\uFFFD"},
	{"abc\000def", "abc\uFFFDdef", "abc\uFFFDdef"},
	{"\\ \"", "\\\\\\ \\\"", "\\\\ \\\""},
}

func TestEscape(t *testing.T) {
	for _, escapeTest := range escapeTests {
		if escapedID := EscapeIdentifier(escapeTest.unescaped); escapeTest.escapedID != escapedID {
			t.Errorf("escapeID\ngot:\n\t'%#v'\n\nexpected:\n\t'%#v'", escapedID, escapeTest.escapedID)
		}
		if escapedString := EscapeString(escapeTest.unescaped); escapeTest.escapedString != escapedString {
			t.Errorf("escapeString\ngot:\n\t'%#v'\n\nexpected:\n\t'%#v'", escapedString, escapeTest.escapedString)
		}
		if unescapedID := Unescape(escapeTest.escapedID); escapeTest.unescaped != unescapedID {
			t.Errorf("unescapeID\ngot:\n\t'%#v'\n\nexpected:\n\t'%#v'", unescapedID, escapeTest.unescaped)
		}
		if unescapedString := Unescape(escapeTest.escapedString); escapeTest.unescaped != unescapedString {
			t.Errorf("unescapeString\ngot:\n\t'%#v'\n\nexpected:\n\t'%#v'", unescapedString, escapeTest.unescaped)
		}
	}
}
