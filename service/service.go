// Get the ports range from the command line, list of ports to skip 
// for simulaiton of failure of bind
// Bind the specified ports
// Wait for TCP connections from a client
// Accept connection, collect the port number, figure out the client PID
// When the required number of ports are knocked or a timeout expired send  
// all possible combinations of the collected port knocks and the ports 
// the service failed to bind to the server
 
package main

import (
	"net"
	"fmt"
	"flag"
	"time"
	"regexp"
	"os/exec"
	"bytes"
	"strings"
	"sync"
	"port-knocking-ipc/utils"
)

type KnockingState struct {
	ports []int
	expirationTime time.Time
	pid int
}
type Knocks struct {
	mutex sync.Mutex
	state map[int]*KnockingState
	portsBase        int
	portsRange      []int
	failedToBind    []int
	listeners       []net.Listener
	boundPorts      []int
	portsRangeSize  int
	tolerance       int
	tupleSize       int
}

var knocks = Knocks{state: make(map[int]*KnockingState),
	portsBase : *flag.Int("port_base", 21380, "Base port number"),
	portsRangeSize : *flag.Int("port_range", 10, "Size of the ports range"),
	tolerance : *flag.Int("tolerance", 20, "Percent of tolerance for port bind failures"),
	tupleSize : *flag.Int("port_range", 10, "Size of the ports range")/2,
}

// Add the port to the map of knocking sequences 
func (knocks *Knocks) addKnock(pid int, port int) *KnockingState{
	const timeout = time.Duration(5) //s
	expirationTime := time.Now().UTC().Add(time.Second*timeout)
	
	knocks.mutex.Lock()
	defer knocks.mutex.Unlock()
	state, ok := knocks.state[pid]
	if !ok {
		state = &KnockingState{ []int{}, expirationTime, int(pid) } 
		knocks.state[pid] = state 
	}
	state.ports = append(state.ports, port)
	state.expirationTime = expirationTime
	fmt.Println("Collected for", pid, state.ports)
	return state
}

// get list of ports to bind
func getPortsToBind() []int{
	ports := []int{}
	for i := 0;i < knocks.portsRangeSize; i += 1 {
		ports = append(ports, knocks.portsBase+i)
	}  	
	return ports
}

// bind the specified ports 
func bindPorts(ports []int) (listeners []net.Listener, boundPorts []int, failedToBind []int) {
	listeners = []net.Listener{}
	failedToBind = []int{}
	boundPorts = []int{}
	for _, port := range ports {
		name := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", name)
		if err == nil {
			listeners = append(listeners, listener)
			boundPorts = append(boundPorts, port)
		} else {
			failedToBind = append(failedToBind, port)
		}
	}
	if len(failedToBind) != 0 {
		fmt.Printf("Failed to bind ports %v\n", failedToBind)		
	}
	fmt.Println("Bound", ports)
	return listeners, boundPorts, failedToBind	
}

// I am looking for line like 
// "tcp        0      0 127.0.0.1:36518         127.0.0.1:21380         ESTABLISHED 26396/firefox  "
// In the output of the 'netstat'
func getPid(port int) (pid int, ok bool) {
	command := exec.Command("netstat", "-ntp")
	var out bytes.Buffer
	command.Stdout = &out
	err := command.Run()
	if err == nil {
		output := strings.Split(out.String(), "\n")
		re := regexp.MustCompile(fmt.Sprintf("tcp\\s+\\S+\\s+\\S+\\s+\\S+:%d.+ESTABLISHED\\s+([0-9]+)/(\\S+)", port))
		for _, line := range output {
			 match := re.FindStringSubmatch(line)
			 if len(match) > 0 {
			 	pid, ok := utils.AtoPid(match[1])
			 	return pid, ok
			 }
		} 
		fmt.Println("Failed to match port", port, out.String())
		return 0, false		 	
	} else {
		fmt.Println("Failed to start nestat:", err)
		return 0, false		 	
	}
}

// Return true if all tuples are collected or timeout
func isCompleted(state *KnockingState) bool {
	if state.expirationTime.Before(time.Now().UTC()) {
		return true 
	}
	// I want to allocate enough tuples to reach the specifed tolerance level
	tuples := (knocks.tolerance * knocks.tupleSize)/100 + 2
	if knocks.tolerance == 0 {
		tuples = 1		
	}  
	if len(state.ports) % (tuples * knocks.tupleSize) == 0{
		return true
	}
	return false
}

// Send "/session?ports=...&pid=..." to the server
// I have to divide the collected ports by tuples of size  
func sendQueryToServer(pid int, ports []int) {
}

// Goroutine which periodically checks if any knocking sequences completed
func completeKnocks() {
	knocks.mutex.Lock()
	defer knocks.mutex.Unlock()
	
	completedKnocks := []*KnockingState{}
	for _, state := range knocks.state {
		if isCompleted(state) {
			completedKnocks = append(completedKnocks, state)
		}
	}
	for _, state := range completedKnocks {
		delete(knocks.state, state.pid)
		sendQueryToServer(state.pid, state.ports)
	}
	time.Sleep(1 * time.Second)
}


// Goroutine to accept incoming connection
func handleAccept(listener net.Listener) {
	defer listener.Close()	
	for {
		connection, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept failed", err)
		}
		defer connection.Close()
		remoteAddress := connection.RemoteAddr()
		// Based on https://groups.google.com/forum/#!topic/golang-nuts/JLzchxXm5Vs
		// See also https://golang.org/ref/spec#Type_assertions
		port := remoteAddress.(*net.TCPAddr).Port
		fmt.Println("New connection port", port)
		pid, ok := getPid(port)
		if ok {			
			state := knocks.addKnock(pid, port)
			if isCompleted(state) {
				delete(knocks.state, state.pid)
				sendQueryToServer(state.pid, state.ports)
			}
		} else {
			fmt.Println("Failed to recover pid for port", port)			
		}
	}
}

func main() {
	ports := getPortsToBind()
	knocks.listeners, knocks.boundPorts, knocks.failedToBind = bindPorts(ports)
	for _, listener := range knocks.listeners {
		go handleAccept(listener)
	}
	go completeKnocks()
	for {
		time.Sleep(100 * time.Millisecond)
	}		
}
