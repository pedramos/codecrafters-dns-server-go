package main

import (
	"strconv"
	"testing"
)

func TestUint2Bits(t *testing.T) {
	tmp := uint2Bits(4, 4)
	var b [4]bool
	copy(b[:], tmp)
	if b != [4]bool{false, true, false, false} {
		t.Errorf("four=%s", strconv.FormatInt(4, 2))
		t.Errorf("Expected []bool{false, true, false, false}, got :%#v\n", b)
	}
}
