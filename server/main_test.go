package main

import (
	"testing"
	"port-knocking-ipc/utils/combinations"
	"port-knocking-ipc/utils"
)


func TestGenerator(t *testing.T) {
	var generator = combinations.Init(([]int{0,1,2,3})[:], 2)
	var tuples = getPortsCombinations(&generator, 2)
	var text = tuplesToText(tuples)
	var expectedText = "0,1\n0,2\n"
	if text != expectedText {
		t.Errorf("Got '%s' expected '%s'\n", text, expectedText)
	}	
}

func TestKeyTupleConverter(t *testing.T) {
	base := uint64(1)
	expectedTuple := []int{base+1,base+2,base+3,base+4,base+5,base+6,base+7,base+8}
	expectedKey := uint64(0x0102030405060708)
	key := tupleToKey(base, expectedTuple)
	if key != expectedKey {
		t.Errorf("Got '%x' expected '%x'\n", key, expectedKey)
	}		
	tuple := keyToTuple(base, key)
	if !utils.Compare(tuple, expectedTuple) {
		t.Errorf("Got '%v' expected '%v'\n", tuple, expectedTuple)
	}		
}

