package main

import (
	"log"
	"time"

	"github.com/cloudwego/shmipc-go"
	"github.com/hashicorp/go-plugin/examples/rpc/shmipc/shared"
	"github.com/hashicorp/go-plugin/shmipc/rpc"
)

func main() {
	conf := shmipc.DefaultSessionManagerConfig()
	conf.Address = "../shm.sock"
	conf.Network = "unix"
	conf.MemMapType = shmipc.MemMapTypeMemFd
	conf.SessionNum = 1
	conf.InitializeTimeout = 100 * time.Second

	client, err := rpc.NewClient(conf)
	if err != nil {
		log.Fatal(err)
	}

	var reply shared.PingResponse
	err = client.Call("Service.Ping", &shared.PingRequest{Data: []byte("data")}, &reply)
	if err != nil {
		log.Fatal(err)
	}
}
