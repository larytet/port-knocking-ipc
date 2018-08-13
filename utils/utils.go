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
