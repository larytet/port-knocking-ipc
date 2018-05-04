package main

import (
	"testing"
	"port-knocking-ipc/utils/combinations"
)


func TestMain(t *testing.T) {
	var generator = combinations.Init(([]int{0,1,2,3})[:], 2)
	var tuples = getPortsCombinations(&generator, 2)
	var text = tuplesToText(tuples)
	var expectedText = "0,1\n0,2\n"
	if text != expectedText {
		t.Errorf("Got '%s' expected '%s'\n", text, expectedText)
	}
	
}

