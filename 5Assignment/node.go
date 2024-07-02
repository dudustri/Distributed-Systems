package main

import (
	//"bufio"
	//"context"
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	proto "replication/grpc"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
)

type NodeType string

const (
	Primary   NodeType = "Primary"
	Secondary NodeType = "Secondary"
)

type Node struct {
	Type NodeType
	proto.UnimplementedAuctionServiceServer
	maxbid                  int32
	isAuctionEnded          bool
	mutex                   sync.Mutex
	logger                  *log.Logger
	nodes                   map[string]proto.AuctionServiceClient
	nodeId                  int32
	clientIdMaxBid			string
	nodeTime                time.Time
	lastTimeReceivedPrimary time.Time
	secondaryPIDs 			map[string]int
	secondaryNodesTimeCheck map[proto.AuctionServiceClient]time.Time
	portName				string
	leaderPID				int
}

var port = flag.Int("port", 5454, "node port number")
var nodeId = flag.Int("id", 7, "node id")
var timestampStr = flag.String("timestamp", "", "Timestamp from primary node")
var leaderPID = flag.Int("leaderPID", 0, "Leader PID")

func main() {
	flag.Parse()

	// Open a log file for the peer node
	logDir := "logs/" // dict where logs will be stored

	// create new log dict if not exist
	if err := os.MkdirAll(logDir, 0700); err != nil { // read write and execute permission
		log.Fatalf("Error creating log directory: %v", err)
	}

	// Construct the log file path using the process ID
	logFilePath := fmt.Sprintf(logDir+"node_%v_log.txt", *nodeId)
	
	// Create the log file
	logFile, err := os.Create(logFilePath)

	// If there's an error creating the log file, log the error and exit
	if err != nil {
		log.Fatalf("Error creating log file: %v", err)
	}
	defer logFile.Close() // cloes when main existst

	// Create a logger for writing log messages
	logger := log.New(logFile, "", log.LstdFlags)

	//We create a slice of size 3 to store the addresses of our nodes
	addresses := make([]string, 4)

	//We loop through our slice and create the addresses for our nodes
	for i := 0; i <= 3; i++ {
		address := "localhost:" + fmt.Sprintf("%d", 5450+i)
		addresses[i] = address
	}

	// Start gRPC node peer listener
	log.Printf("Starting peer node %d at port %d\n", *nodeId, *port)

	// Do timestamp check
	var timestamp time.Time
	var timeErr error
	if *timestampStr != "" {
		timestamp, timeErr = time.Parse(time.RFC3339, *timestampStr)
		if timeErr != nil {
			log.Fatalf("Failed to parse timestamp: %v", err)
		}
	} else {
		timestamp = time.Now() // Use the current time if no timestamp is provided
	}

	thisNode := startNode(*port, logger, int32(*nodeId), timestamp, *leaderPID)

	log.Printf("%d", thisNode.leaderPID)

	// Give the nodes client-like capabilities
	for _, address := range addresses {
		if address == "localhost:"+strconv.Itoa(*port) { //avoid node trying to connect to itself
			continue //skip if same node
		}

		// go routine to connect to the grpc server -giving it client-like capabilities
		go func(address string) {
			// optional arguments for dial grpc server
			var opts []grpc.DialOption
			opts = append(opts, grpc.WithBlock(), grpc.WithInsecure())

			//establish connection to grpc server at specified node
			conn, err := grpc.Dial(address, opts...)
			if err != nil {
				log.Fatalf("Failed to connect to %s: %v", address, err)
				logger.Fatalf("Failed to connect to %s: %v", address, err)
			}
			defer conn.Close()

			client := proto.NewAuctionServiceClient(conn) // create a client for the grpc-service defined in proto
			log.Printf("%v %v",address, client)
			thisNode.mutex.Lock()                           //since we are doing go-routines, we lock to ensure
			thisNode.nodes[address] = client // put the node string adress as key and the client connection as value
			// thisNode.clients = append(thisNode.clients, client) //
			thisNode.mutex.Unlock()
			select {} // keeps the connection to all other nodes alive
		}(address)
	}
	if(thisNode.Type == Primary){
		go thisNode.verifySecondaryNodesHeartbeat()
	}else{
		go thisNode.startheartBeat()
		go thisNode.verifyPrimaryNodeAlive()
	}
		select {}
}

