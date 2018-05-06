// Send HTTP GET to the server
// Parse the XML reponses, write the ports combination to the file /tmp/PID
// Establish TCP connections with the service using the ports specified in the XML file
// Poll the file /tmp/PID for 10s. If the file is not removed, print error, remove the file

package main

import (
	"net/http"
	"net/url"
	"flag"
	"fmt"
	"os"
	"strings"
	"io/ioutil"
	"strconv"
)

// Send HTTP GET to the host
func knock(host string) {
	response, err := http.Get(host)	
	if err == nil {
		defer response.Body.Close()
		text, err := ioutil.ReadAll(response.Body)
		if err == nil {
			fmt.Println(text)			
		} else {
			fmt.Println(err)			
		}
	} else {
		fmt.Println(err)
	}	
} 

// Port knocking - send HTTP GET for the specified ports on the localhost 
func portKnocking(ports []int) {
	for port := range ports {
		host := fmt.Sprintf("http://127.0.0.1:%d", port)
		go knock(host)
	}	
}

// Parse string "0,1,2,3,\n0,1,2,4,\n", return [[0,1,2,3], [0,1,2,4]]
func getPorts(text string) [][]int{
	tuplesStr := strings.Split(text, "\n")
	tuples := [][]int{}
	for _, tupleStr := range tuplesStr {
		portsStr := strings.Split(tupleStr, ",")
		ports := []int{}
		for _, portStr := range portsStr {
			port, err := strconv.Atoi(portStr)
			if err == nil {
				ports = append(ports, port)	
			} 
		}
		if len(ports) > 0 {
			tuples = append(tuples, ports)
		}
	} 	
	return tuples
}

func getPidFilename(pid int) string {
	pidFilename := fmt.Sprintf("/tmp/knock_%d", pid)  
	return pidFilename
}

func createPidFile(ports []int) {
	pid := os.Getpid()
	pidFilename := getPidFilename(pid)
    text := []byte(fmt.Sprintf("%d\n%v\n", pid, ports))
    err := ioutil.WriteFile(pidFilename, text, 0644)
    if err != nil {
		fmt.Println("Failed to write file", pidFilename)
    }
}

// Spawn goroutines to knock the ports specified in the server response 
func handleResponse(text string) {
	ports := []int{}
	tuples := getPorts(text)
	for _, tuple := range tuples {
		for _, port := range tuple {
			ports = append(ports, port)
		}	
	}
	portKnocking(ports)
	pid := os.Getpid()
	fmt.Printf("%d:knocking %v\n", pid, ports)
}

func main() {
	host := *flag.String("host", "127.0.0.1", "Server name")
	port := *flag.Int("port", 8080, "Server port")
	host = fmt.Sprintf("%s:%d", host, port)  
	url := &url.URL{
		Scheme:   "http",
		Host:     host,
	}
	response, err := http.Get(url.String())
	if err != nil {
		panic(err)
	}	
	defer response.Body.Close()
	text, err := ioutil.ReadAll(response.Body)
	if err == nil {
		handleResponse(string(text))
	}
}