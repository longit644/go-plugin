package main

import (
	"context"
	"log"
	"math"
	"os"
	"path"
	"time"

	pb "github.com/hashicorp/go-plugin/examples/rpc/grpc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}

	// Set up a connection to the server.
	conn, err := grpc.Dial("unix:///"+path.Join(cwd, "../grpc.sock"),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(math.MaxInt32)),
		grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(math.MaxInt32)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewServiceClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	_, err = client.Ping(ctx, &pb.PingRequest{Data: []byte("data")})
	if err != nil {
		log.Printf("could not greet: %v", err)
	}
}
