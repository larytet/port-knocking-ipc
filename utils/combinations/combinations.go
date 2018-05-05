// Based on http://rosettacode.org/wiki/Combinations#C.23

package combinations

import (
    "sync"
	"port-knocking-ipc/utils"
)

type Stack []int

func (s Stack) Push(v int) Stack {
    return append(s, v)
}

func (s Stack) Pop() (Stack, int) {
    l := len(s)
    return s[:l-1], s[l-1]
}

type State struct {
	source []int
	m int  
	n int
	mutex sync.Mutex
	s Stack
	result Stack
}

// Initialize the generator of combinations of size m 
// from the array 'source'
func Init(source []int, m int) State {
	var state State
	state.source = source
	state.m, state.n = m, len(source)
	state.reset()
	return state
}

// Return next unique combination of integers
func (state *State) Next() []int {
	state.mutex.Lock()

	result := state.next()
	
	// Save CPU cycles by not usingn defer
	state.mutex.Unlock()
	return result	
}

// Return next unique combination of integers
// Wrap around 
func (state *State) NextWrap() []int {
	state.mutex.Lock()

	result := state.next()
	if result == nil {
		state.reset()	
		result = state.next()
	}
	 	
	// Save CPU cycles by not usingn defer
	state.mutex.Unlock()
	return result
}

func (state *State) next() []int {	
	for len(state.s) > 0 {
		i := len(state.s) - 1
		var j int
		state.s, j = state.s.Pop()
		for j < state.n {
			var value = state.source[j]
			j ++
			state.s = state.s.Push(j)
			state.result[i] = value
			i ++
			if i == state.m {
				// I can save memory allocation here and return the same
				// array again and again
				return utils.CloneSlice(state.result)
			}
		}
	}
	return nil	
}

func (state *State) reset() {
	s := make(Stack, 0)
	s = s.Push(0) // start from index 0
	state.s = s
	state.result = make(Stack, state.m)
}
