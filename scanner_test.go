package quicktemplate

import (
	"bytes"
	"reflect"
	"testing"
)

func TestScannerPlainSuccess(t *testing.T) {
	testScannerSuccess(t, "{%plain%}{%endplain%}", nil)
	testScannerSuccess(t, "{%plain%}{%foo bar%}asdf{%endplain%}", []tt{
		{ID: Text, Value: "{%foo bar%}asdf"},
	})
	testScannerSuccess(t, "{%plain%}{%foo{%endplain%}", []tt{
		{ID: Text, Value: "{%foo"},
	})
	testScannerSuccess(t, "aa{%plain%}bbb{%cc%}{%endplain%}{%plain%}dsff{%endplain%}", []tt{
		{ID: Text, Value: "aa"},
		{ID: Text, Value: "bbb{%cc%}"},
		{ID: Text, Value: "dsff"},
	})
	testScannerSuccess(t, "mmm{%plain%}aa{% bar {%%% }baz{%endplain%}nnn", []tt{
		{ID: Text, Value: "mmm"},
		{ID: Text, Value: "aa{% bar {%%% }baz"},
		{ID: Text, Value: "nnn"},
	})
	testScannerSuccess(t, "{% plain dsd %}0{%comment%}123{%endcomment%}45{% endplain aaa %}", []tt{
		{ID: Text, Value: "0{%comment%}123{%endcomment%}45"},
	})
}

func TestScannerPlainFailure(t *testing.T) {
	testScannerFailure(t, "{%plain%}sdfds")
	testScannerFailure(t, "{%plain%}aaaa%{%endplain")
	testScannerFailure(t, "{%plain%}{%endplain%")
}

func TestScannerCommentSuccess(t *testing.T) {
	testScannerSuccess(t, "{%comment%}{%endcomment%}", nil)
	testScannerSuccess(t, "{%comment%}foo{%endcomment%}", nil)
	testScannerSuccess(t, "{%comment%}foo{%endcomment%}{%comment%}sss{%endcomment%}", nil)
	testScannerSuccess(t, "{%comment%}foo{%bar%}{%endcomment%}", nil)
	testScannerSuccess(t, "{%comment%}foo{%bar {%endcomment%}", nil)
	testScannerSuccess(t, "{%comment%}foo{%bar&^{%endcomment%}", nil)
	testScannerSuccess(t, "{%comment%}foo{% bar\n\rs%{%endcomment%}", nil)
	testScannerSuccess(t, "xx{%x%}www{% comment aux data %}aaa{% comment %}{% endcomment %}yy", []tt{
		{ID: Text, Value: "xx"},
		{ID: TagName, Value: "x"},
		{ID: TagContents, Value: ""},
		{ID: Text, Value: "www"},
		{ID: Text, Value: "yy"},
	})
}

func TestScannerCommentFailure(t *testing.T) {
	testScannerFailure(t, "{%comment%}...no endcomment")
	testScannerFailure(t, "{% comment %}foobar{% endcomment")
}

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
	testScannerSuccess(t, "foo{% bar %}{% baz aa (123)%}321", []tt{
		{ID: Text, Value: "foo"},
		{ID: TagName, Value: "bar"},
		{ID: TagContents, Value: ""},
		{ID: TagName, Value: "baz"},
		{ID: TagContents, Value: "aa (123)"},
		{ID: Text, Value: "321"},
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
	var tokens []tt
	for s.Next() {
		tokens = append(tokens, tt{
			ID:    s.Token().ID,
			Value: string(s.Token().Value),
		})
	}
	if err := s.LastError(); err == nil {
		t.Fatalf("expecting error when scanning %q. got tokens %v", str, tokens)
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
