package utils

import (
    "strconv"
	"strings"
	"os"
	"fmt"
	"time"
    "math/rand"
	"runtime/debug"
)

// ToString returns conveiniet presentation of a slice of integers
func ToString(a []int, delim string) string {
    return strings.Trim(strings.Replace(fmt.Sprint(a), " ", delim, -1), "[]")
}

// CloneSlice clones the slice - veird? do not ask, this is golang thing
func CloneSlice(a []int) []int{
	return append([]int(nil), a...)	
}

// Compare two slices of integers
// Based on https://stackoverflow.com/questions/15311969/checking-the-equality-of-two-slices
// What about reflect.DeepEqual(a, b)?
func Compare(a, b []int) bool {

    if a == nil && b == nil { 
        return true; 
    }

    if a == nil || b == nil { 
        return false; 
    }

    if len(a) != len(b) {
        return false
    }

    for i := range a {
        if a[i] != b[i] {
            return false
        }
    }

    return true
}

// MakeRange is similar to the Python's range - creates a slice of integers  
func MakeRange(start, count int) []int {
    a := make([]int, count)
    for i := range a {
        a[i] = start + i
    }
    return a
}

// Max returns the maximum of two arguments
// Golang is capable of comparing only floating point numbers
// See https://stackoverflow.com/questions/27516387/what-is-the-correct-way-to-find-the-min-between-two-integers-in-go
// https://mrekucci.blogspot.co.il/2015/07/dont-abuse-mathmax-mathmin.html
func Max(x, y int) int {
    if x > y {
        return x
    }
    return y
}

// PrintStack helps to get rid of import runtime/debug everywhere
func PrintStack() {
	debug.PrintStack()
}

// AtoPID converts a string to process ID
func AtoPID(s string) (int, bool) {
	pid, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}	
	if pid <= 0 {
		return 0, false
	}		
	// Linux PID is 20 bits?
	if pid >= 0xFFFFFF {
		return 0, false
	}		
	return pid, true
}

// AtoIPPort converts a string to TCP port
func AtoIPPort(s string) (int, bool) {
	port,  err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	if port < 0 {
		return 0, false
	}		
	if port >= 0xFFFF {
		return 0, false
	}		
	return port, true
}

// PathExists returns true if the path exists
func PathExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

// GetPidFilename returns /tmp/knock_PID
func GetPidFilename(pid int) string {
	pidFilename := fmt.Sprintf("/tmp/knock_%d", pid)  
	return pidFilename
}

// Contains returns true if the slice contains the specified value  
// No "contains" method in Golang (rolling my eyes again)
// https://stackoverflow.com/questions/10485743/contains-method-for-a-slice
func Contains(s []int, e int) bool {
    for _, a := range s {
        if a == e {
            return true
        }
    }
    return false
}

// GetTupleSize returns number of ports in a tuple give the ports range size
func GetTupleSize(portsRangeSize int) int {
	return portsRangeSize/2
}


// GetTuplesCount returns number of tuples to send given the tuple size and 
// required tolerance
func GetTuplesCount(tolerance int, tupleSize int) int {
	tuplesCount := tolerance * tupleSize/100 + 2
	if tolerance == 0 {
		tuplesCount = 1		
	} 
	return tuplesCount
}

// InitRand calls to math.rand.Seed()
func InitRand() {
	rand.Seed((int64)(time.Now().UnixNano()))	
}

// RemoveElementFromSlice removes the specified element from the slice
func RemoveElementFromSlice(a []int, index int) []int {
	return append(a[:index], a[index+1:]...)	
}


// Reflection based loop which prints all fields in the statistics
func StatisticsPrintf(data interface{}, columns int, format string) string {
	t := reflect.TypeOf(data)
	v := reflect.ValueOf(data)
	if format == "" {
		format = "%-24s %9d "
	}
	var buffer bytes.Buffer
	for i := 0; i < v.NumField(); i++ {
		if i%columns == 0 {
			buffer.WriteString("\n")
		}
		buffer.WriteString(fmt.Sprintf(format, t.Field(i).Name, v.Field(i)))
	}

	return buffer.String()
}


// This accumulator is fast, but not thread safe. Race when
// calling Tick() and Add() and between calls to Add() produces not reliable result
// Use InitSync(), TickSync() and AddSync() if thread safety is desired
type Accumulator struct {
	counters []accumulatorCounter
	cursor   uint64
	size     uint64
	count    uint64
	mutex    *sync.Mutex
}

type AccumulatorResult struct {
	Nonzero   bool
	MaxWindow uint64
	Max       uint64
	Results   []uint64
}

