package main

import (
	proto "chittyChat/grpc"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"

	// "time"
	"sync"
)

//creating a structure for the vector clock
type VectorClock map[string]int64

// Struct that will be used to represent the Server.
type Server struct {
	proto.UnimplementedChittyChatServiceServer
	mutex sync.Mutex
	name string
	port int
	channel map[string][]chan *proto.Message
	// this link is to map the channel used to each client
	linkChannelClient map[chan *proto.Message]string
	clock VectorClock
}

var logFile *os.File
var logger *log.Logger

// Used to get the user-defined port for the server from the command line
var port = flag.Int("port", 0, "server port number")


func main() {
	// Get the port from the command line when the server is run
	flag.Parse()

	// Open a log file for the client
    logDir := "logs/"
	if err := os.MkdirAll(logDir, 0700); err != nil {
		log.Fatalf("Error creating log directory: %v", err)
	}
	logFilePath := logDir + "server.txt"
	logFile, err := os.Create(logFilePath)
	if err != nil {
		log.Fatalf("Error creating log file: %v", err)
	}
	defer logFile.Close()


    // Create a logger for writing chat messages to the log file
    logger = log.New(logFile, "", log.LstdFlags)

	// Create a server struct
	server := &Server{
		name: "chitty chat server",
		port: *port,
		channel: make(map[string][]chan *proto.Message),
		linkChannelClient: make(map[chan *proto.Message]string),
		clock: make(VectorClock),
	}

	logger.Printf("------- SERVER LOG: %s --------", server.name)
	log.Printf("------- SERVER LOG: %s --------", server.name)

	// Start the server
	go startServer(server)

	// Keep the server running until it is manually quit
	select {}

}

func startServer(server *Server) {

	// Create a new grpc server

	// Make the server listen at the given port (convert int port to string)
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(server.port))
	//initialize server vector clock as 0
	server.clock[server.name] = 0

	if err != nil {
		logger.Fatalf("Could not create the server %v", err)
	}
	logger.Printf("Started server at port: %d\n", server.port)
	log.Printf("Started server at port: %d\n", server.port)

	// Create a new grpc server
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	// Register the grpc server and serve its listener
	proto.RegisterChittyChatServiceServer(grpcServer, server)
	serveError := grpcServer.Serve(listener)
	if serveError != nil {
		logger.Fatalf("Could not serve listener")
	}
}


func (server *Server) Broadcast(msg *proto.Message) error {

	go func() {
		logger.Printf("Broadcasting the message sent by %v to all clients in channel %v  -> %v \n",msg.Sender, msg.Channel.Name, msg.Message)
		log.Printf("Broadcasting the message sent by %v to all clients in channel %v \n",msg.Sender, msg.Channel.Name,)
		streams := server.channel[msg.Channel.Name]
		for _, msgChan := range streams {
			if msg.Sender != server.linkChannelClient[msgChan]{
				server.clock[server.name]++
				// Increment the server vector clock [sending pack]
				msg.VectorClock = server.clock
				msgChan <- msg
				time.Sleep(time.Second/10)
			}
		}
	}()

	return nil

}

func (server *Server) Publish(msgStream proto.ChittyChatService_PublishServer) error{
	msg, err := msgStream.Recv()

	if err == io.EOF {
		return nil
	}

	if err != nil {
		return err
	}

	
    server.mutex.Lock()
	// for every item inside the received vector compare if it is bigger than the current one in the server vector
	for name, value := range msg.VectorClock {
    	if server.clock[name] < value {
        	server.clock[name] = value
    	}
	}

	// Increment the server vector clock [received pack]
    server.clock[server.name]++

	logger.Printf("Server received a Publish request from %v to channel %v [Vector Clock-> %v]:. Message: %v", msg.Sender, msg.Channel.Name, formatVectorClockToString(server.clock), msg.Message)
	log.Printf("Server received a Publish request from %v to channel %v [Vector Clock-> %v]", msg.Sender, msg.Channel.Name, formatVectorClockToString(server.clock))

	msg.VectorClock = server.clock

	go server.Broadcast(msg)

	server.mutex.Unlock()

	ack := proto.MessageAck{Status: "Sent \\o/"}
	msgStream.SendAndClose(&ack)

	return nil
}

