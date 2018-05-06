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
}
type Knocks struct {
	mutex sync.Mutex
	state map[int]*KnockingState
}

var knocks Knocks
 
func (knocks *Knocks) addKnock(pid int, port int) {
	const timeout = time.Duration(2) //s
	expirationTime := time.Now().UTC().Add(time.Second*timeout)
	
	knocks.mutex.Lock()
	defer knocks.mutex.Unlock()
	state, ok := knocks.state[pid]
	if !ok {
		state = &KnockingState{ []int{}, expirationTime } 
		knocks.state[pid] = state 
	} 
	state.ports = append(state.ports, port)
	state.expirationTime = expirationTime
	fmt.Println("Collected for", pid, state.ports)
}

func getPortsToBind() []int{
	portsBase := *flag.Int("port_base", 21380, "Base port number")
	portsRangeSize := *flag.Int("port_range", 10, "Size of the ports range")
	ports := []int{}
	for i := 0;i < portsRangeSize; i += 1 {
		ports = append(ports, portsBase+i)
	}  	
	return ports
}

func bindPorts(ports []int) []net.Listener{
	listeners := []net.Listener{}
	failedToBind := []int{}
	for _, port := range ports {
		name := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", name)
		if err == nil {
			listeners = append(listeners, listener)
		} else {
			failedToBind = append(failedToBind, port)
		}
	}
	if len(failedToBind) != 0 {
		fmt.Printf("Failed to bind ports %v\n", failedToBind)		
	}
	fmt.Println("Bound", ports)
	return listeners	
}

// I am looking for line like 
// "tcp6       0      0 :::21380                :::*                    LISTEN      27581/service"
// In the output of the 'netstat'
func getPid(port int) (pid int, ok bool) {
	command := exec.Command("netstat", "-npl")
	var out bytes.Buffer
	command.Stdout = &out
	err := command.Run()
	if err == nil {
		output := strings.Split(out.String(), "\n")
		re := regexp.MustCompile(fmt.Sprintf("tcp.+:::%d.+LISTEN\\s+([0-9]+)/\\S+", port))
		for _, line := range output {
			 match := re.FindStringSubmatch(line)
			 if len(match) > 0 {
			 	pid, ok := utils.AtoPid(match[0])
			 	return pid, ok
			 }
		} 
		fmt.Println("Failed to match port", port, "in", output)
		return 0, false		 	
	} else {
		fmt.Println("Failed to start nestat:", err)
		return 0, false		 	
	}
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
		fmt.Println("New connection found!")
		remoteAddress := connection.RemoteAddr()
		port := remoteAddress.(*net.TCPAddr).Port
		fmt.Println("Port", port)
		pid, ok := getPid(port)
		if ok {
			knocks.addKnock(pid, port)
		} else {
			fmt.Println("Failed to recover pid for port", port)			
		}
	}
}

func main() {
	ports := getPortsToBind()
	listeners := bindPorts(ports)
	for _, listener := range listeners {
		go handleAccept(listener)
	}
	for {
		time.Sleep(100)
	}		
}
