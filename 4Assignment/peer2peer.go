package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	proto "peerMutex/grpc"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc"
)

type PeerNode struct {
	proto.UnimplementedCriticalSectionServer
	processID        int
	timestamp        int64
	requesting       bool
	extNodeInCS		 bool
	idInCs			 int32
	mutex            sync.Mutex
	logger           *log.Logger
	clients		 	 []proto.CriticalSectionClient
}

var port = flag.Int("port", 5454, "peer port number")
var processID = flag.Int("id", 7, "peer process id")


func main() {
	flag.Parse()

	// Open a log file for the peer node
	logDir := "logs/"  // dict where logs will be stored 
	
	// create new log dict if not exist
	if err := os.MkdirAll(logDir, 0700); err != nil { // read write and execute permission
		log.Fatalf("Error creating log directory: %v", err)
	}
	
	// Construct the log file path using the process ID
	logFilePath := fmt.Sprintf(logDir+"peer_%v_log.txt", *processID)
	
	// Create the log file
	logFile, err := os.Create(logFilePath)

	// If there's an error creating the log file, log the error and exit
	if err != nil {
		log.Fatalf("Error creating log file: %v", err)
	}
	defer logFile.Close() // cloes when main existst

	// Create a logger for writing log messages 
	logger := log.New(logFile, "", log.LstdFlags)


	// Service Discovery (using a predefined list of nodes)

	// Implementing a possible range of addresses
	nodes := make([]string, 11)

	for i := 0; i <= 10; i++ {
		address := "localhost:" + fmt.Sprintf("%d", 5450+i)
		nodes[i] = address
	}

	// Start gRPC node peer listener
	log.Printf("Starting peer node %d at port %d\n", *processID, *port)
	thisNode:= startPeerListener(*port, *processID, logger)
	log.Printf("Peer node %d started at port %d\n", *processID, *port)

	// Register the peer with a predefined list of nodes
	// loop through the nodes
	for _, node := range nodes {
		if node == "localhost:"+strconv.Itoa(*port){ //avoid node trying to connect to itself
			continue //skip if same node
		}
		
		// go routine to connect to the grpc server
		go func(node string) {
			// optional arguments for dial grpc server
			var opts []grpc.DialOption
			opts = append(opts, grpc.WithBlock(), grpc.WithInsecure())

			conn, err := grpc.Dial(node, opts...) //establish connection to grpc server at specified node
			if err != nil {
				log.Fatalf("Failed to connect to %s: %v", node, err)
				logger.Fatalf("Failed to connect to %s: %v", node, err)
			}
			defer conn.Close()
			
			client := proto.NewCriticalSectionClient(conn) // create a client for the grpc-service defined in proto
			log.Printf("%v", client)	
			thisNode.mutex.Lock() //since we are doing go-routines, we lock to ensure 
			thisNode.clients = append(thisNode.clients, client) //
			thisNode.mutex.Unlock()
			select {} // keeps the connection to all other nodes alive
		}(node)
	}  
	
	// possible infinite while loop to check the size of the client. You can enter only when the amount of clients (thisNode.clients) is >= 2
	// print message that we initialized the servive it will only be able to got to critical section when there are 2 clients
	log.Printf("Initializing the service. Waiting for 2 clients to start the network...\n")
	logger.Printf("Initializing the service. Waiting for 2 clients or more to start the network...\n")
	for len(thisNode.clients) < 2 {
		time.Sleep(time.Second / 10)
	}
	log.Printf("Network is ready for use!\n")
	logger.Printf("Network is ready for use!\n")

	// scanner that reads the input from the console
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		message := scanner.Text() // scans over each line from the message
		thisNode.requesting = true 
		thisNode.timestamp = time.Now().Unix() // could be queue here
		var counterNICS = 0
		if thisNode.extNodeInCS { // if there is another node in the critical section then keep monotoring
			for thisNode.extNodeInCS{
				time.Sleep(time.Second)
				log.Printf("There is another node in the critical section [Node Id: %d]. Please, wait...(%d)\n", thisNode.idInCs, counterNICS)
				logger.Printf("There is another node in the critical section [Node Id: %d]. Please, wait...(%d)\n", thisNode.idInCs, counterNICS)
				counterNICS++ //number of times the given node tries to access the critical section
				}
			}
			// Try to enter in the critical section as soon as the other node leaves
			counterEnter := 0
			for {
				if tryToEnterCriticalSection(thisNode, message, counterEnter) {
					break
				}
				counterEnter++
				log.Printf("Counter tryToEnterCriticalSection %d", counterEnter)
			}
			thisNode.mutex.Lock()
			log.Printf("This peer node now is in the critical section! ID: %d", *processID)
			logger.Printf("This peer node now is in the critical section! ID: %d", *processID)

			log.Printf("Printing the message during the critical section:" + message)
			logger.Printf("Printing the message during the critical section:" + message)

			log.Printf("Waiting for 5 seconds before leaving the critical section")
			logger.Printf("Waiting for 5 seconds before leaving the critical section")

			time.Sleep(time.Second * 5)
			//leaving the critical section
			for _, client := range thisNode.clients {
			_, _ = client.LeaveCriticalSection(context.Background(), &proto.RequestMessage{
				Timestamp: int32(time.Now().Unix()),
				ProcessId: int32(*processID),
				Message: "Leaving critical section",
				})
			}
			thisNode.timestamp = time.Now().Unix()
			thisNode.requesting = false
			// unlock the mutex
			thisNode.mutex.Unlock()
			log.Printf("This peer node now left the critical section! ID: %d", *processID)
			logger.Printf("This peer node now left the critical section! ID: %d", *processID)
	}
}