func (a *Accumulator) Init(size uint64) {
	a.counters = make([]accumulatorCounter, size)
	a.size = size
	a.count = 0
	a.Reset()
}

func (a *Accumulator) InitSync(size uint64) {
	a.Init(size)
	a.mutex = &sync.Mutex{}
}

func (a *Accumulator) Reset() {
	a.cursor = 0
	a.count = 0
	// Probably faster than call to make()
	for i := uint64(0); i < a.size; i++ {
		a.counters[i].summ = 0
		a.counters[i].updates = 0
	}
}

func (a *Accumulator) Size() uint64 {
	return a.size
}

func (a *Accumulator) incCursor(cursor uint64) uint64 {
	if cursor >= (a.size - 1) {
		return 0
	} else {
		return (cursor + 1)
	}
}

// Return the results - averages over the window of "size" entries
// Use "divider" to normalize the output in the same copy path
func (a *Accumulator) GetAverage(divider uint64) AccumulatorResult {
	return a.getResult(divider, true)
}

// Use "divider" to normalize the output in the same copy path
func (a *Accumulator) GetSumm(divider uint64) AccumulatorResult {
	return a.getResult(divider, false)
}

func (a *Accumulator) GetSummSync(divider uint64) AccumulatorResult {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.getResult(divider, false)
}

// Use "divider" to normalize the output in the same copy path
func (a *Accumulator) getResult(divider uint64, average bool) AccumulatorResult {
	var nonzero bool = false
	if divider == 0 {
		divider = 1
	}
	size := a.size
	var cursor uint64 = a.cursor
	if size > a.count {
		size = a.count
		cursor = a.size
	}
	results := make([]uint64, size)
	max := uint64(0)
	maxWindow := uint64(0)
	for i := uint64(0); i < size; i++ {
		cursor = a.incCursor(cursor)
		updates := a.counters[cursor].updates
		if updates > 0 {
			nonzero = true
			summ := a.counters[cursor].summ
			if maxWindow < summ {
				maxWindow = summ
			}
			var result uint64
			if average {
				result = (summ / (divider * updates))
			} else {
				result = (summ / divider)
			}
			if max < result {
				max = result
			}
			results[i] = result
		} else {
			results[i] = 0
		}
	}
	return AccumulatorResult{
		Results:   results,
		Nonzero:   nonzero,
		Max:       max,
		MaxWindow: maxWindow,
	}
}

func (a *Accumulator) Add(value uint64) {
	cursor := a.cursor
	a.counters[cursor].summ += value
	a.counters[cursor].updates += 1
}

func (a *Accumulator) AddSync(value uint64) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.Add(value)
}

func (a *Accumulator) Tick() {
	cursor := a.incCursor(a.cursor)
	a.cursor = cursor
	a.counters[cursor].summ = 0
	a.counters[cursor].updates = 0
	a.count += 1
}

func (a *Accumulator) TickSync() {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.Tick()
}

type CyclicBuffer struct {
	data  []interface{}
	full  bool
	size  int
	index int
	mutex *sync.Mutex
}

func (cb *CyclicBuffer) Init(size int) {
	cb.mutex = &sync.Mutex{}
	cb.data = make([]interface{}, size)
	cb.index = 0
	cb.full = false
	cb.size = size
}

func (cb *CyclicBuffer) Append(d interface{}) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	var index = cb.index
	cb.data[index] = d
	index += 1
	if index >= cb.size {
		index = 0
		cb.full = true
	}
	cb.index = index
}

type CyclicBufferIterator struct {
	index int
	count int
	cb    *CyclicBuffer
}

func CreateCyclicBufferIterator(cb *CyclicBuffer) *CyclicBufferIterator {
	var it CyclicBufferIterator
	it.cb = cb
	if cb.full {
		it.index = cb.index
		it.count = cb.size
	} else {
		it.index = 0
		it.count = cb.index
	}
	return &it
}

func (it *CyclicBufferIterator) Value() interface{} {
	value := it.cb.data[it.index]
	it.index += 1
	if it.index >= it.cb.size {
		it.index = 0
	}
	it.count -= 1
	return value
}

func (it *CyclicBufferIterator) Next() bool {
	return (it.count > 0)
}

func (cb *CyclicBuffer) Get() []interface{} {
	var index int
	var count int
	if cb.full {
		index = cb.index
		count = cb.size
	} else {
		index = 0
		count = cb.index
	}
	res := make([]interface{}, 0, count)
	for i := 0; i < count; i++ {
		d := cb.data[index]
		res = append(res, d)
		index += 1
		if index >= cb.size {
			index = 0
		}
	}
	return res
}
