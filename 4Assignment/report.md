# Mandatory Handin 4 — Distributed Mutual Exclusion — Report

************************Introduction************************

In this report we shall 1) present our solution and describe why our system meets the system requirements and 2) discuss our algorithm using examples from our logs  (see ‘logs…).

## **1) Solution to requirements**

### **************Peer.proto**************

To present our solution, we find it suitable to start with our `peer.proto` file since it can be seen as the ‘skeleton’ for our solution. Firstly, it consists of a service which holds three rpc calls:

```protobuf
service CriticalSection {
	rpc RequestToEnterCriticalSection (RequestMessage) returns (ResponseMessage);
	rpc LeaveCriticalSection (RequestMessage) returns (ResponseMessage);
	rpc ValidateRequestEnterCriticalSection (ValidationMessage) returns (ResponseMessage);
}
```

We go through the implementation of each method below but for now it shall simply serve as an initial overview. Besides the methods above, we also created three different message types in our `peer.proto`:

```protobuf
message RequestMessage {
    int32 timestamp = 1;
    int32 process_id = 2;
    string message = 3;
}

message ResponseMessage {
    bool permission = 1;
}

message ValidationMessage {
    int32 timestamp = 1;
}
```

In short, we create three different messages: `RequestMessage` which consist of a timestamp, process_id and a message. It will be used when a node wishes to enter its critical section and send a request to all other peer nodes  expecting a permission. In addition to this, the `ResponseMessage` is simply a bool that depends on the outcome of `RequestToEnterCriticalSection` and `ValidateRequestEnterCriticalSection`. Lastly, we create the `ValidationMessage` which just has a timestamp. We explain the use of the validation-message as we go through our `peer2peer.go` below.

### **************Peer2peer.go and solution to requirements**************

*R1**:** Implement a system with a set of peer nodes, and a Critical Section, that represents a sensitive system operation. Any node can at any time decide it wants access to the Critical Section. Critical section in this exercise is emulated, for example by a print statement, or writing to a shared file on the network.*

Firstly, we create the set of peer nodes by making a slice of size 11 and consisting of strings.

Next we iterate through the slice and create local adresses for every number in the slice.

```protobuf
nodes := make([]string, 11)

	for i := 0; i <= 10; i++ {
		address := "localhost:" + fmt.Sprintf("%d", 5450+i)
		nodes[i] = address
```

Next, we run `startPeerListener()` which in short 1) starts a listener on the given port, 2) creates a new grpc server, 3) creates a new node with all of attributes within `PeerNode`.

Next, we loop through all of nodes and create a go-routine for each. Within every go-routine we start, we establish a connection to the grpc-server at the specified node. In addition, within every go-routine we create a client for the grpc-service defined in our proto-file. Lastly for every go-routine, we append the newly created client into `thisNode.clients` with mutual exclusion to ensure a happens-before-relation. We use `select{}` to always keep the connection alive while running the service. Otherwise, once the go-routine finishes it will kill the  thread and then the adresses dissapears with the port. 

Next, we create a scanner that reads strings from standard input. We set `thisNode.requesting` to `true` to let all other nodes know that the current one is requesting to enter. Next, we check whether there is a another node within the critical section. If yes, we keep monitoring, if no, we run `tryToEnterCriticalSection()`.  In short terms, this method makes our current node send a request (rpc-call) to every node that is ‘live’ with a timestamp from the exact moment the message was entered. 

```go
for _, client := range thisNode.clients {
		// request for every node in the list
		response, err := client.RequestToEnterCriticalSection(context.Background(), &proto.RequestMessage{
			Timestamp: int32(thisNode.timestamp),
			ProcessId: int32(*processID), // Ensure processID is not nil and is a pointer
			Message: message,
		...
```

Every rpc-call to `RequestToEnterCriticalSection()` is validated by a rpc-call to `ValidateRequestEnterCriticalSection()` which compares the timestamp of the node requesting to enter the critical section to all the other timestamps of the nodes within the system. Also, `RequestToEnterCriticalSection()` also makes use of locks to ensure that only node at a time can call the method. Lastly, we set the `p.extNodeInCS` to true and `p.requesting` to inform the other nodes that a node is currently in the critical section. Last of all, if of all the calls to `RequestToEnterCriticalSection()` returns true, `tryToEnterCriticalSection()` returns true, the node breaks out of its loop, we lock the mutex and the node can finally enter the  critical section. Once the message has been sent, the node uses `LeaveCriticalSection()` to leave the critical section, we set `thisNode.requesting = false` to inform the other nodes that the current one is no longer requesting to enter.

### 2) Provide a discussion of your algorithm, using examples from your logs (Technical Requirement)

*Our thoughts on what we would do better*:

**1. Implementing a Queue for Message Handling**

Currently we handling one message at a time. It would be better to queuing messages for each node. Currently our system is limited to processing a single message per terminal, which can be a bottleneck in scenarios where multiple nodes need to communicate simultaneously or when a node needs to handle multiple tasks in succession. For example a scenario where three nodes, A, B, and C, are trying to access a shared resource simultaneously. With the current setup, if node A is sending a message we can’t send another message for a different node. 

Implementing a message queue for each node can address this issue. When a node sends a message, it's added to the node's respective queue. The system then processes these messages one after the other. This not only ensures orderly processing of requests but also improves the system's ability to handle concurrent requests.

**2. Dynamic Node Management through Service Discovery**

Currently we hard-coded the number of nodes in our system. It's not scalable or flexible, especially in dynamic environments where the number of nodes can change frequently.

Imagine a distributed system, where virtual machines (nodes) are frequently added or removed based on the demand. Our hard coded number would require constant manual updates, making it impractical and error-prone.

A solution would be implementing service discovery for automatic node management. In this approach, when a new node is added to the network, it registers itself with a shared resource (like a registry or a directory service). Similarly, when a node leaves or fails, it is automatically deregistered. This dynamic approach allows the system to scale efficiently and adapt to changes in the network topology without manual intervention.

### Provide a link to a Git repo with your source code in the report

- Link to our Repository:
    
    [https://github.itu.dk/edtr/distributedsystems/tree/main/4Assignment](https://github.itu.dk/edtr/distributedsystems/tree/main/4Assignment)
    

### Include system logs, that document the requirements are met, in the appendix of your report

- Please look at the log files in our linked repository:
[https://github.itu.dk/edtr/distributedsystems/tree/4_assignment/4Assignment/logs](https://github.itu.dk/edtr/distributedsystems/tree/main/4Assignment/logs)