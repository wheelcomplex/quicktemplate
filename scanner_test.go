package quicktemplate

import (
	"bufio"
	"bytes"
	"reflect"
	"testing"
)

func TestScannerSuccess(t *testing.T) {
	scannerTestSuccess(t, "", nil)
	scannerTestSuccess(t, "foobar", []tt{{ID: Text, Value: "foobar"}})
	scannerTestSuccess(t, "{% foo bar baz %}", []tt{
		{ID: TagName, Value: "foo"},
		{ID: TagContents, Value: "bar baz"},
	})
	scannerTestSuccess(t, "foo{%bar%}baz", []tt{
		{ID: Text, Value: "foo"},
		{ID: TagName, Value: "bar"},
		{ID: TagContents, Value: ""},
		{ID: Text, Value: "baz"},
	})
	scannerTestSuccess(t, "{{{%\n\r\tfoo bar\n\rbaz%%\n   \r %}}", []tt{
		{ID: Text, Value: "{{"},
		{ID: TagName, Value: "foo"},
		{ID: TagContents, Value: "bar\n\rbaz%%"},
		{ID: Text, Value: "}"},
	})
}

func scannerTestSuccess(t *testing.T, str string, expectedTokens []tt) {
	r := bytes.NewBufferString(str)
	br := bufio.NewReader(r)
	s := NewScanner(br)
	var tokens []tt
	for s.Next() {
		tokens = append(tokens, tt{
			ID:    s.Token().ID,
			Value: string(s.Token().Value),
		})
	}
	if err := s.LastError(); err != nil {
		t.Fatalf("unexpected error: %s. str=%q", err, str)
	}
	if !reflect.DeepEqual(tokens, expectedTokens) {
		t.Fatalf("unexpected tokens %v. Expecting %v. str=%q", tokens, expectedTokens, str)
	}
}

type tt struct {
	ID    int
	Value string
}
