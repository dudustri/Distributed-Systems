// tcp_server.go
package main

import (
	"fmt" // I/O Operations
	"net" // networking libary
)

func main() {
	listener, err := net.Listen("tcp", "localhost:8080") // server listening for incoming tcp connections on port localhost:8080
	if err != nil { // if error while starting the listener - if not nil jmp
		panic(err)
	}
	defer listener.Close() // ensures closing and freeing the resources

	for {
		conn, err := listener.Accept() // each itteration the server wait for a new client connection
		if err != nil {
			fmt.Println("Connection failed:", err)
			continue
		}
		// allowing multiple client connections concurrently
		go handleRequest(conn) // if client connects the server spwans a new goroutine to handle the handshake process
	}
}

func handleRequest(conn net.Conn) {
	/*
	Function for the handshake process
	*/
	
	defer conn.Close()

	// buffer size 1024 bytes from the server side to read incoming data from the client
	buffer := make([]byte, 1024)
	len, err := conn.Read(buffer) // len = number of bytes read
	if err != nil {
		fmt.Println("Error reading:", err)
		return
	}

	// Assuming the client sends "SYN" to initiate the handshake
	if string(buffer[:len]) == "SYN" { // SYN from the client as initial message - slice the buffer read the ASCII Code
		conn.Write([]byte("SYN-ACK")) // sever sends SYN-ACK back to the client 
	}

	len, err = conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading:", err)
		return
	}

	// Expecting ACK from client to finish handshake
	if string(buffer[:len]) == "ACK" {  // ACK from the client to complete three way handshake
		fmt.Println("Handshake complete!") 
	}
}