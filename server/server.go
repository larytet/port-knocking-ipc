// Get the predefined range of ports from the command line argument
// Wait for HTTP GET from a cient, generate an XML containing a set of port tuples choosen from the range of ports
// Add the set of ports to the dictionary of existing sessions
// Send the generated XML file to the client
// If a service connects get the ports and PID from the URL query, look for the file /tmp/PID, compare the data
// in the file with the ports stored in the dictionary. If there is a match removed the file /tmp/PID

package main

import (
    "sync"
    "regexp"
    "sync/atomic"
	"flag"
	"fmt"
	"log"
	"net/http"
	"bytes"
	"time"
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

type SessionId uint32
type KeyId uint64
type SessionState struct {
	expirationTime time.Time
	tuples [][]int
}


type Configuration struct {
	portsBase        int
	portsRange      []int
	portsRangeSize  int
	tolerance       int
	generator       combinations.State
	tuples          int
	tupleSize       int
	lastSessionId   SessionId
	mapSessions     map[SessionId]SessionState        
	mapTuples       map[KeyId]SessionId
	// No generics in the Golang? RME. If I want a thread safe map 
	// 'class' I have to duplicate the code for every map
	// I will use a single mutex which rules them all 
	mapMutex        sync.Mutex
	urlQueryRegex   *regexp.Regexp
}

// Setup the server configuration accrding to the command line options
func (configuration *Configuration) init() *Configuration {
	configuration.portsBase = *flag.Int("port_base", 21380, "an int")
	configuration.portsRangeSize = *flag.Int("port_range", 10, "an int")
	configuration.tolerance = *flag.Int("tolerance", 20, "an int")
	configuration.lastSessionId = SessionId(0)
	configuration.initCombinationsGenerator()
	configuration.mapSessions = make(map[SessionId]SessionState)        
	configuration.mapTuples = make(map[KeyId]SessionId)
	configuration.urlQueryRegex, _ = regexp.Compile(`\?ports=([0-9,]+)&pid=([0-9]+)`)	
	return configuration
}

// Initialize the generation for port combinations 
func (configuration *Configuration) initCombinationsGenerator() *Configuration {
	configuration.portsRange  = utils.MakeRange(configuration.portsBase, configuration.portsRangeSize)
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

// Normalize the ports in the ports tuple by substracting the minimal port number (base)
// mask the ports numner by  
// tuple[0] goes to the MSB
func tupleToKey(base uint64, tuple []int) KeyId {
	var key uint64 = 0
	for  index := uint64(0);index < uint64(len(tuple));index++ {
		if index >= MaxTupleSize {
			break
		}
		port := uint64(tuple[index]) 
		port = port - base
		port = port & MaxPortMask
		key = key << MaxPortRangeSizeBits 
		key = key | port
	}
	return KeyId(key)	
}

// Reverse of tupleToKey 
func keyToTuple(base uint64, key KeyId) (tuple []int) {
	for i := uint64(0);i < MaxTupleSize;i++ {
		port := uint64(key) & (MaxPortMask << (64-MaxPortRangeSizeBits))
		port = port >> (64-MaxPortRangeSizeBits)
		port += base
		tuple = append(tuple, int(port))
		key = key << MaxPortRangeSizeBits
	}
	return tuple	
}

func getExpirationTime() time.Time {
	const sessionTimeout = time.Duration(10) //s
	expirationTime := time.Now().UTC().Add(time.Second*sessionTimeout)
	return expirationTime
}

// Add the session to the map of sessions, all tuples to the map of tuples
func (configuration *Configuration) addSession(id SessionId, tuples [][]int) {
	configuration.mapMutex.Lock()
	defer configuration.mapMutex.Unlock()
	configuration.mapSessions[id] = SessionState{getExpirationTime(), tuples}
	base := uint64(configuration.portsBase)
	for _, tuple := range tuples {
		key := tupleToKey(base, tuple)
		configuration.mapTuples[key] = id
	}
}

func (configuration *Configuration) removeSession(id SessionId) {
	configuration.mapMutex.Lock()
	defer configuration.mapMutex.Unlock()
	sessionState, ok := configuration.mapSessions[id]
	if !ok {
		return
	}
	tuples := sessionState.tuples
	base := uint64(configuration.portsBase)
	for _, tuple := range tuples {
		key := tupleToKey(base, tuple)
		_, ok = configuration.mapTuples[key]
		if ok {
			delete(configuration.mapTuples, key)
		}
	}
	delete(configuration.mapSessions, id)
}

// Handle URL quries
func (configuration *Configuration) httpHandlerQuery(response http.ResponseWriter, query string, match [][]string) {
	for s := range match {
		fmt.Println(s)
	}
}

// HTTP server hook
func (configuration *Configuration) httpHandler(response http.ResponseWriter, request *http.Request) {
	query := request.URL.Path[1:]
	match := configuration.urlQueryRegex.FindAllStringSubmatch(query, -1)
	if len(match) != 0 {
		configuration.httpHandlerQuery(response, query, match)
	} else {
		tuples := getPortsCombinations(&configuration.generator, configuration.tuples)
		text := tuplesToText(tuples)
		sessionId := atomic.AddUint32((*uint32)(&configuration.lastSessionId), 1)
		configuration.addSession(SessionId(sessionId), tuples) 
		fmt.Fprintf(response, text)
	}
}

func main() {
	var configuration Configuration 
	configuration.init()
	http.HandleFunc("/", configuration.httpHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
