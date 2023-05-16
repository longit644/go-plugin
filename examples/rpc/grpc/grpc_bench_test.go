package shmipc

import (
	"context"
	"math"
	"os"
	"path"
	"testing"

	pb "github.com/hashicorp/go-plugin/examples/rpc/grpc/proto"
	rpcshared "github.com/hashicorp/go-plugin/examples/rpc/shared"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Benchmark(b *testing.B) {
	cwd, err := os.Getwd()
	if err != nil {
		b.Errorf("error = %v, want nil", err)
	}

	// Set up a connection to the server.
	conn, err := grpc.Dial("unix:///"+path.Join(cwd, "./grpc.sock"),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(math.MaxInt32)),
		grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(math.MaxInt32)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		b.Errorf("error = %v, want nil", err)
	}
	defer conn.Close()
	client := pb.NewServiceClient(conn)
	if err != nil {
		b.Errorf("error = %v, want nil", err)
	}

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
				_, err := client.Ping(context.Background(), &pb.PingRequest{Data: bm.data})
				if err != nil {
					b.Errorf("error = %v, want nil", err)
				}
			}
		})
	}
}
