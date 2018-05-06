package utils

import (
    "strconv"
	"strings"
	"fmt"
	"runtime/debug"
)

func ToString(a []int, delim string) string {
    return strings.Trim(strings.Replace(fmt.Sprint(a), " ", delim, -1), "[]")
}

// clone the slice - veird? do not ask, this is golang thing
func CloneSlice(a []int) []int{
	return append([]int(nil), a...)	
}

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

func MakeRange(start, count int) []int {
    a := make([]int, count)
    for i := range a {
        a[i] = start + i
    }
    return a
}

func Max(x, y int) int {
    if x > y {
        return x
    }
    return y
}

func PrintStack() {
	debug.PrintStack()
}

func AtoPid(s string) (int, bool) {
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

func AtoIpPort(s string) (int, bool) {
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
