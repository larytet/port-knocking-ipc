package main

import (
	"testing"
	"port-knocking-ipc/utils"
)


func TestGetPorts(t *testing.T) {
	tuples := getPorts("0,1,2,3\n4,5,6,7\n")
	expectedTuples := [][]int{
		{0,1,2,3},
		{4,5,6,7},
	}
	if len(expectedTuples) != len(tuples) {
		t.Errorf("Got %v expected %v\n", tuples, expectedTuples)		
	} else {
		for i := 0;i < len(expectedTuples);i++ {
			if !utils.Compare(tuples[i], expectedTuples[i]) {
				t.Errorf("Got %v expected %v\n", tuples, expectedTuples)
			}			
		}
	}
}

