package rpc

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	"github.com/cloudwego/shmipc-go"
)

// ServerError represents an error that has been returned from
// the remote side of the RPC connection.
type ServerError string

func (e ServerError) Error() string {
	return string(e)
}

var ErrShutdown = errors.New("connection is shut down")

// If set, print log statements for internal and I/O errors.
var debugLog = false

// Client represents an RPC Client.
// There may be multiple outstanding Calls associated
// with a single Client, and a Client may be used by
// multiple goroutines simultaneously.
type Client struct {
	smgr *shmipc.SessionManager

	mutex    sync.Mutex // protects following
	closing  bool       // user has called Close
	shutdown bool       // server has told us to stop
}

// NewClient returns a new Client to handle requests to the
// set of services at the other end of the connection.
// It adds a buffer to the write side of the connection so
// the header and payload are sent as a unit.
//
// The read and write halves of the connection are serialized independently,
// so no interlocking is required. However each half may be accessed
// concurrently so the implementation of conn should protect against
// concurrent reads or concurrent writes.
func NewClient(conf *shmipc.SessionManagerConfig) (*Client, error) {
	smgr, err := shmipc.NewSessionManager(conf)
	if err != nil {
		return nil, fmt.Errorf("rpc: can't create client: %s", err)
	}

	return &Client{smgr: smgr}, nil
}

// Close calls the underlying codec's Close method. If the connection is already
// shutting down, ErrShutdown is returned.
func (client *Client) Close() error {
	client.mutex.Lock()
	if client.closing {
		client.mutex.Unlock()
		return ErrShutdown
	}
	client.closing = true
	client.mutex.Unlock()

	return client.smgr.Close()
}

// Call invokes the named function, waits for it to complete, and returns its error status.
func (client *Client) Call(serviceMethod string, args ShmWriter, reply ShmReader) error {
	// Register this call.
	client.mutex.Lock()
	if client.shutdown || client.closing {
		client.mutex.Unlock()
		return ErrShutdown
	}

	stream, err := client.smgr.GetStream()
	if err != nil {
		return fmt.Errorf("rpc: can't get stream: %s", err)

	}
	defer client.smgr.PutBack(stream)

	writer := stream.BufferWriter()
	data, err := writer.Reserve(2)
	if err != nil {
		return fmt.Errorf("rpc: can't write service method len: %s", err)
	}
	binary.BigEndian.PutUint16(data, uint16(len(serviceMethod)))

	if err := writer.WriteString(serviceMethod); err != nil {
		return fmt.Errorf("rpc: can't write service method: %s", err)
	}

	if err := args.WriteToShm(writer); err != nil {
		return fmt.Errorf("rpc: can't write to shared memory: %s", err)
	}

	if err := stream.Flush(false); err != nil {
		return fmt.Errorf("rpc: can't flush request to peer: %s", err)
	}

	reader := stream.BufferReader()
	data, err = reader.ReadBytes(4)
	if err != nil {
		return fmt.Errorf("rpc: can't read error message's len: %s", err)
	}
	errmsgLen := binary.BigEndian.Uint32(data)

	if errmsgLen > 0 {
		data, err = reader.ReadBytes(int(errmsgLen))
		if err != nil {
			return fmt.Errorf("rpc: can't read error message: %s", err)
		}

		return errors.New(string(data))
	}

	if err := reply.ReadFromShm(stream.BufferReader()); err != nil {
		return fmt.Errorf("rpc: can't read from shared memory: %s", err)
	}

	return nil
}
