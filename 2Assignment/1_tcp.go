package main

import (
	"fmt"
	"math/rand"
	"time"
)

type packet struct {
	syn int
	ack int
	timeStamp time.Time
}

func genRanSeqNum(min, max int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max-min+1) + min
}

func client(clientChannel chan packet, serverChannel chan packet) {
	fmt.Println("Client is sending to server...")
	time.Sleep(2 * time.Second)

	//create and send
	randomSeqNum := genRanSeqNum(1, 1000)
	timeStamp := time.Now()
	synPacket := packet{syn: randomSeqNum, timeStamp: timeStamp}
	clientChannel <- synPacket

	//wait to receive SYN-ACK
	synAckPacket := <-serverChannel
	fmt.Println("Client has received SYN-ACK packet from the server:", "{ syn:", synAckPacket.syn, "ack:", synAckPacket.ack, "}", synAckPacket.timeStamp.Format("2006-01-02 15:04:05"))
	time.Sleep(2 * time.Second)

	//send ack-packet to acknowledge the servers seq number
	ackPacket := packet{syn: synAckPacket.ack + 1, ack: synAckPacket.syn + 1, timeStamp: timeStamp}
	clientChannel <- ackPacket
}

func server(clientChannel chan packet, serverChannel chan packet) {
	fmt.Println("Server is receiving from client...")
	time.Sleep(2 * time.Second)

	//wait to receive from client
	synPacket := <-clientChannel
	fmt.Println("Server received: ", "{ syn:", synPacket.syn, "ack:", synPacket.ack, "}", synPacket.timeStamp.Format("2006-01-02 15:04:05"))
	time.Sleep(2 * time.Second)

	//create and send SYN-ACK to client
	serverSyn := genRanSeqNum(1, 1000)
	serverSynAck := packet{syn: serverSyn, ack: synPacket.syn + 1, timeStamp: synPacket.timeStamp}
	serverChannel <- serverSynAck
	fmt.Println("Server sent SYN-ACK packet to client:", "{ syn:", serverSynAck.syn, "ack:", serverSynAck.ack, "}", serverSynAck.timeStamp.Format("2006-01-02 15:04:05"))
	time.Sleep(2 * time.Second)

	//wait to receive from client
	ackPacket := <-clientChannel
	fmt.Println("Server received ACK packet from client:", "{ syn:", ackPacket.syn, "ack:", ackPacket.ack, "}", ackPacket.timeStamp.Format("2006-01-02 15:04:05"))
	time.Sleep(2 * time.Second)
}

func main() {
	clientChannel := make(chan packet)
	serverChannel := make(chan packet)

	go client(clientChannel, serverChannel)
	go server(clientChannel, serverChannel)

	// Sleep for a while to allow the client-server communication to complete
	// In a real application, you would need a more robust synchronization mechanism
	// to ensure proper communication.
	time.Sleep(15 * time.Second)
}