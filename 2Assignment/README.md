# Mandatory hand-in 2

For this mandatory hand-in-2 we included in total 4 files according to the three diffrent levels. The number infornt of the file represents the implementation of the level. We anser the five question based on our implementatioins.

#### a) What are packages in your implementation? What data structure do you use to transmit data and meta-data?

*(1)* <br>
We only use standard go libraries: fmt, math/rand and time. The main data structure i use for tranmitting data and meta-data is the 'packet' which represent a network packet.

*(2)* <br>
For the second question we use fmt for I/O operations net as networking libary whioch provides basic networking functions. In this question it is used to create and manage the TCP server. For transmitting the data between client and server we're using buffers of size 1024 bytes.

*(3)* <br>
We only use standard go libraries: fmt, math/rand and time.
The main data structure we use for tranmitting data in handshake *3* and meta-data is
the 'packet' which represent a network packet.


#### b) Does your implementation use threads or processes? Why is it not realistic to use threads?

*(1, 2, 3)* <br>
All implementations do not use threads or processes. We're using goroutines which is an effective and optimized way to handle network processing in go when compared with threads or processes. The main benefit of a Go program is that thousands or even millions of goroutines can be spawned efficiently and therefore are better for networks than typical operating system threads.

#### c) In case the network changes the order in which messages are delivered, how would you handle message re-ordering?

*(1)* <br>
Each packet contains a timestamp. The receiver/server would then be able to order messages 
based on when they were created.

*(2)* <br>
In this implementation we didn't include a way to re-order the messages. But in reality we could use sequencs numbers for example to re-order the messages or duplicates.

*(3)* <br>
If the network changes the order in which messages are delivered, we would address message re-ordering in the middleware by implementing a mechanism using flags or conditions to analyze and reorder the packets accordingly.

#### d) In case messages can be delayed or lost, how does your implementation handle message loss?

*(3)* <br>
If messages can potentially be delayed or lost, our implementation addresses message loss by incorporating a middleware that utilizes channels to forward messages. Within this middleware, we can implement routines to identify lost packets and reorder them based on specific criteria. As a demonstration, we've included a straightforward example of message forwarding with simulated delays.

#### e) Why is the 3-way handshake important?

*(1, 2, 3)* <br>
It helps ensure that data is being transferred in the correct order. By making sequence numbers, the client and the server can agree on how to order packets in a correct and fair manner.
