# <span style="color: lightyellow;">How to run this project</span>

<span style="color: lightblue;">**Starting the peers:**</span> In a terminal application insert this command to run the first peer node:
&nbsp;

```
go run peer2peer.go -port 5454 -id 1
```

Afterwards, do the same procedure in different terminals to start the other two peer nodes.

```
go run peer2peer.go -port 5455 -id 2
```

```
go run peer2peer.go -port 5456 -id 3
```

You can specify the ids as you want for each peer node and choose a port in a range between 5450 - 5460. Due to time limitations we didn't implemented a handler for different ports (outside this range) or to check different ids. So, when you are specifying the ids make sure that you are inserting different ones to make the process more clear.

&nbsp;

With all the terminals running check the message: "Network is ready for use!". Then, you can type a message in the terminal and see that node trying to enter in the critical section. Inside the critical section it will print your message and freeze for 5 seconds. In the meanwhile you can try to type another message in the other nodes to check the rules being respected and when the previous node left the critical section the one you typed before should enter (fairness). Also, it will keep try to get the resource and at some point the node that is trying will have access to.
