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
	"strings"
	"io/ioutil"
	"strconv"
)

func knock(port int) {
	host := fmt.Sprintf("127.0.0.1:%d", port)
	response, err := http.Get(host)	
	if err == nil {
		defer response.Body.Close()
	}	
	fmt.Println("Knock", port)
} 

func getPorts(text string) [][]int{
	tuplesStr := strings.Split(text, "\n")
	tuples := [][]int{}
	for _, tupleStr := range tuplesStr {
		portsStr := strings.Split(tupleStr, "\n")
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

func handleResponse(text string) {
	tuples := getPorts(text)
	for _, tuple := range tuples {
		for _, port := range tuple {
			fmt.Println("Knock", port)
			go knock(port)
		}	
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