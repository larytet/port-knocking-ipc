// Get the predefined range of ports from the command line argument
// Wait for HTTP GET from a cient, generate an XML containing a set of port tuples choosen from the range of ports
// Add the set of ports to the dictionary of existing sessions
// Send the generated XML file to the client
// If a service connects get the ports and PID from the URL query, look for the file /tmp/PID, compare the data
// in the file with the ports stored in the dictionary. If there is a match removed the file /tmp/PID

package main

import (
    "sync"
    "os"
    "strings"
    "sync/atomic"
    "net/url"
    "math/rand"
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

const maxPortRangeSizeBits uint64 = 8 // bits 
const maxPortMask uint64 = ((1 << maxPortRangeSizeBits)-1) 
const maxPortRangeSize uint64 = (1 << maxPortRangeSizeBits)  // ports in a range
const maxTupleSize uint64 = 64/maxPortRangeSizeBits // ports in a tuple   

type sessionID uint32
type keyID uint64
type sessionState struct {
	id sessionID
	expirationTime time.Time
	tuples [][]int
}


type configuration struct {
	portsBase        int
	portsRange      []int
	portsRangeSize  int
	tolerance       int
	generator       combinations.State
	tuples          int
	tupleSize       int
	lastSessionID   sessionID
	mapSessions     map[sessionID]sessionState        
	mapTuples       map[keyID]sessionID
	// No generics in the Golang? RME. If I want a thread safe map 
	// 'class' I have to duplicate the code for every map
	// I will use a single mutex which rules them all 
	mapMutex        sync.Mutex
}

// Setup the server configuration accrding to the command line options
func createConfiguration() *configuration   {
	c := configuration{
		portsBase : *flag.Int("port_base", 21380, "Base port number"),
		portsRangeSize : *flag.Int("port_range", 10, "Size of the ports range"),
		tolerance : *flag.Int("tolerance", 20, "Percent of tolerance for port bind failures"),
		lastSessionID : sessionID(0),
		mapSessions : make(map[sessionID]sessionState),        
		mapTuples : make(map[keyID]sessionID),
	}
	result := &c
	result.initCombinationsGenerator()
	
	return result
}

// Initialize the generation for port combinations 
func (c *configuration) initCombinationsGenerator() *configuration {
	c.portsRange  = utils.MakeRange(c.portsBase, c.portsRangeSize)
	c.tupleSize = c.portsRangeSize/2
	// I want to allocate enough tuples to reach the specifed tolerance level
	c.tuples = (c.tolerance*c.tupleSize)/100 + 2
	if c.tolerance == 0 {
		c.tuples = 1		
	}  
	
	c.generator = combinations.Init(c.portsRange, c.tupleSize)
	
	return c
}

// Get next set of port combinations
// Randomly skip combinations with the specified probabilty. I do the skipping part
// to thwart replay attacks. The idea is that the server will only rarely repeat  
// allocated combinations. TODO More thinking is required here
func getPortsCombinations(generator *combinations.State, count int, skipProbability int) ([][]int) {
	tuples := make([][]int, 0)
	for count > 0 {
		toSkip := (skipProbability > 0) && (rand.Intn(100) < skipProbability)
		if !toSkip {
			// I have to clone the slice generator.Next() returns the same reference
			tuple := generator.NextWrap()
			tuples = append(tuples, tuple)
			count--
		}
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
func tupleToKey(base uint64, tuple []int) keyID {
	var key uint64
	for  index := uint64(0);index < uint64(len(tuple));index++ {
		if index >= maxTupleSize {
			break
		}
		port := uint64(tuple[index]) 
		port = port - base
		port = port & maxPortMask
		key = key << maxPortRangeSizeBits 
		key = key | port
	}
	return keyID(key)	
}

// Reverse of tupleToKey 
func keyToTuple(base uint64, key keyID) (tuple []int) {
	for i := uint64(0);i < maxTupleSize;i++ {
		port := uint64(key) & (maxPortMask << (64-maxPortRangeSizeBits))
		port = port >> (64-maxPortRangeSizeBits)
		port += base
		tuple = append(tuple, int(port))
		key = key << maxPortRangeSizeBits
	}
	return tuple	
}

func getExpirationTime() time.Time {
	const sessionTimeout = time.Duration(10) //s
	expirationTime := time.Now().UTC().Add(time.Second*sessionTimeout)
	return expirationTime
}

// Add the session to the map of sessions, all tuples to the map of tuples
func (c *configuration) addSession(id sessionID, tuples [][]int) {
	c.mapMutex.Lock()
	defer c.mapMutex.Unlock()
	c.mapSessions[id] = sessionState{id, getExpirationTime(), tuples}
	base := uint64(c.portsBase)
	for _, tuple := range tuples {
		key := tupleToKey(base, tuple)
		c.mapTuples[key] = id
	}
}

func (c *configuration) removeSession(id sessionID) (tuples, tuplesRemoved [][]int, ok bool) {
	c.mapMutex.Lock()
	defer c.mapMutex.Unlock()
	sessionState, ok := c.mapSessions[id]
	if !ok {
		return nil, nil, false
	}
	tuples = sessionState.tuples
	tuplesRemoved = [][]int{}
	base := uint64(c.portsBase)
	for _, tuple := range tuples {
		key := tupleToKey(base, tuple)
		_, ok = c.mapTuples[key]
		if ok {
			delete(c.mapTuples, key)
			tuplesRemoved = append(tuplesRemoved, tuple)
		}
	}
	delete(c.mapSessions, id)
	return tuples, tuplesRemoved, true
}

func parseURLQuerySessionPorts(portsStr []string, tupleSize int) ([][]int, bool) {
	if len(portsStr) != 1 {
		return nil, false
	}
	portsStrs := strings.Split(portsStr[0], ",")
	if len(portsStrs) < 1 {
		return nil, false
	}
	portsStrs = portsStrs[:len(portsStrs)-1]
	if len(portsStrs) < 1 {
		return nil, false
	}
	var ports [][]int
	portsCount := 0
	for _, portStr := range portsStrs {
		if portsCount % tupleSize == 0 {
			ports = append(ports, []int{})			
		}		
		port, ok := utils.AtoIPPort(portStr)
		if !ok {
			return nil, false
		}
		tupleIndex := portsCount/tupleSize
		ports[tupleIndex] = append(ports[tupleIndex], port)
		portsCount++
	}
	return ports, true
}

func parseURLQuerySessionPid(pidStr []string) (int, bool) {
	if len(pidStr) != 1 {
		return 0, false 
	}
	pid,  ok := utils.AtoPid(pidStr[0])
	if !ok {
		return 0, false 		
	}
	return pid, true
}

// Look for all tuples in the map, collect session IDs
// Number of sessions can be any positive number, can be zero.
// The tuples can match more than one session if, for example, the client 
// has failed to bind all ports  
func (c *configuration) findSessions(tuples [][]int) []sessionState {
	sessions := []sessionState{}
	c.mapMutex.Lock()
	defer c.mapMutex.Unlock()
	for _, tuple := range tuples {
		key := tupleToKey(uint64(c.portsBase), tuple)
		sessionID, ok := c.mapTuples[key]
		if ok {
			session, ok := c.mapSessions[sessionID]
			if ok {
				if len(sessions) > 0 {
					if sessions[0].id != sessionID {
						sessions = append(sessions, session)
					} 
				} else {
					sessions = append(sessions, session)
				} 
			} 
		}
	}
	return sessions
}

// Handle URL query /session?ports=...&pid=...
func (c *configuration) httpHandlerSession(response http.ResponseWriter, query url.Values) {
	portsStr, ok := query["ports"]
	if !ok {
		fmt.Fprintf(response, "No parameter 'ports'")
		return
	}
	pidStr, ok := query["pid"]
	if !ok {
		fmt.Fprintf(response, "No parameter 'pid'")
		return
	}
	tuples, ok := parseURLQuerySessionPorts(portsStr, c.tupleSize)
	if !ok {
		fmt.Fprintf(response, "Failed to parse '%s'", tuples)
		return
	}
	pid, ok := parseURLQuerySessionPid(pidStr)
	if !ok {
		fmt.Fprintf(response, "Failed to parse '%s'", pidStr)
		return
	}
	sessions := c.findSessions(tuples)
	if len(sessions) == 0 {
		fmt.Fprintf(response, "No session is found for %v, pid %d", tuples, pid)
		return
	}
	if len(sessions) > 1 {
		fmt.Fprintf(response, "Found %v (%d) sessions for tuples %v, pid %d", sessions, len(sessions), tuples, pid)
		return
	}
	session := sessions[0]
	tuples, tuplesRemoved, ok := c.removeSession(session.id)
	if !ok {
		fmt.Fprintf(response, "Failed to remove sesion %v for %v, pid %d", session, tuples, pid)
		return
	}
	if len(tuples) != len(tuplesRemoved) {
		fmt.Fprintf(response, "Failed to remove all tuples for %v, tuples=%v, removed=%v, pid=%d", session, tuples, tuplesRemoved, pid)
		return
	}
	pidFilename := utils.GetPidFilename(pid)
	if err := os.Remove(pidFilename); err != nil {
		fmt.Fprintf(response, "Failed to remove file %s %s\n", pidFilename, err)		
	} else {
		fmt.Fprintf(response, "File %s removed\n", pidFilename)				
	}
	fmt.Fprintf(response, "Removed tuples for session %v, pid %d\n", session, pid)
}

// Allocate combinations of ports (ports tuples), generate response text, update the sessions map 
func (c *configuration) httpHandlerRoot(response http.ResponseWriter, query url.Values) {
	tuples := getPortsCombinations(&c.generator, c.tuples, 2)
	id := atomic.AddUint32((*uint32)(&c.lastSessionID), 1)
	c.addSession(sessionID(id), tuples) 
	text := tuplesToText(tuples)
	fmt.Fprintf(response, text)
}

// HTTP server hook
func (c *configuration) httpHandler(response http.ResponseWriter, request *http.Request) {
	path := request.URL.Path[1:]
	query := request.URL.Query()
	if path == "session" {
		c.httpHandlerSession(response, query)
	} else {
		c.httpHandlerRoot(response, query)
	}
}

func main() {
	rand.Seed((int64)(time.Millisecond))
	var c = createConfiguration() 
	http.HandleFunc("/", c.httpHandler)
	port := ":8080"
	fmt.Println("Listening on", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
