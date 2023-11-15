package main

import (
	"fmt"
	"log"
	"net"
	"container/ring"
	"sync"
	"flag"
	"time"
	"io"
	
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	rpc "auction.com/proto"
)

type RM struct{
	neighbour				string
	nextNeighbour			string
	Leader					string
	Peers					map[string]rpc.ElectionServiceClient
	Addr					string
	leaderIsDead			bool
	
	mu 						sync.Mutex
	
	rpc.UnimplementedElectionServiceServer
}

var AddrFlag = flag.String("addr", "default", "RM IP address")

func main(){
	flag.Parse()
	
	rm := &RM{
		Peers: make(map[string]rpc.ElectionServiceClient),
		Addr: *AddrFlag,
		Leader: "localhost:50051",
		leaderIsDead: false,}
	
	lis, err := net.Listen("tcp", rm.Addr) //Listener på denne addr
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	
	rm.Start()
	
	go rm.CheckHeartbeat() // start listning to the 
	
	grpcServer := grpc.NewServer()
	rpc.RegisterElectionServiceServer(grpcServer, rm) //Dette registrerer noden som en værende en TokenRingServiceServer.
	
	// Start listening for incoming connections
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func (rm *RM) Start() {
	// rm.Peers = make(map[string]rpc.ElectionServiceClient) //Instantierer nodens map over peers.
	
	// //Hardcoded list af replica manager
	hardcodedIPsList :=  []string{"localhost:50051", "localhost:50052", "localhost:50053"}
	// Manke a new ring
	hardcodedIPsRing := ring.New(len(hardcodedIPsList))
	
	// Initialize the ring with some addresses
	for i := 0; i < len(hardcodedIPsList); i++ {
		hardcodedIPsRing.Value = hardcodedIPsList[i]
		hardcodedIPsRing = hardcodedIPsRing.Next()
	}
	
	hardcodedIPsRing.Do(func(p any) { // iterates through the ring and calls SetupClient on every address
		if p != nil {
			rm.SetupClient(p.(string))
		}
	})
	
	
	// Run through each of the IPs
	for  i:= 0; i< hardcodedIPsRing.Len(); i++ {
		// Skip the current node
		if hardcodedIPsRing.Value.(string) == rm.Addr {// If the addr is the addr of the node, save n & nn then break
			
			hardcodedIPsRing = hardcodedIPsRing.Next()
			rm.neighbour = hardcodedIPsRing.Value.(string)
			
			hardcodedIPsRing = hardcodedIPsRing.Next()
			rm.nextNeighbour = hardcodedIPsRing.Value.(string)
			break 
		} else {
			hardcodedIPsRing = hardcodedIPsRing.Next()
		}
		
	}
	// go rm.StartListening() //Go routine med kald til "server" funktionaliteten.
	
}

func (rm *RM) SetupClient(addr string) {
	if addr == rm.Addr {
		return
	}
	
	conn, err := grpc.Dial(addr, grpc.WithInsecure()) //Dial op connection to the address
	if err != nil {
		log.Printf("Unable to connect to %s: %v", addr, err)
		return
	}
	
	rm.mu.Lock()
	rm.Peers[addr] = rpc.NewElectionServiceClient(conn) //Create a new tokenringclient and map it with its address.
	fmt.Println("RM(", rm.Addr ,") has connected to Node(",addr, ")") //Print that the connection happened.
	rm.mu.Unlock()
}

func (rm *RM) CheckHeartbeat() {
	for {
		if !rm.leaderIsDead { //If it is known that leader is dead, dont try heartbeats
			time.Sleep(time.Duration(2) * time.Second)
			if rm.Leader != rm.Addr { // If i am not the leader
				_, err := rm.Peers[rm.Leader].Heartbeat(context.Background(), &rpc.Addr{Addr:rm.Addr})
				if err != nil {
					log.Printf(rm.neighbour)
					log.Printf(rm.nextNeighbour)
					log.Printf("Leader is gone!:", err)
					rm.mu.Lock()
					rm.leaderIsDead = true //Stop rm from sending heartbeatchecks to leader until new leader is sat
					rm.mu.Unlock()
					if rm.neighbour != rm.Leader {
						stream, err1 := rm.Peers[rm.neighbour].Election(context.Background())
						if err1 != nil {
							newstream, err2 := rm.Peers[rm.nextNeighbour].Election(context.Background())
							if err2 != nil {
							log.Printf("nextneighbor is gone:", err2)
							continue
							}
						log.Printf("Neighbor is gone:", err1)
						newstream.Send(&rpc.Addr{Addr: rm.Addr})
						newstream.CloseSend()
						continue
						}
					stream.Send(&rpc.Addr{Addr: rm.Addr})
					stream.CloseSend()
					continue
					} else {
						log.Printf("neighbor is leader, trying next leader")
						stream, err2 := rm.Peers[rm.nextNeighbour].Election(context.Background())
							if err2 != nil {
							log.Printf("nextNeighbor is gone:", err2)
							continue
							}
						stream.Send(&rpc.Addr{Addr: rm.Addr})
						stream.CloseSend()
					    continue
					}
					
				}
				fmt.Println("Heartbeat recieved")
			}
		}
		
	}
	
}

func (rm *RM) Heartbeat(ctx context.Context, addr *rpc.Addr) (*rpc.Ack, error) {
	fmt.Println("RM addr:", addr.Addr, "checked for Leader heartbeat")
	return &rpc.Ack{Status: 200}, nil
}

func (rm *RM) SetLeader(ctx context.Context, LeaderAddr *rpc.Addr) (*rpc.Ack, error){
	rm.Leader = LeaderAddr.Addr
	rm.mu.Lock()
	rm.leaderIsDead = false //Stop rm from sending heartbeatchecks to leader until new leader is sat
	rm.mu.Unlock()
	fmt.Println("RM ", rm.Leader , "Is the new Leader")
	return &rpc.Ack{Status: 200}, nil
}

func (rm *RM) Election(stream rpc.ElectionService_ElectionServer) error{
	rm.mu.Lock()
					rm.leaderIsDead = true //Stop rm from sending heartbeatchecks to leader until new leader is sat
					rm.mu.Unlock()
	
	var partitionAddresses []string
	
	for {
		partitionAddr, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		partitionAddresses = append(partitionAddresses, partitionAddr.Addr)
	}
	
	for _, partitionAddr := range partitionAddresses{ 
		if partitionAddr == rm.Addr{ //If my adress is in the list -> End election & find result
			electedAddr := rm.FindNewLeader(partitionAddresses)
			for _, peers := range rm.Peers{ //Broadcast new leader to all RM
				peers.SetLeader(context.Background(), &rpc.Addr{Addr: electedAddr})
			}
			return stream.SendAndClose(&rpc.Ack{
				Status: 200})
			}
		}
		
	partitionAddresses = append(partitionAddresses, rm.Addr)
	
	newStream, err := rm.Peers[rm.neighbour].Election(context.Background())
	if err != nil {
		fmt.Println(err); //Nieghbour er død
		newNewStream, err := rm.Peers[rm.nextNeighbour].Election(context.Background())
		if err != nil {
			fmt.Println(err); //NextNeighbour er død.. Big slick lick fuck lort problem
		} else {
			rm.SendPartitionAddresses(newNewStream, partitionAddresses)
		}
	} else {
		rm.SendPartitionAddresses(newStream, partitionAddresses)
	}
	
	return stream.SendAndClose(&rpc.Ack{
		Status: 200})
}
		
func (rm *RM) FindNewLeader(partitionAddresses []string) string{
		electedAddr := ""
		for _, candidateAddr := range partitionAddresses{ //The elected addr should be the "biggest" address
		if(candidateAddr > electedAddr){
			electedAddr = candidateAddr
		}
	}
	return electedAddr
}

func (rm *RM) SendPartitionAddresses (stream rpc.ElectionService_ElectionClient, partitionAddresses []string) {
	for _, partitionAddr := range partitionAddresses{
		stream.Send(&rpc.Addr{Addr: partitionAddr})
		stream.CloseSend()
	}
}