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
	"net/http"
	"net/url"
	"fmt"
	"flag"
	"time"
	"regexp"
	"os/exec"
	"bytes"
	"strings"
	"sync"
	"io/ioutil"
	"port-knocking-ipc/utils"
)

type knockingState struct {
	ports []int
	expirationTime time.Time
	pid int
}
type knocks struct {
	mutex sync.Mutex
	state map[int]*knockingState
	portsBase        int
	portsRange      []int
	failedToBind    []int
	listeners       []net.Listener
	boundPorts      []int
	portsRangeSize  int
	tolerance       int
	tupleSize       int
	host            string
	port            int
	hostURL         string
}

var knocksCollection = knocks{state: make(map[int]*knockingState),
	portsBase : *flag.Int("port_base", 21380, "Base port number"),
	portsRangeSize : *flag.Int("port_range", 10, "Size of the ports range"),
	tolerance : *flag.Int("tolerance", 20, "Percent of tolerance for port bind failures"),
	host : *flag.String("host", "127.0.0.1", "Server name"),
	port : *flag.Int("port", 8080, "Server port"),
}

// Add the port to the map of knocking sequences 
func (k *knocks) addKnock(pid int, port int) *knockingState{
	const timeout = time.Duration(5) //s
	expirationTime := time.Now().UTC().Add(time.Second*timeout)
	
	state, ok := k.state[pid]
	if !ok {
		state = &knockingState{ []int{}, expirationTime, int(pid) } 
		k.state[pid] = state 
	}
	state.ports = append(state.ports, port)
	state.expirationTime = expirationTime
	return state
}

// get list of ports to bind
func (k *knocks) getPortsToBind() []int{
	ports := []int{}
	for i := 0;i < k.portsRangeSize; i++ {
		ports = append(ports, k.portsBase+i)
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
	fmt.Println("Listening on", ports)
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
		fmt.Println("Failed to match port", port)
		return 0, false		 	
	} 
	fmt.Println("Failed to start nestat:", err)
	return 0, false		 	
}

// Return true if all tuples are collected or timeout
func (k *knocks) isCompleted(state *knockingState) bool {
	if state.expirationTime.Before(time.Now().UTC()) {
		return true 
	}
	// I want to allocate enough tuples to reach the specifed tolerance level
	tuples := utils.GetTuplesCount(k.tolerance, k.tupleSize)
	
	if k.tolerance == 0 {
		tuples = 1		
	}  
	if len(state.ports) % (tuples * k.tupleSize) == 0{
		return true
	}
	return false
}

// Send "/session?ports=...&pid=..." to the server
// I have to divide the collected ports by tuples of size knocks.tupleSize -ports in a tuple are ascending
// If a tuple is not full I check if there are ports which I failed to bind which 
// fall in the tuple's range and create all possible port tuples - combinations of collected ports and
// ports I failed to bind 
func (k *knocks) sendQueryToServer(pid int, ports []int) {
	var text bytes.Buffer
	text.WriteString(k.hostURL) 
	text.WriteString("/session?ports=")
	for _, port := range ports {
		text.WriteString(fmt.Sprintf("%d,", port))
	}
	text.WriteString("&pid=")
	text.WriteString(fmt.Sprintf("%d", pid))
	
	urlQuery := text.String()
	response, err := http.Get(urlQuery)
	if err == nil {
		defer response.Body.Close()
		text, err := ioutil.ReadAll(response.Body)
		if err == nil {
			fmt.Printf("Got repsonse for ulr='%s': %s\n", urlQuery, string(text))
		}		
	} else {
		fmt.Println("Failed to GET", urlQuery)		
	}	
}

// Goroutine which periodically checks if any knocking sequences completed
func (k *knocks) completeKnocks() {
	k.mutex.Lock()	
	completedKnocks := []*knockingState{}
	for _, state := range k.state {
		if k.isCompleted(state) {
			completedKnocks = append(completedKnocks, state)
		}
	}
	for _, state := range completedKnocks {
		delete(k.state, state.pid)
		k.sendQueryToServer(state.pid, state.ports)
	}
	k.mutex.Unlock()
	time.Sleep(1 * time.Second)
}


// Goroutine to accept incoming connection
func (k *knocks) handleAccept(listener net.Listener) {
	localPort := listener.Addr().(*net.TCPAddr).Port
	defer listener.Close()	
	for {
		connection, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept failed", err)
			continue
		}
		remoteAddress := connection.RemoteAddr()
		// Based on https://groups.google.com/forum/#!topic/golang-nuts/JLzchxXm5Vs
		// See also https://golang.org/ref/spec#Type_assertions
		port := remoteAddress.(*net.TCPAddr).Port
		pid, ok := getPid(port)
		connection.Close()
		if ok {			
			k.mutex.Lock()
			//fmt.Printf("New connection localPort=%d, remotePort=%d, pid=%d\n", localPort, port, pid)
			state := k.addKnock(pid, localPort)
			if k.isCompleted(state) {
				//fmt.Printf("Completed pid=%d\n", pid)
				delete(k.state, state.pid)
				k.sendQueryToServer(state.pid, state.ports)
			}
			k.mutex.Unlock()
		} else {
			fmt.Println("Failed to recover pid for port", port)			
		}
		//fmt.Println("Done port", port)			
	}
}

func main() {
	knocksCollection.tupleSize = utils.GetTupleSize(knocksCollection.portsRangeSize)
	ports := knocksCollection.getPortsToBind()
	knocksCollection.listeners, knocksCollection.boundPorts, knocksCollection.failedToBind = bindPorts(ports)
	url := &url.URL{
		Scheme:   "http",
		Host:     fmt.Sprintf("%s:%d", knocksCollection.host, knocksCollection.port),
	}
	knocksCollection.hostURL = url.String()  
	for _, listener := range knocksCollection.listeners {
		go knocksCollection.handleAccept(listener)
	}
	
	// Start a background thread to handle timeout expiration 
	// of knock sequences
	go knocksCollection.completeKnocks()
	
	// Block the main thread, TODO turn to daemon
	for {
		time.Sleep(100 * time.Millisecond)
	}		
}
