package frontend

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"
	"container/ring"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	rpc "auction.com/proto"
)
type frontend Struct {
	Leader					int64
	Addr					string
	Id						int64
	ReqQueue				[] //??? what type in a request
}

func (rm *RM) StartListening() {
	lis, err := net.Listen("tcp", rm.Addr) //Listener på denne addr
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
}
grpcServer := grpc.NewServer()
	TRS.RegisterElectionServiceServer(grpcServer, rm) //Dette registrerer noden som en værende en TokenRingServiceServer.

	// Start listening for incoming connections
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func main(){

}