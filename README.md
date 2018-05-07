# port-knocking-ipc

## Introduction

Based on the idea https://stackoverflow.com/questions/49978730/reliable-delivery-of-information-between-a-browser-and-a-locally-run-service-usi

The goal is to reliably deliver a key from a WEB server to a locally running service via a WEB page opened in a WEB browser without introducing an HTTPS server on the local machine. 

Some applications:

* Browser version identification
* User identification
* Getting system settings 
* Modifying the local system via SaaS 

The server generates a combination of ports from a predefined port range. The server generates an HTML page which establishes connections to the WEB server running on the local host (127.0.0.1). The service listens for the connection attempts, sorts the "knocks" by process ID. The service closes the connections. The service sends the collected "knocks" to the server with the required information. The server can response with further instructions. 

## Limitations

The server is susceptible to the replay attacks. For example an adversary can constantly send a query with a specific port combination until it gets a positive response from the server. The server can introduce "holes" when choosing ports combinations by skipping a random number of combinations.

The service should divide the stream of collected port knocks into ports tuples. Service probably failed to bind some ports. The service assumes the ascending order of ports in the ports tuples.
The client (a browser) should not reorder the ports in the tuples. Usually the order of "knocks" can be enforced in the JS. If the order is not possible to
enforce the client can introduce "start tuple" knock between ports tuples. A start tuple knock is knocking a special port which service surely could bind. The client sends the tuple start knock, waits, follows by the ports of the tuple in arbitrary order, waits again, repeats with the next tuple.

## In the source code

### Server

Get the predefined range of ports from the command line argument
Wait for HTTP GET from a cient, generate an XML containing a set of port tuples choosen from the range of ports
Add the set of ports to the dictionary of existing sessions
Send the generated XML file to the client
If a service connects get the ports and PID from the URL query, look for the file /tmp/PID, compare the data
in the file with the ports stored in the dictionary. If there is a match removed the file /tmp/PID

### Client

Send HTTP GET to the server
Parse the XML reponses, write the ports combination to the file /tmp/PID
Establish TCP connections with the service using the ports specified in the XML file
Poll the file /tmp/PID for 10s. If the file is not removed, print error, remove the file

### Service

Get the ports range from the command line, list of ports to skip 
for simulaiton of failure of bind
Bind the specified ports
Wait for TCP connections from a client
Accept connection, collect the port number and the client PID
When the required number of ports are knocked or a timeout expired send  
all possible combinations of the collected port knocks and ports the service 
failed to bind to the server


## Tolerance for failures to bind ports
 
The suggested scheme allows the service to tolerate failure to bind some of the ports in the predefined range. The idea is that if the service failed to bind a port it will send all possible combinations of the collected "knocks" and the ports the service failed to bind.

Let's say we use 2-tuples of 5
```python
import itertools 
list(itertools.combinations([0,1,2,3,4,5],2))
[(0, 1), (0, 2), (0, 3), (0, 4), (0, 5), (1, 2), (1, 3), (1, 4), (1, 5), (2, 3), (2, 4), (2, 5), (3, 4), (3, 5), (4, 5)]
```

The server chooses three 2-tuples. The server sends all three to the end point. All possible choices:

    (0, 1), (0, 2), (0, 3)
    (0, 4), (0, 5), (1, 2)
    (1, 3), (1, 4), (1, 5)
    (2, 3), (2, 4), (2, 5)
    (3, 4), (3, 5), (4, 5)

The possible port knocks look like

    (0,0,0,1,2,3)
    (0,0,1,2,4,5)
    (1,1,1,3,4,5)
    (2,2,2,3,4,5)
    (3,3,4,4,5,5)

The end point collects the port knocks. Let's say that one end point (A) failed to bind port 0 and the second end point (B) failed to bind port 1. The collected port knocks 

       A                             b
    (1,2,3)                    (0,0,0,2,3)
    (1,2,4,5)                  (0,0,2,4,5)
    (1,1,1,3,4,5)              (3,4,5)
    (2,2,2,3,4,5)              (2,2,2,3,4,5) 
    (3,3,4,4,5,5)              (3,3,4,4,5,5)

The end points send the following responses to the server

          A                                   B
    [(0,0,0,1,2,3)]                    [(0,0,0,1,2,3)]
    [(0,0,1,2,4,5)]                    [(0,0,1,2,4,5)]
    [(1,1,1,3,4,5)]                    [(1,1,1,3,4,5)]
    [(2,2,2,3,4,5)]                    [(2,2,2,3,4,5)]
    [(3,3,4,4,5,5)]                    [(3,3,4,4,5,5)]

Let's say that one end point failed to bind port 0 and the second end point failed to bind ports 0 and 1. The collected port knocks 

       A                         B
    (1,2,3)                    (2,3)
    (1,2,4,5)                  (2,4,5)
    (1,1,1,3,4,5)              (3,4,5)
    (2,2,2,3,4,5)              (2,2,2,3,4,5) 
    (3,3,4,4,5,5)              (3,3,4,4,5,5)

The end points send the following responses to the server

           A                  B
    [(0,0,0,1,2,3)]    [(0,0,0,0,2,3), (0,0,0,1,2,3), (0,0,1,1,2,3), (0,1,1,1,2,3), (1,1,1,1,2,3)]
    [(0,0,1,2,4,5)]    [(0,0,0,2,4,5), (0,0,1,2,4,5), (0,1,1,2,4,5), (1,1,1,2,4,5)]
    [(1,1,1,3,4,5)]    [(0,0,0,3,4,5), (0,0,1,3,4,5), (0,1,1,3,4,5), (1,1,1,3,4,5)]
    [(2,2,2,3,4,5)]    [(2,2,2,3,4,5)]
    [(3,3,4,4,5,5)]    [(3,3,4,4,5,5)]


## Usage

    git clone https://github.com/larytet/port-knocking-ipc.git
    cd port-knocking-ipc
    ./buildall;./testall
    ~/go/bin/server &
    ~/go/bin/service &
    ~/go/bin/client
    
## Links

* http://marcio.io/2015/07/handling-1-million-requests-per-minute-with-golang/
* https://stackoverflow.com/questions/49978730/reliable-delivery-of-information-between-a-browser-and-a-locally-run-service-usi
* https://stackoverflow.com/questions/10125881/send-a-message-from-javascript-running-in-a-browser-to-a-windows-batch-file
* http://kb.mozillazine.org/Register_protocol

