package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"time"

	"github.com/yunuskilicdev/distributedsystems/src/labrpc"
	// Replace with the actual package path where raft.go is located
)

func main() {
	const numNodes = 3

	// Create an array of Raft nodes
	var rafts []*Raft
	for i := 0; i < numNodes; i++ {
		// Set up a listener for RPC
		l, err := net.Listen("tcp", ":0")
		if err != nil {
			log.Fatalf("listen error: %v", err)
		}

		// Create a slice of *labrpc.ClientEnd
		// Assuming this is the correct way to create instances of ClientEnd
		clientEnds := make([]*labrpc.ClientEnd, numNodes)

		// Create a Raft instance
		rf := Make(clientEnds, i, &Persister{}, make(chan ApplyMsg))
		rafts = append(rafts, rf)

		// Register the RPC server
		rpc.Register(rf)
		rpc.HandleHTTP()
		go http.Serve(l, nil)
	}

	// Simulate Raft interaction
	fmt.Println("Starting Raft nodes...")
	time.Sleep(2 * time.Second) // Wait for leader election

	// Test leader election
	fmt.Println("Testing leader election...")
	var leader *Raft
	for _, rf := range rafts {
		if term, isLeader := rf.GetState(); isLeader {
			fmt.Printf("Leader found: Node %d, Term %d\n", rf.me, term)
			leader = rf
			break
		}
	}
	if leader == nil {
		fmt.Println("No leader elected.")
		os.Exit(1)
	}

	// Test log replication
	fmt.Println("Testing log replication...")
	index, term, isLeader := leader.Start("test command")
	if !isLeader {
		fmt.Println("Leader lost during log replication.")
		os.Exit(1)
	}
	fmt.Printf("Log entry replicated at index %d, term %d\n", index, term)

	// Shutdown Raft nodes
	fmt.Println("Shutting down Raft nodes...")
	for _, rf := range rafts {
		rf.Kill()
	}
}
