package calsync

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestFallback(t *testing.T) {
	d := parseDescription("foo")
	equals(t, "", d.prefix)
	equals(t, "foo", d.suffix)
}

func TestCommentPreservation(t *testing.T) {
	d := &description{
		"testprefix",
		"testsuffix",
	}
	s := d.String()
	equals(t, "testprefix\n"+delim+"\ntestsuffix", s)
	d = parseDescription(s)
	equals(t, "testprefix", d.prefix)
	equals(t, "testsuffix", d.suffix)
}

// equals fails the test if exp is not equal to act.
func equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}
