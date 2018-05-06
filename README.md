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

    
## Links

* http://marcio.io/2015/07/handling-1-million-requests-per-minute-with-golang/
* https://stackoverflow.com/questions/49978730/reliable-delivery-of-information-between-a-browser-and-a-locally-run-service-usi
