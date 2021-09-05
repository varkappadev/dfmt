package main

import (
	"math"
	"testing"
)

func indentStringTest(t *testing.T, pretty bool, count int, minlen int, maxlen int) {
	indent := createIndentString(pretty, count)
	if len(indent) < minlen {
		t.Errorf("indent string '%s' is too short (< %d)", indent, minlen)
	}
	if len(indent) > maxlen {
		t.Errorf("indent string '%s' is too long (> %d)", indent, maxlen)
	}
}

func TestIndentStrings(t *testing.T) {
	indentStringTest(t, true, 2, 1, math.MaxInt32)
	indentStringTest(t, true, 0, 1, math.MaxInt32)
	indentStringTest(t, true, -1, 1, math.MaxInt32)
	indentStringTest(t, false, -1, 0, 0)
	indentStringTest(t, false, 2, 0, 0)
}

func TestCliBuilder(t *testing.T) {
	app := configureApp()
	if app == nil {
		t.Errorf("failed to create application cli")
	} else {
		app.PrintLongHelp()
	}
}
