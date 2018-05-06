// Get the ports range from the command line, list of ports to skip 
// for simulaiton of failure of bind
// Bind the specified ports
// Wait for TCP connections from a client
// Accept connection, collect the port number and the client PID
// When the required number of ports are knocked or a timeout expired send  
// all possible combinations of the collected port knocks and ports the service 
// failed to bind to the server
 
package main

import (
	"net"
	"fmt"
	"flag"
//	"os"
//	"port-knocking-ipc/utils"
)

func getPortsToBind() []int{
	portsBase := *flag.Int("port_base", 21380, "Base port number")
	portsRangeSize := *flag.Int("port_range", 10, "Size of the ports range")
	ports := []int{}
	for port := portsBase;port < (portsBase+portsRangeSize); port += 1 {
		ports = append(ports, port)
	}  	
	return ports
}

func bindPorts(ports []int) []net.Listener{
	listeners := []net.Listener{}
	failedToBind := []int{}
	for port := range ports {
		name := fmt.Sprintf("localhost:%d", port)
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
	return listeners	
}

func closeListeners(listeners []net.Listener) {
	for _, listener := range listeners {
		listener.Close()
	}
}

func main() {
	ports := getPortsToBind()
	listeners := bindPorts(ports)
	defer closeListeners(listeners)	
}