func tryToEnterCriticalSection(thisNode *PeerNode, message string, counter int) bool {
	for _, client := range thisNode.clients {
		// request for every node in the list
		response, err := client.RequestToEnterCriticalSection(context.Background(), &proto.RequestMessage{
			Timestamp: int32(thisNode.timestamp),
			ProcessId: int32(*processID), // Ensure processID is not nil and is a pointer
			Message: message,
		})

		if err != nil {
			// handle the error appropriately
			log.Printf("Error while requesting to enter critical section: %v\n", err)
			return false
		}
		if !response.Permission {
			log.Printf("Unauthorized to enter in the critical section right now. Number of tries: %d\n", counter)
			// waits 25 ms to try again
			time.Sleep(time.Second / 4)
			return false
		}
	}
	return true
}

func (p *PeerNode) ValidateRequestEnterCriticalSection(ctx context.Context, req *proto.ValidationMessage) (*proto.ResponseMessage, error) {
	if ((p.requesting && (req.Timestamp <= int32(p.timestamp))) || !p.requesting) {
		// validation function that checks the other nodes
		// get list of clients and check if all of them are true
		return &proto.ResponseMessage{Permission: true}, nil
	}
	return &proto.ResponseMessage{Permission: false}, nil
}

func (p *PeerNode) RequestToEnterCriticalSection(ctx context.Context, req *proto.RequestMessage) (*proto.ResponseMessage, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	var response *proto.ResponseMessage
	var err error
	if ((p.requesting && (req.Timestamp < int32(p.timestamp))) || !p.requesting) {
		// validation function that checks the other nodes
		// get list of clients and check if all of them are true
		// TODO: change chlient for specific client
		// loop through all the clients and check if all of them are true
		for _, client := range p.clients {
			response, err = client.ValidateRequestEnterCriticalSection(context.Background(), &proto.ValidationMessage {
				Timestamp: req.Timestamp,})
			if err != nil {
			// Handle the error appropriately
			return nil, err
			}
			if !response.Permission { //if any validation check return false, the node that is sending the request will be denied permission
				return &proto.ResponseMessage{Permission: false}, nil
			}		
		}	
		p.extNodeInCS = true //
		p.requesting = false //the node is done with its request
		p.idInCs = req.ProcessId
		// Print the message that another peer node is entering in the critical section
		log.Printf("External Peer Entering in the Critical Section - Node %d: %s\n", req.ProcessId, req.Message)	
		p.logger.Printf("External Peer Entering in the Critical Section - Node %d: %s\n", req.ProcessId, req.Message)
		//return the permission true!
		return &proto.ResponseMessage{Permission: true}, nil
	}
	return &proto.ResponseMessage{Permission: false}, nil
}

func (p *PeerNode) LeaveCriticalSection(ctx context.Context, req *proto.RequestMessage) (*proto.ResponseMessage, error){
	p.mutex.Lock()
	defer p.mutex.Unlock()

	//Check the ID process in the critical section
	if (req.ProcessId == p.idInCs){ // the node that is requesting to leave should be the same as the one within the critical section
		p.extNodeInCS = false
		p.idInCs = 0

		log.Printf("External Peer Leaving the Critical Section - Node %d: %s\n", req.ProcessId, req.Message)	
		p.logger.Printf("External Peer Leaving the Critical Section - Node %d: %s\n", req.ProcessId, req.Message)

		return &proto.ResponseMessage{Permission: true}, nil
	}
	//invalid request (should never happen)
	log.Printf("External Peer Request to Leave but the ID doesn't match - Node %d != idInCS: %d\n", req.ProcessId, p.idInCs)	
	p.logger.Printf("External Peer Request to Leave but the ID doesn't match - Node %d != idInCS: %d\n", req.ProcessId, p.idInCs)
	return &proto.ResponseMessage{Permission: false}, nil
}

func startPeerListener(port int, processID int, logger *log.Logger) *PeerNode{
	// Create a new grpc peer node
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		log.Fatalf("Could not create the peer node: %v", err)
		logger.Fatalf("Could not create the peer node: %v", err)
	}
	log.Printf("Started peer node at port: %d\n", port)  // logs in the console
	logger.Printf("Started peer node at port: %d\n", port)  // logs in the file

	// Create a new grpc server
	peer := grpc.NewServer()

	thisNode := &PeerNode{
		processID:        processID,
		timestamp:        time.Now().Unix(),
		requesting:       false,
		extNodeInCS:      false,
		idInCs:           0,
		mutex:            sync.Mutex{},
		logger:           logger,
		clients:		  []proto.CriticalSectionClient{},
	}

	proto.RegisterCriticalSectionServer(peer, thisNode) 

	// Start serving the listener
	go func() {
		if err := peer.Serve(listener); err != nil {  // go rotine listens to the server
			log.Fatalf("Failed to serve listener: %v", err)
			logger.Fatalf("Failed to serve listener: %v", err)
		}
	}()

	return thisNode
}