# DSHW5

Setting up with 3 Servers, 2 Clients, and the Frontend:

Running the servers:

1. Open the first three terminals and navigate to the ReplicaManager directory:

cd /ReplicaManager

2. In the first terminal, start the first server which is set as the Leader and creates the log-file.txt:
go run RM.go -addr localhost:50051

3. In the second terminal, start the second server:
go run RM.go -addr localhost:50052

4. In the third terminal, start the third server:
go run RM.go -addr localhost:50053

Running the Frontend:

1. Open the fourth terminal and navigate to the FrontEnd directory:
cd /FrontEnd

2. Start the Frontend:
go run FrontEnd.go

Running the Clients:

1. Open the fifth and sixth terminals and navigate to the Client directory:
cd /Client

2. In the first terminal, create a client with ID 1:
go run Client.go -id 1

3. In the second terminal, create a client with ID 2:
go run Client.go -id 2

Bidding in the Auction:
You have 120 seconds to bid in the auction.
In the client terminals, use the following commands:

1. To place a bid type:
bid

The terminal will prompt: "Type bid-amount." Enter the desired bid amount as an integer.

2. To check the auction status type:
result

If the auction is ongoing, the leading bidder and the current bid amount will be displayed. If the auction is over, the winner and the final bid will be shown.
