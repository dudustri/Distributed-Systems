package main

import (
    "fmt"
    "net"
    "sync"
    "time"
    "math/rand"
    "os"
)

func main() {
    fmt.Println("------------------------\nstarting simulation <>\n------------------------")
    time.Sleep(time.Second*3/2)

    //chosing an address for the handshake simulation server
    serverAddress := "localhost:8082"

    //create a new simulated middleware
    middleware := MiddlewareForwarder()

    //defer makes sure that for all the cases that main is executed it will execut the function in the final (similar as finally in java)
    defer middleware.Stop()

    // Start the middleware in a goroutine
    go middleware.Start()

    // Create a WaitGroup to synchronize client termination
    var clientServerSync sync.WaitGroup

    // Create pointer to WaitGroup
    synchronizeClientServer := &clientServerSync

    // Start the server
    synchronizeClientServer.Add(1)
    go startServer(serverAddress, middleware, synchronizeClientServer) //, synchronizeServer)

    // Start the client and add 
    synchronizeClientServer.Add(1)
    go startClient(serverAddress, middleware, synchronizeClientServer)

    // Wait for the client to complete its processing
    synchronizeClientServer.Wait()

    fmt.Println("------------------------\nsimulation finished <>\n------------------------")

    //coming back to the terminal
    os.Exit(0)
}

func startServer(address string, middleware *Middleware, barrier *sync.WaitGroup) {
    //make sure that the barrier will not block the execution if some error occurs
    defer barrier.Done()

    //listen to the specified server port using net package
    listener, errorListening := net.Listen("tcp", address)
    if errorListening != nil {
        fmt.Println("Error listening:", errorListening.Error())
        return
    }
    // This defer will close the listener before finish this function or in case of error
    defer listener.Close()

    fmt.Println("Server listening on", address)

    for {
        connection, errorToListen := listener.Accept()
        if errorToListen != nil {
            fmt.Println("Error accepting connection:", errorToListen.Error())
            continue
        }
        barrier.Add(1)
        go processRequests(connection, middleware, barrier)

        break
        
    }

    listener.Close()
}

func startClient(address string, middleware *Middleware, barrierClient *sync.WaitGroup) {

    // make sure that the barrier will not create a deadlock if some error happens
    defer barrierClient.Done()

    connection, errorToConnect := net.Dial("tcp", address)
    if errorToConnect != nil {
        fmt.Println("Error connecting to server:", errorToConnect.Error())
        return
    }
    // make sure to close the connection before remove the function from the stack
    defer connection.Close()

    time.Sleep(time.Second)
    fmt.Println("Client connected to server")

    // Simulate the TCP handshake from client side!
    _, errorHandshakeSendSync := connection.Write([]byte("SYN"))
    if errorHandshakeSendSync != nil {
        fmt.Println("Error sending SYN:", errorHandshakeSendSync.Error())
        return
    }

    response := make([]byte, 6)
    _, errorHandshakeReceiveSyncAckno := connection.Read(response)
    if errorHandshakeReceiveSyncAckno != nil {
        fmt.Println("Error receiving SYNACK:", errorHandshakeReceiveSyncAckno.Error())
        return
    }

    // process the response from the server
    if string(response) == "SYNACK" {

        time.Sleep(time.Second)
        fmt.Println("TCP Handshake - SYN ACK - Client Side Completed")

        time.Sleep(3*time.Second)
        fmt.Println("Writting ACK in the connection to the server")

        // Send "ACK" to acknowledge the handshake
        _, errorHandshakeSendAckno := connection.Write([]byte("ACK"))
        if errorHandshakeSendAckno != nil {
            fmt.Println("Error sending ACK:", errorHandshakeSendAckno.Error())
            return
        }

        // Send a message through the forwarder after the handshake
        time.Sleep(3*time.Second)
        fmt.Println("Sending a friendly message to the lovely buffed server <>")
        time.Sleep(time.Second)
        middleware.SendMessageToServer("Hello from the client after handshake. Can you send me a big Hi!?")
    }
}


