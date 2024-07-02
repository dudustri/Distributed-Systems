To launch the auction system, run both node and client applications using the commands in the terminal. 

**Starting the Nodes**

Initiating the Primary Node:
Open a terminal window and execute the following command to start the primary node of the auction system:
```
go run node.go -port 5450 -id 11
```

This command initializes the primary node on port 5450 with an identification number of 11. Upon running the above command for the primary node, the system is programmed to automatically generate all necessary secondary nodes. 

**Starting a Client**

Launching a Client Instance:
Open a second terminal window and input the following command to run a client:
```
go run client.go -id 20
```
This command starts a client instance with an identification number of 20.

**Making Bids and Checking Auction Status**

Once the client is running, it is possible to interact with the auction system in two ways:

1. Placing a Bid:
To place a bid, simply type a numeric value (representing the bid amount) and press Enter. For example, to bid $20, you would type:
```
20
```
1. Checking Auction Status: If you want to check the current status of the auction, type status and press Enter. This command fetches the latest auction information, including the highest bid and the leading bidder (if any):
```
status
```