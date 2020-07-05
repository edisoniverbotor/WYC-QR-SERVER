package main

// MIT Licensed - see LICENSE

import "testing"

func TestHexEncoer(t *testing.T) {
	s := HexEscapeNonASCII("abc/你好")
	if s != "abc/%e4%bd%a0%e5%a5%bd" {
		t.Errorf("Oops Expected x got %s\n", s)
	}
}

/* vim: set noai ts=4 sw=4: */
