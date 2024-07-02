// tcp_client.go
package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	// connect to the server adress and port
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	
	// Send SYN to server - conection is made the client send SYN message
	conn.Write([]byte("SYN"))

	buffer := make([]byte, 1024)
	len, err := conn.Read(buffer) // read the buffer
	if err != nil {
		fmt.Println("Error reading:", err)
		return
	}
 
	// Expecting SYN-ACK from the server
	if string(buffer[:len]) == "SYN-ACK" {
		time.Sleep(1 * time.Second)
		conn.Write([]byte("ACK")) // writes back ACK to the server
	}
}