func processRequests(connection net.Conn, middleware *Middleware, barrierHandleRequest *sync.WaitGroup) {
    // make sure that the connection is closed
    defer connection.Close()
    defer barrierHandleRequest.Done()

    request := make([]byte, 3)
    _, errorReadRequest := connection.Read(request)
    if errorReadRequest != nil {
        fmt.Println("Error reading request:", errorReadRequest.Error())
        return
    }

    if string(request) == "SYN" {
        time.Sleep(time.Second)
        fmt.Println("Server received the Synchronize request...")
        time.Sleep(time.Second)
        fmt.Println("Server sending back the Synchronize-Acknowledge response")

        // syn ack response simulation
        _, errorHandshakeSendSyncAckno := connection.Write([]byte("SYNACK"))
        if errorHandshakeSendSyncAckno != nil {
            fmt.Println("Error sending SYN ACK:", errorHandshakeSendSyncAckno.Error())
            return
        }

        // The client is expected to send an "ACK" to complete the handshake
        response := make([]byte, 3)
        _, errorHandShakeReceiveAckno := connection.Read(response)
        if errorHandShakeReceiveAckno != nil {
            fmt.Println("Error receiving ACK:", errorHandShakeReceiveAckno.Error())
            return
        }
        if string(response) == "ACK" {

            time.Sleep(time.Second)
            fmt.Println("Successful recepection of Acknowledge from Client")
            time.Sleep(time.Second)
            fmt.Println("TCP Handshake complete - server side")
            time.Sleep(time.Second)

            // Receive a message from the Client and send the response
            for {
                message := <-middleware.inputServerChannel
                if( message != ""){
                    time.Sleep(time.Second/2)
                    fmt.Println("Answering the client request!")
                    middleware.AnswerClient("Hiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiii")
                    break
                    }
            }

            time.Sleep(time.Second*2)
            fmt.Println("Shutting down the server...")
            time.Sleep(time.Second)
            fmt.Println("in ... 3 \\o/")
            time.Sleep(time.Second)
            fmt.Println(".. 2 _o/")
            time.Sleep(time.Second)
            fmt.Println(". 1 _o_")
            time.Sleep(time.Second)
            fmt.Println("Done!")
            
        }
    }
}

type Middleware struct {
    inputClientChannel chan string
    outputClientChannel chan string
    inputServerChannel chan string
    outputServerChannel chan string
    stopChannel chan struct{}
    midBarrier sync.WaitGroup
}

func MiddlewareForwarder() *Middleware {
    return &Middleware{
        inputClientChannel: make(chan string),
        outputClientChannel: make(chan string),
        inputServerChannel: make(chan string),
        outputServerChannel: make(chan string),
        stopChannel: make(chan struct{}),
    }
}

func (middleware *Middleware) Start() {
    
    middleware.midBarrier.Add(1)
    //make sure to not create a deadlock
    defer middleware.midBarrier.Done()

    for {
        select {

        // IMPORTANT: The lost package was not implemented
        // However here would be possible to add a case for lost packages based on some criteria!!!

        case msg := <-middleware.outputClientChannel:
            // Simulate message delay with random duration (0-2 seconds)
            delay := time.Duration(rand.Intn(2000)) * time.Millisecond
            time.Sleep(delay)
            fmt.Printf("Forwarding message: %s\n", msg)
            middleware.inputServerChannel <- msg

        case msg := <-middleware.outputServerChannel:
            // Simulate message delay with random duration (0-2 seconds)
            delay := time.Duration(rand.Intn(2000)) * time.Millisecond
            time.Sleep(delay)
            fmt.Printf("Forwarding message: %s\n", msg)
            middleware.inputClientChannel <- msg

        case <-middleware.stopChannel:
            return
        }
    }
}

func (middleware *Middleware) SendMessageToServer(msg string) {
    middleware.outputClientChannel <- msg
}

func (middleware *Middleware) AnswerClient(msg string) {
    middleware.outputServerChannel <- msg
}


func (middleware *Middleware) Stop() {
    close(middleware.stopChannel)
    middleware.midBarrier.Wait()
    close(middleware.inputClientChannel)
    close(middleware.outputClientChannel)
    close(middleware.inputServerChannel)
    close(middleware.outputServerChannel)
}