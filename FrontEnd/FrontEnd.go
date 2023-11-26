package main

import (
	"log"
	"net"
	"time"
	"os"

	rpc "auction.com/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type FE struct {
	Leader 					string
	LeaderConnection		rpc.FrontEndServiceClient
	auctionActive			bool
	//ReqQueue				[]

	rpc.UnimplementedFrontEndServiceServer
}

func main() {
	fe := &FE{
		Leader: "localhost:50051",
		LeaderConnection: nil,
		auctionActive: true,
	}

	file, err := os.OpenFile("../log_file.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Set log output to the file
	log.SetOutput(file)

	fe.SetupServer(fe.Leader)

	lis, err := net.Listen("tcp", "localhost:50069") //Listener på denne addr
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	rpc.RegisterFrontEndServiceServer(grpcServer, fe) //Dette registrerer noden som en værende en TokenRingServiceServer.

	go fe.AuctionTimer()
	// Start listening for incoming connections
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

	

}


func (fe *FE) SetLeader(ctx context.Context, LeaderAddr *rpc.Addr) (*rpc.Ack, error) {
	fe.Leader = LeaderAddr.Addr
	fe.SetupServer(LeaderAddr.Addr)
	return &rpc.Ack{Status: 200}, nil
}

//rpc Bid(amount) returns (Ack) {}
// rpc Result(Empty) returns (result) {}

func (fe *FE) Bid(ctx context.Context, bidAmount *rpc.Amount) (*rpc.Outcome, error) {
	if fe.auctionActive{
		Outcome, err := fe.LeaderConnection.Bid(context.Background(), &rpc.Amount{Amount: bidAmount.Amount, Id: bidAmount.Id})
		if err != nil {
			log.Println("Error calling Bid on leader: ", err)
			return &rpc.Outcome{Outcome: "Exception"}, err
		}
		return Outcome, nil
	} else {
		return &rpc.Outcome{Outcome: "Exception: Auction FINISHED"}, nil
	}
}

func (fe *FE) Result(ctx context.Context, empty *rpc.Empty) (*rpc.BidResult, error) {
	results, err := fe.LeaderConnection.Result(context.Background(), &rpc.Empty{})
	if err != nil {
		log.Println("error calling result on leader: ", err)
	}
	if fe.auctionActive {
		return &rpc.BidResult{Result: " Highest bidder: " + results.Result}, nil
	} else {
		return &rpc.BidResult{Result: " Auction is over! Winner is" + results.Result}, nil
	}
	
}


func (fe *FE) SetupServer(addr string) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Println("Unable to connect to server: ", addr)
		return
	}
	fe.LeaderConnection = rpc.NewFrontEndServiceClient(conn)
	log.Println("Frontend has connected to leader ", addr)
}

func (fe *FE) AuctionTimer() {
	time.Sleep(time.Second*120)
	fe.auctionActive = false;
}

