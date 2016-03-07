package quicktemplate

import (
	"bytes"
	"reflect"
	"testing"
)

func TestScannerSuccess(t *testing.T) {
	testScannerSuccess(t, "", nil)
	testScannerSuccess(t, "a%}{foo}bar", []tt{{ID: Text, Value: "a%}{foo}bar"}})
	testScannerSuccess(t, "{% foo bar baz(a, b, 123) %}", []tt{
		{ID: TagName, Value: "foo"},
		{ID: TagContents, Value: "bar baz(a, b, 123)"},
	})
	testScannerSuccess(t, "foo{%bar%}baz", []tt{
		{ID: Text, Value: "foo"},
		{ID: TagName, Value: "bar"},
		{ID: TagContents, Value: ""},
		{ID: Text, Value: "baz"},
	})
	testScannerSuccess(t, "{{{%\n\r\tfoo bar\n\rbaz%%\n   \r %}}", []tt{
		{ID: Text, Value: "{{"},
		{ID: TagName, Value: "foo"},
		{ID: TagContents, Value: "bar\n\rbaz%%"},
		{ID: Text, Value: "}"},
	})
	testScannerSuccess(t, "{%%}", []tt{
		{ID: TagName, Value: ""},
		{ID: TagContents, Value: ""},
	})
	testScannerSuccess(t, "{%%aaa bb%}", []tt{
		{ID: TagName, Value: ""},
		{ID: TagContents, Value: "%aaa bb"},
	})
}

func TestScannerFailure(t *testing.T) {
	testScannerFailure(t, "a{%")
	testScannerFailure(t, "a{%foo")
	testScannerFailure(t, "a{%% }foo")
	testScannerFailure(t, "a{% foo %")
	testScannerFailure(t, "b{% fo() %}bar")
	testScannerFailure(t, "aa{% foo bar")
}

func testScannerFailure(t *testing.T, str string) {
	r := bytes.NewBufferString(str)
	s := NewScanner(r)
	for s.Next() {
	}
	if err := s.LastError(); err == nil {
		t.Fatalf("expecting error when scanning %q", str)
	}
}

func testScannerSuccess(t *testing.T, str string, expectedTokens []tt) {
	r := bytes.NewBufferString(str)
	s := NewScanner(r)
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
