package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	pb "replication/grpc" // replace with your actual package path
	"strconv"

	"google.golang.org/grpc"
)

var clientId = flag.Int("id", 7, "client id")

func main() {

	flag.Parse()

	// Open a log file for the peer node
	logDir := "logs/"  // dict where logs will be stored 
	
	// create new log dict if not exist
	if err := os.MkdirAll(logDir, 0700); err != nil { // read write and execute permission
		log.Fatalf("Error creating log directory: %v", err)
	}

	// Construct the log file path using the process ID
	logFilePath := fmt.Sprintf(logDir+"client_%v_log.txt", *clientId)
	
	// Create the log file
	logFile, err := os.Create(logFilePath)

	// If there's an error creating the log file, log the error and exit
	if err != nil {
		log.Fatalf("Error creating log file: %v", err)
	}
	defer logFile.Close() // cloes when main existst

	// Create a logger for writing log messages 
	logger := log.New(logFile, "", log.LstdFlags)

    // Connect to the primary node
    conn, err := grpc.Dial("localhost:5450", grpc.WithInsecure(), grpc.WithBlock())
    if err != nil {
        log.Fatalf("did not connect: %v", err)
		logger.Fatalf("did not connect: %v", err)
    }
    defer conn.Close()
    client := pb.NewAuctionServiceClient(conn) // create a client for the grpc-service defined in proto

    // Scanner for user input
    scanner := bufio.NewScanner(os.Stdin)
    log.Println("Enter your bid amount (integer):")
    for scanner.Scan() {
        input := scanner.Text() // Scans a line of input

        // Convert input to integer
        bidAmount, err := strconv.Atoi(input)

        if err == nil {
			// Make a bid
			requestBid(client, int32(bidAmount), logger)
			continue
        } 
		if input == "status" {
			requestResult(client, logger)
            continue
    	} else if err != nil {
			log.Printf("Invalid input, please enter an integer: %v", err)
			continue
		}
	}

    if err := scanner.Err(); err != nil {
        log.Fatalf("Error reading input: %v", err)
		logger.Fatalf("Error reading input: %v", err)
    }

    // You might want to move getting the auction result outside the for loop
    // depending on how you want your client to behave
}


func requestBid(client pb.AuctionServiceClient, amount int32, logger *log.Logger) {
	response, err := client.Result(context.Background(), &pb.ResultRequest{})
	if err != nil {
		log.Fatalf("could not get result: %v", err)
		logger.Fatalf("could not get result: %v", err)
	}
	if response.IsAuctionEnded == true {
		log.Printf("Auction is ended! Winner is %s with bid %d", response.Winner, response.MaxBid)
		logger.Printf("Auction is ended! Winner is %s with bid %d", response.Winner, response.MaxBid)
		return
	}else if response.MaxBid >= amount {
		log.Printf("Your bid (%d) is not higher than the current max bid (%d)", amount, response.MaxBid)
		logger.Printf("Your bid (%d) is not higher than the current max bid (%d)", amount, response.MaxBid)
		return
	}else {
		log.Printf("Sending a bid request with the amount (%d)", amount)
		logger.Printf("Sending a bid request with the amount (%d)", amount)
		response, err := client.Bid(context.Background(), &pb.BidRequest{Bid: amount, ClientId: "client"+string(*clientId)})
	if err != nil {
		log.Fatalf("could not bid: %v", err)
		logger.Fatalf("could not bid: %v", err)
	}
	log.Printf("Node Response: %s", response.Ack)
	logger.Printf("Node Response: %s", response.Ack)
	}
}
	func requestResult(client pb.AuctionServiceClient, logger *log.Logger) {
		response, err := client.Result(context.Background(), &pb.ResultRequest{})

		if err != nil {
			log.Fatalf("could not get result: %v", err)
			logger.Fatalf("could not get result: %v", err)
		}
		
		if response.IsAuctionEnded == true {
			log.Printf("Auction is ended! Winner is %s with bid %d", response.Winner, response.MaxBid)
			logger.Printf("Auction is ended! Winner is %s with bid %d", response.Winner, response.MaxBid)
			return
		} else {
			log.Printf("The current max bid of the auction is %d", response.MaxBid)
			logger.Printf("The current max bid of the auction is %d", response.MaxBid)
			return
		}
	}	