func (s *Node) Bid(ctx context.Context, req *proto.BidRequest) (*proto.AckResponse, error) {
	// check
	if s.Type == Secondary {
		return &proto.AckResponse{Ack: "Exception: This node is secondary can't update other nodes"}, nil
	}
	var timeNow = time.Now()
	durationSinceTimestamp := timeNow.Sub(s.nodeTime)
	if durationSinceTimestamp > 3*time.Minute {
		s.isAuctionEnded = true
		log.Printf("Auction has ended (3 minutes has past). Start time: %v Now: %v\n", s.nodeTime, time.Now())
		s.logger.Printf("Auction has ended (3 minutes has past). Start time: %v Now: %v\n", s.nodeTime, time.Now())
		return &proto.AckResponse{Ack: "Exception: Auction has ended"}, nil
	}
	if req.Bid > s.maxbid {
		s.maxbid = req.Bid
		s.clientIdMaxBid = req.ClientId
		for _, node := range s.nodes {
			node.UpdateNode(context.Background(), &proto.BidRequest{Bid: s.maxbid, ClientId: s.clientIdMaxBid, LeaderPID: int32(s.leaderPID)}) // update all other nodes of the new max bid
		}
		log.Printf("New Bid accepted with value: %d client: %v\n", req.Bid, req.ClientId)
		s.logger.Printf("New Bid accepted with value: %d client: %v\n", req.Bid, req.ClientId)
		return &proto.AckResponse{Ack: "Sucess: Bid received"}, nil
	} else {
		log.Printf("Bid rejected - value: %d client: %v\n", req.Bid, req.ClientId)
		s.logger.Printf("Bid rejected - value: %d client: %v\n", req.Bid, req.ClientId)
		return &proto.AckResponse{Ack: "Failed: Bid amount is less than current max bid"}, nil
	}
}

func (s *Node) UpdateNode(ctx context.Context, req *proto.BidRequest) (*proto.AckResponse, error) {
	// inform all other nodes of the new max bid
	s.maxbid = req.Bid
	s.clientIdMaxBid = req.ClientId
	s.leaderPID = int(req.LeaderPID)

	log.Printf("Bid rejected - value: %d client: %v\n", req.Bid, req.ClientId)
	s.logger.Printf("Bid rejected - value: %d client: %v\n", req.Bid, req.ClientId)

	return &proto.AckResponse{Ack: "Success: Nodes updated"}, nil
}

func (s *Node) Result(ctx context.Context, req *proto.ResultRequest) (*proto.ResultResponse, error) {
	log.Printf("A client request for results.")
	s.logger.Printf("A client request for results.")
	if time.Now().Sub(s.nodeTime) > 3 * time.Minute {
		s.isAuctionEnded = true
		log.Printf("Auction has ended (3 minutes has past). Start time: %v Now: %v\n", s.nodeTime, time.Now())
		s.logger.Printf("Auction has ended (3 minutes has past). Start time: %v Now: %v\n", s.nodeTime, time.Now())
	}
	return &proto.ResultResponse{MaxBid: s.maxbid, IsAuctionEnded: s.isAuctionEnded, Winner: s.clientIdMaxBid}, nil
}

