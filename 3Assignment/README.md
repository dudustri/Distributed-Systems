# <span style="color: lightyellow;">How to run this project</span>

<span style="color: lightblue;">**Starting the server:**</span> In a terminal application insert this command to run the server with your chosen port (i.e. 5455):
&nbsp;

```
go run server/server.go -port 5455
```

<span style="color: lightblue;">**Starting the clients:**</span> After that open 3 new terminals in vscode or your in your favorite shell and invoke a client for each one using this command:

```
go run client/client.go -name client_1 -server "localhost:5455"
```

&nbsp;
In this example we are setting the client name as client_1 and passing the server address as being the localhost:5455. If you changed the port you should pass the new port in the arguments instead.
There is an extra flag that can be passed called -channel "channel_name" which will make the client join this channel name in the server. The default name will be always set as chittychat. You don't need to use this flag, however, do it if you want but, be aware that if different clients joined different channels they cannot comunicate with each other.

&nbsp;
In sequence you can create another two clients for each terminal you had opened. You can use the commands below as an example:

```
go run client/client.go -name client_2 -server "localhost:5455"
```

```
go run client/client.go -name client_3 -server "localhost:5455"
```

<span style="color: lightblue;">**Sending messages:**</span> If you received the confirmation message that the client joined the server it means that it is working and you can send messages to the chat.
With all the terminals opened you can send a message from one client to the other ones just writing in the console a message below 128 characters. It should be send to all other clients (you can check all of it in real time with you split your screen to fit all the terminals at same time).

<span style="color: lightblue;">**Leaving the chat / closing the client:**</span>
To leave the chat you can just use the command `ctrl + c` to interrupt/terminate the client and then leave the chat. All other clients should receive the message that the specific client left.

&nbsp;

# <span style="color: lightyellow;">Docker Execution</span>

<span style="color: salmon;"> **The containers are running but we couldn't enable communication between the them...** </span>

Running the server and provided clients with docker run the command:

```
./run_docker-chat_example.sh
```

To stop the containers, use:

```
docker-compose down
```

To list the containers:

```
docker ps
```

<span style="color: salmon;">**Not working**</span> - To execute a something in the terminal:

```
docker exec -it **container_name_or_id** sh -> i.e. docker exec -it client-ionic sh
```
