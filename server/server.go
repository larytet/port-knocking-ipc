// Get the predefined range of ports from the command line argument
// Wait for HTTP GET from a cient, generate an XML containing a set of port tuples choosen from the range of ports
// Add the set of ports to the dictionary of existing sessions
// Send the generated XML file to the client
// If a service connects get the ports and PID from the URL query, look for the file /tmp/PID, compare the data
// in the file with the ports stored in the dictionary. If there is a match removed the file /tmp/PID

package main

import (
    //"sync"
    "sync/atomic"
	"flag"
	"fmt"
	"log"
	"net/http"
	"bytes"
	"port-knocking-ipc/utils/combinations"
	"port-knocking-ipc/utils"
)

type Configuration struct {
	portBase        int
	portsRange      []int
	portsRangeSize  int
	tolerance       int
	generator       combinations.State
	tuples          int
	tupleSize       int
	lastSessionId   uint32
	mapSessionPorts map[uint32][][]int        
	mapPortsSession map[uint64]uint32
}

// Setup the server configuration accrding to the command line options
func (configuration *Configuration) init() *Configuration {
	configuration.portBase = *flag.Int("port_base", 21380, "an int")
	configuration.portsRangeSize = *flag.Int("port_range", 10, "an int")
	configuration.tolerance = *flag.Int("tolerance", 20, "an int")
	configuration.lastSessionId = 0
	configuration.initCombinationsGenerator()
	configuration.mapSessionPorts = make(map[uint32][][]int)        
	configuration.mapPortsSession = make(map[uint64]uint32)
	return configuration
}

// Initialize the generation for port combinations 
func (configuration *Configuration) initCombinationsGenerator() *Configuration {
	configuration.portsRange  = utils.MakeRange(configuration.portBase, configuration.portsRangeSize)
	var portsCount = len(configuration.portsRange)
	configuration.tupleSize = portsCount/2
	// I want to allocate enough tuples to reach the specifed tolerance level
	configuration.tuples = (configuration.tolerance*configuration.tupleSize)/100 + 2
	if configuration.tolerance == 0 {
		configuration.tuples = 1		
	}  
	
	configuration.generator = combinations.Init(configuration.portsRange, configuration.tupleSize)
	
	return configuration
}

// Get next set of port combinations
func getPortsCombinations(generator *combinations.State, count int) ([][]int) {
	tuples := make([][]int, 0)
	for i := 0;i < count;i++ {
		// I have to clone the slice generator.Next() returns the same reference
		tuple := generator.NextWrap()
		tuples = append(tuples, tuple)
	}
	return tuples
}

// Generate text containing the ports to knock
// Every line is a list of ports separated by a comma
func tuplesToText(tuples [][]int) string {
	var text bytes.Buffer 
	for i := 0;i < len(tuples);i++ {
		tuple := tuples[i]
		text.WriteString(utils.ToString(tuple, ","))
		text.WriteString("\n")
	}
	return text.String()
}

// HTTP server hook
func (configuration *Configuration) httpHandler(response http.ResponseWriter, request *http.Request) {
	tuples := getPortsCombinations(&configuration.generator, configuration.tuples)
	text := tuplesToText(tuples)
	sessionId := atomic.AddUint32(&configuration.lastSessionId, 1)
	configuration.mapSessionPorts[sessionId] = tuples 
	//fmt.Fprintf(response, "Hi there, I love %s!", request.URL.Path[1:])
	fmt.Fprintf(response, text)
}

func main() {
	var configuration Configuration 
	configuration.init()
	http.HandleFunc("/", configuration.httpHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
