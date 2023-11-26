package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	rpc "auction.com/proto"
	// "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var ClientIdFlag = flag.String("id", "default", "Client id")

var server rpc.FrontEndServiceClient

func main() {
	flag.Parse()

	file, err := os.OpenFile("../log_file.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Set log output to the file
	log.SetOutput(file)

	ConnectToServer()

	go CheckForCommands()

	select{}

}

func CheckForCommands(){
	reader := bufio.NewReader(os.Stdin)

	for{
		var command,_ = reader.ReadString('\n')
		command = strings.TrimRight(command, "\n")

		switch command {
		case "result": runResult()
		case "bid": runBid()
		}
	}
	

}

func runBid() {
	fmt.Println("Type bid-amount:")
	reader := bufio.NewReader(os.Stdin)

	var bid,_ = reader.ReadString('\n')
	bid = strings.TrimRight(bid, "\n")

	bidAmount,err1 := strconv.ParseInt(bid, 10, 64)
	if err1 != nil {
		fmt.Println("only ints!")
		return
	}

	log.Println("Client", *ClientIdFlag, "tries to bid: ", bidAmount)

	Outcome, err2 := server.Bid(context.Background(), &rpc.Amount{Amount: bidAmount, Id: *ClientIdFlag})
	if err2 != nil {
		log.Println("Error calling Bid on frontend: ", err2)
		return
	}
	log.Println("Client", *ClientIdFlag , ":" ,Outcome.Outcome)

}

func runResult() {
	bidResult, _ := server.Result(context.Background(), &rpc.Empty{})
	log.Println("Client", *ClientIdFlag, ":", bidResult)
}

func ConnectToServer(){
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	conn, err := grpc.Dial("localhost:50069", opts...)
	if err != nil {
		log.Println("Fail to dial: %v", err)
		return
	}
	log.Println("Client",*ClientIdFlag ,": Connection established to FrontEnd")

	server = rpc.NewFrontEndServiceClient(conn)

}
