package main

import (
	"log"
	"net"

	"github.com/cloudwego/shmipc-go"
	"github.com/hashicorp/go-plugin/examples/rpc/shmipc/shared"
	"github.com/hashicorp/go-plugin/shmipc/rpc"
)

type Service struct{}

func (kv *Service) Ping(req *shared.PingRequest, resp *shared.PingResponse) error {
	resp.Data = req.Data
	return nil
}

func main() {
	lis, err := net.ListenUnix("unix", &net.UnixAddr{Name: "../shm.sock", Net: "unix"})
	if err != nil {
		log.Fatal(err)
	}
	defer lis.Close()

	server := rpc.NewServer(shmipc.DefaultConfig())

	if err := server.Register(&Service{}); err != nil {
		log.Fatal(err)
	}

	server.Accept(lis)
}
