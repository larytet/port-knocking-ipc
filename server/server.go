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

// Golang's map does not support a key to be a "slice". A key can be olnly fixed size array, string or a structure
// I am not kidding https://stackoverflow.com/questions/26559568/using-variable-length-array-as-a-map-key-in-golang
// Lack of generics (don't ask) does not allow me to define an interface supporintg "comparable" 
// The idea (my speculation) is that the authors intended to force hashtable keys to be an integer or a string, enforce
// specific designs 
// The goog news are that uint64 and uint32 keys use a bypass - very fast hashing https://github.com/golang/go/issues/13271
// I implement two functions which convert between the ports tuples and uint64. This approach introduces limitations:
// * Complicates use of multiple port ranges 
// * Limits size of the ports range
// * Limts number of ports in the ports tuple
// Number of possible combinations for 8 ports tuple from a range of 128 ports https://www.wolframalpha.com/input/?i=128+choose+8
const MaxPortRangeSizeBits uint64 = 8 // bits 
const MaxPortMask uint64 = ((1 << MaxPortRangeSizeBits)-1) 
const MaxPortRangeSize uint64 = (1 << MaxPortRangeSizeBits)  // ports in a range
const MaxTupleSize uint64 = 64/MaxPortRangeSizeBits // ports in a tuple   

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


// Normalize the ports in the ports tuple by substracting the minimal port number (base)
// mask the ports numner by  
// tuple[0] goes to the MSB
func tupleToKey(base int, tuple []int) uint64 {
	var key uint64 = 0
	for  index := uint64(0);index < uint64(len(tuple));index++ {
		if index >= MaxTupleSize {
			break
		}
		port := uint64(tuple[index]) & MaxPortMask
		key = key << MaxPortRangeSizeBits 
		key = key | port
	}
	return key	
}

// Reverse of tupleToKey 
func keyToTuple(base int, key uint64) (tuple []int) {
	for i := uint64(0);i < MaxTupleSize;i++ {
		port := key & (MaxPortMask << (64-MaxPortRangeSizeBits))
		port = port >> (64-MaxPortRangeSizeBits)
		tuple = append(tuple, int(port))
		key = key << MaxPortRangeSizeBits
	}
	return tuple	
}

func main() {
	var configuration Configuration 
	configuration.init()
	http.HandleFunc("/", configuration.httpHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
