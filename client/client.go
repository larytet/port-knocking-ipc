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
	"time"
	"os"
	"strings"
	"io/ioutil"
	"strconv"
	"port-knocking-ipc/utils"
)

// Send HTTP GET to the host
// Blocking
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
// Unblocking 
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

func createPidFile(ports []int) (string, bool) {
	pid := os.Getpid()
	pidFilename := getPidFilename(pid)
    text := []byte(fmt.Sprintf("%d\n%v\n", pid, ports))
    err := ioutil.WriteFile(pidFilename, text, 0644)
    if err != nil {
		fmt.Println("Failed to write file", pidFilename)
		return pidFilename, false
    }
	return pidFilename, true
}

// Wait until "server" removes the file
func waitForPidfile(filename string) {
	timeout := time.Duration(10*1000)
	check_period := time.Duration(100)
	period := timeout/check_period 
	for !utils.PathExists(filename) && period > 0 {
		time.Sleep(check_period * time.Millisecond)
		period -= 1			
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
	// portKnockig() does not block
	portKnocking(ports)
	pidFilename, ok := createPidFile(ports)
	if ok {
		waitForPidfile(pidFilename)
	}
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