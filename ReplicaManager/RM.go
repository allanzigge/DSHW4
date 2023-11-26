package main

import (
	"container/ring"
	"flag"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
	"time"
	"os"

	rpc "auction.com/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type RM struct{
	neighbour				string
	nextNeighbour			string
	Leader					string
	electionPeers			map[string]rpc.ElectionServiceClient
	frontEndPeers			map[string]rpc.FrontEndServiceClient
	Addr					string
	leaderIsDead			bool
	isLeader				bool
	FE						rpc.FrontEndServiceClient
	
	mu 						sync.Mutex

	highestBid				*rpc.Amount
	
	rpc.UnimplementedFrontEndServiceServer
	rpc.UnimplementedElectionServiceServer
}

var AddrFlag = flag.String("addr", "default", "RM IP address")

func main(){
	flag.Parse()

	file, err := os.OpenFile("../log_file.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Set log output to the file
	log.SetOutput(file)
	
	rm := &RM{
		electionPeers: make(map[string]rpc.ElectionServiceClient),
		frontEndPeers: make(map[string]rpc.FrontEndServiceClient),
		Addr: *AddrFlag,
		Leader: "localhost:50051",
		leaderIsDead: false,
		isLeader: false,
		highestBid: &rpc.Amount{Amount: 0, Id: "-100"}}
	
	lis, err := net.Listen("tcp", rm.Addr) //Listener på denne addr
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	
	rm.Start()

	go rm.SetupServer(lis)

	// go rm.SetupGrpcFEServer(lis)
	
	go rm.CheckHeartbeat() // start listning to the Leader
	
	select{}
}


func (rm *RM) SetupServer(lis net.Listener){
	grpcRMServer := grpc.NewServer()
	rpc.RegisterElectionServiceServer(grpcRMServer, rm) //This registres the Node as a ReplicaManager.
	rpc.RegisterFrontEndServiceServer(grpcRMServer, rm)

	// Start listening for incoming connections
	if err := grpcRMServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func (rm *RM) Start() {
	if rm.Leader == rm.Addr {
		rm.mu.Lock()
		rm.isLeader = true
		rm.mu.Unlock()
	}
	// rm.electionPeers = make(map[string]rpc.ElectionServiceClient) //Instantierer nodens map over peers.
	
	// Hardcoded list af replica manager
	hardcodedIPsList :=  []string{"localhost:50051", "localhost:50052", "localhost:50053"}
	// Manke a new ring
	hardcodedIPsRing := ring.New(len(hardcodedIPsList))

	rm.connectToFE()
	
	
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
	for  i:= 0; i < (2*hardcodedIPsRing.Len()); i++ {
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

func (rm *RM) connectToFE(){
	conn, err := grpc.Dial("localhost:50069", grpc.WithInsecure()) //Dial op connection to the address
	if err != nil {
		log.Printf(rm.Addr, ": Unable to connect to le FrontEnd: %v", err)
	return
	}
	rm.FE = rpc.NewFrontEndServiceClient(conn)
}

func (rm *RM) SetupClient(addr string) {
	if addr == rm.Addr {
		return
	}
	
	conn, err := grpc.Dial(addr, grpc.WithInsecure()) //Dial op connection to the address
	if err != nil {
		log.Printf(rm.Addr, ": Unable to connect to %s: %v", addr, err)
		return
	}
	
	rm.mu.Lock()
	rm.electionPeers[addr] = rpc.NewElectionServiceClient(conn) //Create a new tokenringclient and map it with its address.
	rm.frontEndPeers[addr] = rpc.NewFrontEndServiceClient(conn) 
	log.Println("RM(", rm.Addr ,") has connected to Node(",addr, ")") //Print that the connection happened.
	rm.mu.Unlock()
}

func (rm *RM) CheckHeartbeat() {
	for {
		if !rm.leaderIsDead { //If it is known that leader is dead, dont try heartbeats
			time.Sleep(time.Duration(2) * time.Second)
			if rm.Leader != rm.Addr { // If i am not the leader
				_, err := rm.electionPeers[rm.Leader].Heartbeat(context.Background(), &rpc.Addr{Addr:rm.Addr})
				if err != nil {
					log.Printf(rm.Addr, ":Leader is gone!: ", err)
					rm.mu.Lock()
					rm.leaderIsDead = true //Stop rm from sending heartbeatchecks to leader until new leader is sat
					rm.mu.Unlock()
					if rm.neighbour != rm.Leader {
						
						stream, err1 := rm.electionPeers[rm.neighbour].Election(context.Background())
						if err1 != nil {
						log.Printf(rm.Addr, ":Neighbor is gone:", err1)
							newstream, err2 := rm.electionPeers[rm.nextNeighbour].Election(context.Background())
							if err2 != nil {
							log.Printf(rm.Addr, ":nextneighbor is gone:", err2)
							continue
							}
						newstream.Send(&rpc.Addr{Addr: rm.Addr})
						newstream.CloseSend()
						continue
						}
					stream.Send(&rpc.Addr{Addr: rm.Addr})
					stream.CloseSend()
					continue
					} else {
						log.Printf(rm.Addr, ": neighbor is leader, trying next neighbour")
						stream, err2 := rm.electionPeers[rm.nextNeighbour].Election(context.Background())
							if err2 != nil {
							log.Printf(rm.Addr, ":nextNeighbor is gone:", err2)
							continue
							}
						stream.Send(&rpc.Addr{Addr: rm.Addr})
						stream.CloseSend()
					    continue
					}
					
				}
			}
		}
		
	}
	
}

func (rm *RM) Heartbeat(ctx context.Context, addr *rpc.Addr) (*rpc.Ack, error) {
	return &rpc.Ack{Status: 200}, nil
}

func (rm *RM) SetLeader(ctx context.Context, LeaderAddr *rpc.Addr) (*rpc.Ack, error){
	rm.Leader = LeaderAddr.Addr
	if rm.Leader == rm.Addr {
		rm.mu.Lock()
		rm.isLeader = true
		rm.mu.Unlock()
	}
	rm.mu.Lock()
	rm.leaderIsDead = false //Stop rm from sending heartbeatchecks to leader until new leader is sat
	rm.mu.Unlock()
	log.Println(rm.Addr, ": RM ", rm.Leader , "Is the new Leader")
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
			rm.Leader = electedAddr;
			if rm.Leader == rm.Addr {
				rm.mu.Lock()
				rm.isLeader = true
				rm.mu.Unlock()
			}
			rm.leaderIsDead = false;
			for _, peers := range rm.electionPeers{ //Broadcast new leader to all RM
				peers.SetLeader(context.Background(), &rpc.Addr{Addr: electedAddr})
			}
			rm.FE.SetLeader(context.Background(), &rpc.Addr{Addr: electedAddr})
			return stream.SendAndClose(&rpc.Ack{
				Status: 200})
			}
		}
		
	partitionAddresses = append(partitionAddresses, rm.Addr)
	newStream, err := rm.electionPeers[rm.neighbour].Election(context.Background())
	if err != nil {
		log.Println(rm.Addr, ": My nieghbor is gone"); //Nieghbour er død
		newNewStream, err := rm.electionPeers[rm.nextNeighbour].Election(context.Background())
		if err != nil {
			log.Println(rm.Addr, ": my nextNeighbor is gone"); //NextNeighbour er død..
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
	}
	stream.CloseSend()
}

func (rm *RM) Bid(ctx context.Context, bidAmount *rpc.Amount) (*rpc.Outcome, error) {
	if rm.isLeader {
		for _, peer := range rm.frontEndPeers {
			_, err := peer.Bid(context.Background(), bidAmount)
			if err != nil {
				log.Println(rm.Addr, ": Error calling bid on peer: ", peer)
				continue
			}
		}
	}
	log.Println(rm.Addr, ": set new highest bid to: ", bidAmount.Amount)
	return rm.compareBids(bidAmount), nil
}

func (rm *RM) Result(ctx context.Context, empty *rpc.Empty) (*rpc.BidResult, error) {
	return &rpc.BidResult{Result: "Client " + rm.highestBid.Id + " with amount: " + strconv.FormatInt(rm.highestBid.Amount, 10)}, nil
}

func (rm *RM) compareBids(newBid *rpc.Amount) *rpc.Outcome {
	if rm.highestBid.Amount < newBid.Amount {
		rm.highestBid = newBid
		return &rpc.Outcome{Outcome: "Succes!"}
	} else {
		return &rpc.Outcome{Outcome: "Fail..."}
	}
}