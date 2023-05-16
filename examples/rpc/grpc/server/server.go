package main

import (
	"context"
	"log"
	"net"

	pb "github.com/hashicorp/go-plugin/examples/rpc/grpc/proto"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedServiceServer
}

func (s *server) Ping(ctx context.Context, in *pb.PingRequest) (*pb.PingResponse, error) {
	return &pb.PingResponse{Data: in.Data}, nil
}

func main() {
	lis, err := net.Listen("unix", "../grpc.sock")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer(
		grpc.MaxRecvMsgSize(64 * 1024 * 1024),
	)
	pb.RegisterServiceServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