func startNode(port int, logger *log.Logger, nodeId int32, timestamp time.Time, leaderPIDflag int) *Node {
	// Create a new grpc peer node
	var n *Node

	listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		log.Fatalf("Could not create the node: %v", err)
		logger.Fatalf("Could not create the node: %v", err)
	}
	log.Printf("Started node at port: %d\n", port)    // logs in the console
	logger.Printf("Started node at port: %d\n", port) // logs in the file

	// Create a new grpc server
	nodeListener := grpc.NewServer()

	if port == 5450 {
		// create primary node
		n = &Node{
			Type:                    Primary,
			maxbid:                  0,
			isAuctionEnded:          false,
			mutex:                   sync.Mutex{},
			logger:                  logger,
			nodes:                   make(map[string]proto.AuctionServiceClient),
			nodeId:                  nodeId,
			nodeTime:                timestamp,
			clientIdMaxBid:			 "",
			lastTimeReceivedPrimary: time.Now(),
			secondaryNodesTimeCheck: make(map[proto.AuctionServiceClient]time.Time),
			portName:				 "localhost:"+strconv.Itoa(port),
			secondaryPIDs:           make(map[string]int),
			leaderPID: 				 os.Getpid(),
		}
		initializeSecondaryNodes(n, logger)
	} else {
		// create secondary node
		n = &Node{
			Type:                    Secondary,
			maxbid:                  0,
			isAuctionEnded:          false,
			mutex:                   sync.Mutex{},
			logger:                  logger,
			nodes:                   make(map[string]proto.AuctionServiceClient),
			nodeId:                  nodeId,
			nodeTime:                timestamp,
			clientIdMaxBid:			 "",
			lastTimeReceivedPrimary: time.Now(), //not used by the primary node
			secondaryNodesTimeCheck: make(map[proto.AuctionServiceClient]time.Time), //not used in the secondary nodes
			portName:				 "localhost:"+strconv.Itoa(port),
			secondaryPIDs:           make(map[string]int),//not used in the secondary nodes
			leaderPID: 				 leaderPIDflag,
		}
	}

	proto.RegisterAuctionServiceServer(nodeListener, n) // register the node to the server

	// Start serving the listener
	go func() {
		if err := nodeListener.Serve(listener); err != nil { // go rotine listens to the server
			log.Fatalf("Failed to serve listener: %v", err)
			logger.Fatalf("Failed to serve listener: %v", err)
		}
	}()

	return n
}

func initializeSecondaryNodes(primary *Node, logger *log.Logger) {
	timestamp := primary.nodeTime // Get the current timestamp from the primary node
	for i := 0; i <= 2; i++ {
		port := 5451 + i // Example: calculate secondary node port

		PID := spawnNode(int32(port), int32(i+1), timestamp, logger, primary.leaderPID) 
		if PID != 0{
			primary.secondaryPIDs["localhost:"+strconv.Itoa(port)] = PID 
		}
	}
}

func spawnNode(port int32, nodeId int32, timestamp time.Time, logger *log.Logger, leaderPID int) (PID int) {
	timestampStr := timestamp.Format(time.RFC3339)                                                                                              // Convert timestamp to string
	cmd := exec.Command("go", "run", "node.go", "-port", strconv.Itoa(int(port)), "-id", strconv.Itoa(int(nodeId)), "-timestamp", timestampStr, "-leaderPID", strconv.Itoa(int(leaderPID))) // Run the node.go file with the specified arguments
	err := cmd.Start()
	if err != nil {
		logger.Printf("Failed to start secondary node %d: %v", nodeId, err)
		return 0
	}

	detachError := cmd.Process.Release()
	if detachError != nil {
		logger.Printf("Failed to detach secondary node %d: %v", nodeId, err)
		return 0
	}

	logger.Printf("Started secondary node %d with PID %d", nodeId, cmd.Process.Pid)
	return cmd.Process.Pid
}

/*********HEARTBEAT*********/

func (s *Node) startheartBeat() {
	heartRateMonitor := time.NewTicker(15 * time.Second) //we create a Ticker which sends a time to the heartRateMonitor channel every 15-second
	for {
		select {
		case <-heartRateMonitor.C: //when the channel
			//s.mutex.Lock()
			primaryNodeConn := s.nodes["localhost:5450"] //we get the primary node
			//s.mutex.Unlock()

			log.Printf("Sending a HeartBeat to the primary node connection %v", s.getPortFromConnection(primaryNodeConn))
			s.logger.Printf("Sending a HeartBeat to the primary node")
			response, err := primaryNodeConn.HeartBeat(context.Background(), &proto.HeartBeatRequest{SecondaryNodePort: s.portName, VerifyFailure: false}) //call to rpc method HeartBeat to be executed in the primary node

			if err != nil {
				log.Printf("Heartbeat failed: %v", err)
				s.logger.Printf("Heartbeat failed: %v", err)
			}

			if response.Ok { //we check whether the response was a HeartBeatResponse
				log.Printf("Heartbeat received from the primary node (leader)")
				s.logger.Printf("Heartbeat received from the primary node (leader)")
				s.lastTimeReceivedPrimary = time.Now() //update the state of the current secondary-node
			}
			/*
				if reflect.TypeOf(response).Elem().Name() == "HeartBeatResponse" { //we check whether the response was a HeartBeatResponse
					s.lastTimeReceivedPrimary = time.Now() //update the state of the current secondary-node
				}
			*/
		}
	}
}