func (server *Server) JoinChannel(channel *proto.Channel, msgStream proto.ChittyChatService_JoinChannelServer) error {

	msgChannel := make(chan *proto.Message)
	server.mutex.Lock()

	server.channel[channel.Name] = append(server.channel[channel.Name], msgChannel)
	server.linkChannelClient[msgChannel] = channel.SendersName

	//initialize the clock for the client that is joining the channel
	server.clock[channel.SendersName] = 1
	server.clock[server.name]++

	logger.Printf("%v requested to join in the channel %v [Vector Clock-> %v] \n", channel.SendersName, channel.Name, formatVectorClockToString(server.clock))
	log.Printf("%v requested to join in the channel %v [Vector Clock-> %v] \n", channel.SendersName, channel.Name, formatVectorClockToString(server.clock))

	//force one more because a welcome message will be sent to the user that joined the channel
	server.clock[server.name]++

	// Send a welcome message to the client who joined
	welcomeMsg := &proto.Message{
		Channel: channel,
		Message: "Participant " + channel.SendersName + " joined Chitty-Chat at [Vector Clock -> " + formatVectorClockToString(server.clock) + "] \n" + "Welcome to the channel " + channel.Name + " " + channel.SendersName + "!",
		Sender:  "Very Friendly Server",
		VectorClock: server.clock,
	}
	if err := msgStream.Send(welcomeMsg); err != nil {
		logger.Printf("Failed to send welcome message to %v: %v", channel.SendersName, err)
	}

	go server.Broadcast(
		&proto.Message{
			Channel: channel,
			Message: "Participant " + channel.SendersName + " joined Chitty-Chat at [Vector Clock -> " + formatVectorClockToString(server.clock) + "]",
			Sender:  channel.SendersName,
			VectorClock: server.clock,
	})

	server.mutex.Unlock()


	for {
		select {
		case <-msgStream.Context().Done():
			server.mutex.Lock()
			msgStreams := server.channel[channel.Name]
			for i, stream := range msgStreams {
				if stream == msgChannel {
					//remove the channel from the list/slice of active channels 
					server.channel[channel.Name] = append(server.channel[channel.Name][:i], server.channel[channel.Name][i+1:]...)
					//close the stream since the client left the chat
					close(msgChannel)
					//add one to the vector clock for receiving the info that the client is leaving
					server.clock[server.name]++
					//Question: Do I need to add one to the client that left? I didn't because it didn't sent a message...
					logger.Printf("%v left the channel %v [Vector Clock -> %v]\n", channel.SendersName, channel.Name, formatVectorClockToString(server.clock))
					log.Printf("%v left the channel %v [Vector Clock -> %v]\n", channel.SendersName, channel.Name, formatVectorClockToString(server.clock))
					go server.Broadcast(
						&proto.Message{
							Channel: channel,
							Message: "Participant " + channel.SendersName + " left Chitty-Chat at Vector Clock ->" + formatVectorClockToString(server.clock),
							Sender:  channel.SendersName,
							VectorClock: server.clock,
						})
					server.mutex.Unlock()
				return nil
				}
		}	
		case msg := <-msgChannel:
			logger.Printf("Sending the message to %s [Vector Clock -> %v]: %v \n", server.linkChannelClient[msgChannel], formatVectorClockToString(msg.VectorClock) ,msg.Message)
			log.Printf("Sending the message to %s [Vector Clock -> %v]: %v \n", server.linkChannelClient[msgChannel], formatVectorClockToString(msg.VectorClock) ,msg.Message)
			msgStream.Send(msg)
		}
	}
}

func formatVectorClockToString(vector VectorClock) string {
    result := ""
    for key, value := range vector {
        result += fmt.Sprintf("%s: %d, ", key, value)
    }
    result = strings.TrimSuffix(result, ", ")
    return result
}