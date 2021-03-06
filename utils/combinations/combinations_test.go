package combinations

import (
	"testing"
	"port-knocking-ipc/utils"
)

func run_test(t *testing.T, data []int, size int, expectedResults [][]int) {
	var state = Init(data[:], size)
	var combination Stack
	var expectedResultIndex int = 0
	for {
		combination = state.Next()
		if combination != nil {
			var expectedValue = (expectedResults[expectedResultIndex])[:]
			if !utils.Compare(expectedValue, combination) {
				t.Errorf("%d got=[%s] expected=[%s]\n", 
					expectedResultIndex, 
					utils.ToString(combination, ", "), 
					utils.ToString(expectedValue, ", "))
				t.Fail()
			}
		} else {
			if expectedResultIndex != len(expectedResults) {
				t.Errorf("too few results %d instead of %d\n", expectedResultIndex, len(expectedResults))
				t.Fail()
			}
			break
		}
		expectedResultIndex ++
	}	
}

type CombinationsTestSets struct {
	data []int
	size int
	result [][]int
}

func TestMain(t *testing.T) {
	testSets := []CombinationsTestSets {
		{[]int{0,1,2,3}, 2, [][]int{
			{0, 1}, 
			{0, 2}, 
			{0, 3}, 
			{1, 2}, 
			{1, 3}, 
			{2, 3},
		}},
		{[]int{0,1,1}, 2, [][]int{
			{0, 1}, 
			{0, 1}, 
			{1, 1},
		}},
	}
	for _, testSet := range testSets {
		run_test(t, testSet.data, testSet.size, testSet.result)
	}
}

