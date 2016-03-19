package quicktemplate

import (
	"encoding/json"
	"testing"
)

func TestAppendJSONString(t *testing.T) {
	testAppendJSONString(t, ``)
	testAppendJSONString(t, `f`)
	testAppendJSONString(t, `"`)
	testAppendJSONString(t, `<`)
	testAppendJSONString(t, "\x00\n\r\t\b\f"+`"\`)
	testAppendJSONString(t, `"foobar`)
	testAppendJSONString(t, `foobar"`)
	testAppendJSONString(t, `foo "bar"
		baz`)
	testAppendJSONString(t, `this is a "тест"`)
	testAppendJSONString(t, `привет test`)
}

func testAppendJSONString(t *testing.T, s string) {
	expectedResult, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("unexpected error when encoding string %q: %s", s, err)
	}
	result := appendJSONString(nil, s)
	if string(result) != string(expectedResult) {
		t.Fatalf("unexpected result %q. Expecting %q. original string %q", result, expectedResult, s)
	}
}
