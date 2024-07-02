package main

import (
	"bufio"
	proto "chittyChat/grpc"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"

	"google.golang.org/grpc"
)

type VectorClock map[string]int64

var channelName = flag.String("channel", "chittychat", "Channel name for chatting")
var serverAddress = flag.String("server", "localhost:5455", "Server address")
var userName = flag.String("name", "User", "User/Client name")

// Log file for chat messages
var logFile *os.File
var logger *log.Logger


var clockMutex sync.Mutex


var clientClock = make(VectorClock)

func main() {
	//parse the flags to get the arguments
    flag.Parse()

	clientClock[*userName] = 0

	// Open a log file for the client
    logDir := "logs/"
	if err := os.MkdirAll(logDir, 0700); err != nil {
		log.Fatalf("Error creating log directory: %v", err)
	}
	logFilePath := logDir + *userName + "_" + *channelName + ".txt"
	logFile, err := os.Create(logFilePath)
	if err != nil {
		log.Fatalf("Error creating log file: %v", err)
	}
	defer logFile.Close()


    // Create a logger for writing chat messages to the log file
    logger = log.New(logFile, "", log.LstdFlags)

		logger.Printf("------- CLIENT LOG: %s --------", *userName)
		log.Printf("------- CLIENT LOG: %s --------", *userName)

    //optional arguments for dial grpc server
    var opts []grpc.DialOption
	opts = append(opts, grpc.WithBlock(), grpc.WithInsecure())

    // Connect to the gRPC server
    conn, err := grpc.Dial(*serverAddress, opts...)
    if err != nil {
        logger.Fatalf("Could not connect to server: %v", err)
    }
    defer conn.Close()

    context := context.Background()

    client := proto.NewChittyChatServiceClient(conn)

    // call joinChannel method to join and receive messages (go routine implemented inside the method [parallel])

    go joinChannel(context, client)

    // Read messages from the console and call publish
    scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
        message := scanner.Text()

        //verify message length
        if len(message) > 128 {
        logger.Println("The message has more than 128 characters. Please enter a shorter one...")
        } else{
        go publish(context, client, scanner.Text())
        }
    }
}



func joinChannel(context context.Context, client proto.ChittyChatServiceClient) {

    //define the channel based on the grpc stub definition
	channel := proto.Channel{Name: *channelName, SendersName: *userName}
    // Create a undirectional streaming connection for join in the channel and receive messages (Server-side streaming)
	stream, err := client.JoinChannel(context, &channel)
	if err != nil {
		logger.Fatalf("client.JoinChannel(context, &channel) throws: %v", err)
	}

	clientClock[*userName]++

	logger.Printf("%v send a request to join the channel %v. Waiting for server side confirmation... [Vector Clock -> %v]\n",*userName, *channelName, formatVectorClockToString(clientClock))

    //creates the channel to receive messages from server side
	waitc := make(chan struct{})

    //go routine to receive the messages in parallel execution
	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				close(waitc)
				return
			}
			if err != nil {
				logger.Fatalf("Failed to receive message from channel %v. Error: %v",channelName, err)
			}

			if *userName != in.Sender {
				clockMutex.Lock()
				for key, value := range in.VectorClock {
                    if clientClock[key] < value {
                        clientClock[key] = value
                    }
                }
				clientClock[*userName]++

				logger.Printf("(%v)[Vector Clock-> %v]: %v \n", in.Sender, formatVectorClockToString(clientClock), in.Message)
				log.Printf("(%v): %v \n", in.Sender, in.Message)
				clockMutex.Unlock()
			}
		}
	}()

	<-waitc
}

func publish(context context.Context, client proto.ChittyChatServiceClient, message string) {
	// Create a bidirectional streaming connection for send the message and receive the aknowledge if the message was sent.
    stream, err := client.Publish(context)
	if err != nil {
		logger.Fatalf("Failed to send a message: %v", err)
	}

	clockMutex.Lock()
	clientClock[*userName]++
	tempClock := clientClock
	clockMutex.Unlock()

	msg := proto.Message{
		Channel: &proto.Channel{
			Name:        *channelName,
			SendersName: *userName},
		Message: message,
		Sender:  *userName,
		VectorClock: tempClock,
	}
	stream.Send(&msg)

	logger.Printf("Client requested to publish a message to channel %v [Vector Clock-> %v]: %v\n",*channelName, formatVectorClockToString(clientClock), message)



	ack, err := stream.CloseAndRecv()
    if err != nil {
        logger.Fatalf("Failed to receive a confirmation message: %v", err)
    }
    logger.Printf("Message sent: %v \n", ack)
}


func formatVectorClockToString(vector VectorClock) string {
    result := ""
    for key, value := range vector {
        result += fmt.Sprintf("%s: %d, ", key, value)
    }
    result = strings.TrimSuffix(result, ", ")
    return result
}