package main

import (
	"bufio"
	"context"
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

var server rpc.FrontEndServiceClient

func main() {
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
		log.Println("only ints!")
		return
	}

	_, err2 := server.Bid(context.Background(), &rpc.Amount{Amount: bidAmount})
	if err2 != nil {
		log.Println("Error calling Bid on frontend: ", err2)
		return
	}

}

func runResult() {

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
	log.Println("Connection established to FrontEnd")

	server = rpc.NewFrontEndServiceClient(conn)

}