func (s *Node) HeartBeat(ctx context.Context, req *proto.HeartBeatRequest) (*proto.HeartBeatResponse, error) {

	if !req.VerifyFailure {
	s.mutex.Lock()
	s.secondaryNodesTimeCheck[s.getConnectionFromPort(req.SecondaryNodePort)] = time.Now()
	s.mutex.Unlock()
	log.Printf("Heartbeat received from the secondary node: %v", req.SecondaryNodePort)
	s.logger.Printf("Heartbeat received from the secondary node: %v", req.SecondaryNodePort)
	return &proto.HeartBeatResponse{Ok: true}, nil
	}else{
		log.Printf("An secondary node is sending a heartbeat. It is verifying if this node still operational. Informing it back that it still alive...")
		s.logger.Printf("An secondary node is sending a heartbeat. It is verifying if this node still operational. Informing it back that it still alive...")
		return &proto.HeartBeatResponse{Ok: true}, nil
	}
}

func (s *Node) getConnectionFromPort(nodePort string)(connection proto.AuctionServiceClient) {

	for key, conn := range s.nodes {
		if  nodePort == key{return conn};
	}
	return nil
}

func (s *Node) getPortFromConnection(connection proto.AuctionServiceClient)(nodePort string) {

	for key, conn := range s.nodes {
		if  connection == conn{return key};
	}
	return ""
}


func (s *Node) verifySecondaryNodesHeartbeat(){
	for{
		s.mutex.Lock()
		copySNTimecheck := s.secondaryNodesTimeCheck
		s.mutex.Unlock()
		for connection, timeHB := range copySNTimecheck{
			log.Printf("%v, %v", connection, timeHB )
			if time.Now().Sub(timeHB) > 30 * time.Second {

				failPort := s.getPortFromConnection(connection)

				log.Printf("Heartbeat time error found for connection: %v - port: %v", connection, failPort)
				s.logger.Printf("Heartbeat time error found for connection: %v - port: %v", connection, failPort)

				log.Printf("Sending a Check Node Failure request for all other secondary nodes...")
				s.logger.Printf("Sending a Check Node Failure request for all other secondary nodes...")

				var counter int
				for _, conn := range s.nodes{
					if conn != connection {
						response, err := conn.CheckNodeFailure(context.Background(), &proto.CheckNodeFailureRequest{FailureNodePort: failPort})
						if err != nil {
							log.Printf("Error checking node failure gRPC connection %v", err)
							s.logger.Printf("Error checking node failure gRPC connection %v", err)
							continue
						}
						if response.Ok {
							counter ++
						}
					}
				}
				log.Printf("Check Node Failure by secondary nodes completed.")
				s.logger.Printf("Check Node Failure by secondary nodes completed.")
				// If the counter is less than the nodes lenght it means that the node is failing to all of the other nodes as well and need to be respawned.
				if counter < (len(s.nodes) - 1){
					log.Printf("The node cannot be reached even for the secondary nodes. Restarting the node...")
					s.logger.Printf("The node cannot be reached even for the secondary nodes. Restarting the node...")
					s.restartNode(failPort)
				}else{
					log.Printf("The node could be reached for the secondary nodes. Updating the heartbeat time and continuing the process...")
					s.logger.Printf("The node could be reached for the secondary nodes. Updating the heartbeat time and continuing the process...")
						s.mutex.Lock()
						s.secondaryNodesTimeCheck[connection] = time.Now()
						s.mutex.Unlock()
					return
				}

			}
		}
		time.Sleep(11*time.Second)
	}
}

