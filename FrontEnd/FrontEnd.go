package main

import (
	"log"
	"net"

	rpc "auction.com/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type FE struct {
	Leader string
	//ReqQueue				[]

	rpc.UnimplementedFrontEndServiceServer
}

func main() {
	fe := &FE{
		Leader: "localhost:50051",
	}

	lis, err := net.Listen("tcp", "localhost:50069") //Listener på denne addr
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	rpc.RegisterFrontEndServiceServer(grpcServer, fe) //Dette registrerer noden som en værende en TokenRingServiceServer.

	// Start listening for incoming connections
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

	for {
		log.Println("leader ", fe.Leader)
	}

}


func (fe *FE) SetLeader(ctx context.Context, LeaderAddr *rpc.Addr) (*rpc.Ack, error) {
	fe.Leader = LeaderAddr.Addr
	return &rpc.Ack{Status: 200}, nil
}

//rpc Bid(amount) returns (Ack) {}
// rpc Result(Empty) returns (result) {}

func (fe *FE) Bid(ctx context.Context, bidAmount *rpc.Amount) (*rpc.Ack, error) {
	
	return &rpc.Ack{Status: 200}, nil
}

func (fe *FE) Result(ctx context.Context, empty *rpc.Empty) (*rpc.BidResult, error) {
	
	return &rpc.BidResult{Result: "hold kæft"}, nil
}

/*
func (fe *FE) SetupClient(addr string) {
	if addr == fe.Addr {
		return
	}
}
*/
