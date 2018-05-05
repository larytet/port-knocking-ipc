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

func testKeyTupleConverter(t *testing.T, expectedTuple []int, expectedKey KeyId) {
	base := uint64(1)
	key := tupleToKey(base, expectedTuple)
	if key != expectedKey {
		t.Errorf("Got '%x' expected '%x'\n", key, expectedKey)
	}		
	tuple := keyToTuple(base, key)
	if !utils.Compare(tuple, expectedTuple) {
		t.Errorf("Got '%v' expected '%v'\n", tuple, expectedTuple)
	}		
}

func TestKeyTupleConverter(t *testing.T) {
	expectedTuple := []int{2,3,4,5,6,7,8,9}
	expectedKey := KeyId(0x0102030405060708)
	testKeyTupleConverter(t, expectedTuple, expectedKey)

	expectedTuple = []int{1,1,1,1,1,1,1,1}
	expectedKey = KeyId(0x00)
	testKeyTupleConverter(t, expectedTuple, expectedKey)
}

