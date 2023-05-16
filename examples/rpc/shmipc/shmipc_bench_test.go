package shmipc

import (
	"testing"
	"time"

	"github.com/cloudwego/shmipc-go"
	rpcshared "github.com/hashicorp/go-plugin/examples/rpc/shared"
	"github.com/hashicorp/go-plugin/examples/rpc/shmipc/shared"
	"github.com/hashicorp/go-plugin/shmipc/rpc"
)

func Benchmark(b *testing.B) {
	conf := shmipc.DefaultSessionManagerConfig()
	conf.Address = "./shm.sock"
	conf.Network = "unix"
	conf.MemMapType = shmipc.MemMapTypeMemFd
	conf.SessionNum = 1
	conf.InitializeTimeout = 100 * time.Second

	client, err := rpc.NewClient(conf)
	if err != nil {
		b.Errorf("error = %v, want nil", err)
	}

	var reply shared.PingResponse

	benchmarks := []struct {
		name string
		data []byte
	}{
		{
			name: "64B",
			data: rpcshared.RandBytes(64),
		},
		{
			name: "512B",
			data: rpcshared.RandBytes(512),
		},
		{
			name: "1KiB",
			data: rpcshared.RandBytes(1024),
		},
		{
			name: "4KiB",
			data: rpcshared.RandBytes(4 * 1024),
		},
		{
			name: "16KiB",
			data: rpcshared.RandBytes(16 * 1024),
		},
		{
			name: "32KiB",
			data: rpcshared.RandBytes(32 * 1024),
		},
		{
			name: "64KiB",
			data: rpcshared.RandBytes(64 * 1024),
		},
		{
			name: "256KiB",
			data: rpcshared.RandBytes(256 * 1024),
		},
		{
			name: "512KiB",
			data: rpcshared.RandBytes(512 * 1024),
		},
		{
			name: "1MiB",
			data: rpcshared.RandBytes(1024 * 1024),
		},
		{
			name: "2MiB",
			data: rpcshared.RandBytes(2 * 1024 * 1024),
		},
		{
			name: "4MiB",
			data: rpcshared.RandBytes(4 * 1024 * 1024),
		},
		{
			name: "8MiB",
			data: rpcshared.RandBytes(8 * 1024 * 1024),
		},
		{
			name: "16MiB",
			data: rpcshared.RandBytes(16 * 1024 * 1024),
		},
		{
			name: "32MiB",
			data: rpcshared.RandBytes(32 * 1024 * 1024),
		},
	}
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				err = client.Call("Service.Ping", &shared.PingRequest{Data: bm.data}, &reply)
				if err != nil {
					b.Errorf("error = %v, want nil", err)
				}
				reply.Data = nil
			}
		})
	}
}
