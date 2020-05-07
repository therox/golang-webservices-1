package main

import (
	"bytes"
	"strings"
	"testing"
)

var testOk = `1
2
3
3
4
5`

var testOkResult = `1
2
3
4
5
`

func Test_uniq(t *testing.T) {
	in := strings.NewReader(testOk)
	out := new(bytes.Buffer)
	err := uniq(in, out)
	if err != nil {
		t.Errorf("test for uniq failed (want no error, got error %v)", err)
	}
	if out.String() != testOkResult {
		t.Errorf("test for uniq failed - result not matched. Got %q, want %q", out.String(), testOkResult)
	}
}

var testFail = `
1
2
1`

func Test_uniq_unsorted(t *testing.T) {
	in := strings.NewReader(testFail)
	out := new(bytes.Buffer)
	err := uniq(in, out)
	if err == nil {
		t.Errorf("test for uniq fail failed (want error, got no error)")
	}
}