func (s *Node) restartNode(port string){

	log.Printf("Trying to find the process running for port %s", port)
	s.logger.Printf("Trying to find the process running for port %s", port)
	process, err := os.FindProcess(s.secondaryPIDs[port])
	if err != nil {
		log.Printf("Process for the node not found - %v", err)
		s.logger.Printf("Process for the node not found - %v", err)
	}else{
	log.Printf("Killing the PID %d and respawning the node for %s ...", s.secondaryPIDs[port], port)
	s.logger.Printf("Killing the PID %d and respawning the node for %s ...", s.secondaryPIDs[port], port)

	process.Kill()
	}
	splitPort := strings.Split(port ,":")
	portNumber, err := strconv.Atoi(splitPort[1])
	if err != nil {
		log.Printf("Error converting port number to integer - %v", err)
		s.logger.Printf("Error converting port number to integer - %v", err)
		return
	}
	nodeId := portNumber-5450

	PID := spawnNode(int32(portNumber), int32(nodeId), s.nodeTime, s.logger, s.leaderPID)
	if PID != 0{
			log.Printf("Successfully respawned the node %d with a new PID %d", nodeId, PID)
			s.secondaryPIDs["localhost:"+strconv.Itoa(portNumber)] = PID 
		}
		return
}

func (s *Node) CheckNodeFailure(ctx context.Context, req *proto.CheckNodeFailureRequest) (*proto.CheckNodeFailureResponse, error) {

	log.Printf("Received a request from the leader to check a failure in node %v", req.FailureNodePort)
	s.logger.Printf("Received a request from the leader to check a failure in node %v", req.FailureNodePort)
	log.Printf("Sending a Heartbeat for the possible failure in node")
	s.logger.Printf("Sending a Heartbeat for the possible failure in node")
	response, err := s.nodes[req.FailureNodePort].HeartBeat(context.Background(), &proto.HeartBeatRequest{SecondaryNodePort: s.portName, VerifyFailure: true})
		if err != nil {
			log.Printf("Heartbeat failed with error: %v", err)		
			s.logger.Printf("Heartbeat failed with error: %v", err)	
			return &proto.CheckNodeFailureResponse{Ok: false}, err	
		}else if response.Ok{
			log.Printf("Message received from the node, it still operational. Sending a positive feedback to the leader (primary node).")
			s.logger.Printf("Message received from the node, it still operational. Sending a positive feedback to the leader (primary node).")
			return &proto.CheckNodeFailureResponse{Ok: true}, nil
		}else {//unexpected behavior... ( shouldn't happen )
			log.Printf("An unexpected error occurred")
			s.logger.Printf("An unexpected error occurred")
			return &proto.CheckNodeFailureResponse{Ok: false}, nil
		}
} 

/* Restoring the leader... this couldn't be tested based on several problems. But we tried to implement a verification system that when the secondary
 nodes notice that the leader is out they ask each other if it happened for them also. If yes, they call an election and chose the secondary node with
 the lowest port number to respawn the leader in the port 5450... However the process is failing when we kill the leader because all other children processes
 die with it. */

func (s *Node) verifyPrimaryNodeAlive(){
	leaderPort := "localhost:5450"
	for{
		s.mutex.Lock()
		primaryNodeTimeCheck := s.lastTimeReceivedPrimary
		s.mutex.Unlock()
		if time.Now().Sub(primaryNodeTimeCheck) > 30 * time.Second {
				
			leaderConn := s.nodes[leaderPort]

			log.Printf("Time error found with the leader connection (primary node): %v - port: localhost:5450", leaderConn)
			s.logger.Printf("Time error found with the leader connection (primary node): %v - port: localhost:5450", leaderConn)

			log.Printf("Sending a Check Node Failure request for all other secondary nodes...")
			s.logger.Printf("Sending a Check Node Failure request for all other secondary nodes...")

			var counter int
			for _, conn := range s.nodes{
				if conn != leaderConn {
					response, err := conn.CheckNodeFailure(context.Background(), &proto.CheckNodeFailureRequest{FailureNodePort: leaderPort})
					if err != nil {
						log.Printf("Error checking node failure gRPC connection %v", err)
						s.logger.Printf("Error checking node failure gRPC connection %v", err)
						continue
					}
					if response.Ok {
						counter ++
					}
				}
			}
			log.Printf("Check Primary Node (Leader) Failure by secondary nodes completed.")
			s.logger.Printf("Check Primary Node (Leader) Failure by secondary nodes completed.")
			// If the counter is less than the nodes lenght it means that the node is failing to all of the other nodes as well and need to be respawned.
			if counter < (len(s.nodes) - 1){
				log.Printf("The leader cannot be reached. Calling an election to decide which node will respawn the leader...")
				s.logger.Printf("The leader cannot be reached. Calling an election to decide which node will respawn the leader...")
				if(s.callElectionsToRestartLeader(leaderPort)){
				s.restartNode(leaderPort)
				}
			}else{
				log.Printf("The primary node could be reached. Updating the last time received and continuing the process...")
				s.logger.Printf("The primary node could be reached. Updating the last time received and continuing the process...")
					s.mutex.Lock()
					s.lastTimeReceivedPrimary = time.Now()
					s.mutex.Unlock()
				return
			}

		}
		time.Sleep(11*time.Second)
	}
}

