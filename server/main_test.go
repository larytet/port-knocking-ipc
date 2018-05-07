package main

import (
	"testing"
	"port-knocking-ipc/utils/combinations"
	"port-knocking-ipc/utils"
)


func TestGenerator(t *testing.T) {
	var generator = combinations.Init(([]int{0,1,2,3})[:], 2)
	var tuples = getPortsCombinations(&generator, 2, 0)
	var text = tuplesToText(tuples)
	var expectedText = "0,1\n0,2\n"
	if text != expectedText {
		t.Errorf("Got '%s' expected '%s'\n", text, expectedText)
	}	
}

func testKeyTupleConverter(t *testing.T, expectedTuple []int, expectedKey keyID) {
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

type KeyTupleConverterTestSet struct {
	tuple []int
	key keyID
}

func TestKeyTupleConverter(t *testing.T) {
	testSets := []KeyTupleConverterTestSet {
		{[]int{2,3,4,5,6,7,8,9}, keyID(0x0102030405060708)},
		{[]int{1,1,1,1,1,1,1,1}, keyID(0x0000000000000000)},
	}
	
	for _, testSet := range testSets {
		testKeyTupleConverter(t, testSet.tuple, testSet.key)
	}
}

type ParseUrlQuerySessionPortsTestSet struct {
	portsStr []string
	tupleSize int
	ports [][]int
	ok bool
}

// Warning! Test the method which handles URL parsing 
func TestParseUrlQuerySessionPorts(t *testing.T) {
	testSets := []ParseUrlQuerySessionPortsTestSet {
		{[]string{"0,1,2,3,"}, int(4), [][]int{{0,1,2,3,}, }, bool(true)},
		{[]string{""}, int(4), [][]int(nil), bool(false)},
		{[]string{"0,-1,2,3"}, int(4), [][]int(nil), bool(false)},
		{[]string{"0,0x1FFFF,2,3"}, int(4), [][]int(nil), bool(false)},
		{[]string(nil), int(4), [][]int(nil), bool(false)},
	}
	
	for testIndex, testSet := range testSets {
		tuples, ok := parseUrlQuerySessionPorts(testSet.portsStr, testSet.tupleSize)
		if ok != testSet.ok {
			t.Errorf("Got ok '%t' expected '%t' for test %d\n", ok, testSet.ok, testIndex)			
		}
		for i := 0;i < len(tuples);i++ {
			if !utils.Compare(tuples[i], testSet.ports[i]) {
				t.Errorf("Got ports '%v' expected '%v for test %d'\n", tuples, testSet.ports, testIndex)
			}			
		} 
	}
}

type ParseUrlQuerySessionPidSet struct {
	pidStr []string
	pid int
	ok bool
}

// Warning! Test the method which handles URL parsing 
func TestParseUrlQuerySessionPid(t *testing.T) {
	testSets := []ParseUrlQuerySessionPidSet {
		{[]string{"1"}, int(1), bool(true)},
		{[]string{"-1"}, int(0), bool(false)},
		{[]string{"0"}, int(0), bool(false)},
		{[]string{"9999999999999999999999999999999999999999999999999999999999999999999999999"}, int(0), bool(false)},
		{[]string{"a1"}, int(0), bool(false)},
		{[]string{" "}, int(0), bool(false)},
		{[]string{""}, int(0), bool(false)},
		{[]string{"1", "2"}, int(0), bool(false)},
		{[]string(nil), int(0), bool(false)},
	}
	
	for testIndex, testSet := range testSets {
		pid, ok := parseUrlQuerySessionPid(testSet.pidStr)
		if ok != testSet.ok {
			t.Errorf("Got ok '%t' expected '%t' for test %d\n", ok, testSet.ok, testIndex)			
		}
		if pid != testSet.pid {
			t.Errorf("Got pid '%d' expected '%d' for test %d\n", pid, testSet.pid, testIndex)			
		}
	}
}