func (s *Node) callElectionsToRestartLeader(leaderPort string)(authorized bool){

	splitPort := strings.Split(leaderPort ,":")
	portNumber, err := strconv.Atoi(splitPort[1])
	if err != nil {
		log.Printf("Error converting port number to integer - %v", err)
		s.logger.Printf("Error converting port number to integer - %v", err)
		return
	}

	var counter int
	for nodePort, conn := range s.nodes{
		if nodePort != leaderPort {
			response, err := conn.Election(context.Background(), &proto.ElectionRequest{PortNumber: int32(portNumber)})
			if err != nil {
				log.Printf("Error requesting an election to node %v connection %v: %v", nodePort, conn,  err)
				log.Printf("Error requesting an election to node %v connection %v: %v", nodePort, conn,  err)
				continue
			}
			if response.Authorized {
				counter ++
			}
		}
	}	
	return(counter < (len(s.nodes) - 1))
}


func (s *Node) Election(ctx context.Context, req *proto.ElectionRequest) (*proto.ElectionResponse, error) {

	log.Printf("Received a election request from %v...", req.PortNumber)
	log.Printf("Received a election request from %...", req.PortNumber)
	splitPort := strings.Split(s.portName ,":")
	thisPortNumber, err := strconv.Atoi(splitPort[1])
	if err != nil {
		log.Printf("Error converting port number to integer - %v", err)
		s.logger.Printf("Error converting port number to integer - %v", err)
		return nil, err
	}
	if thisPortNumber < int(req.PortNumber){
		log.Printf("The port number of the request is bigger than the one from this node. Sending back an unauthorized response...")
		s.logger.Printf("The port number of the request is bigger than the one from this node. Sending back an unauthorized response...")
		return &proto.ElectionResponse{Authorized: false}, nil
	}else{
		log.Printf("Election request authorized!")
		s.logger.Printf("Election request authorized!")
		return &proto.ElectionResponse{Authorized: true}, nil
	}
} 

/* IMPORTANT: <<<<<<<<<<<<<<<<<<<<<<<<
 Unfortunately we couldn't make it work here since when we kill the PID all the other nodes are killed with it since they are the children processes...
 We tried to implement in a way to restore (respawn) the leader in the localhost:5450.
 Maybe our approach was wrong for this case...*/

func (s *Node) restartLeader(){

	log.Printf("Trying to find the Leader process running based on the PID %d", s.leaderPID)
	s.logger.Printf("Trying to find the Leader process running based on the PID %d", s.leaderPID)
	process, err := os.FindProcess(s.leaderPID)
	if err != nil {
		log.Printf("Process for the node not found - %v", err)
		s.logger.Printf("Process for the node not found - %v", err)
	}else{
	log.Printf("Killing the PID %d and respawning the leader in the port 5450...", s.leaderPID )
	s.logger.Printf("Killing the PID %d and respawning the leader the port 5450...", s.leaderPID)

	process.Kill()
	}

	PID := spawnNode(int32(5450), int32(11), s.nodeTime, s.logger, s.leaderPID)
	if PID != 0{
			log.Printf("Successfully respawned the leader with a new PID %d", PID)
			s.leaderPID = PID 
		}
		return
}