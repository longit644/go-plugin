package shared

import (
	"encoding/binary"

	"github.com/cloudwego/shmipc-go"
	"github.com/hashicorp/go-plugin/shmipc/rpc"
)

type PingRequest struct {
	Data []byte
}

var _ rpc.ShmReadWriter = (*PingRequest)(nil)

func (r *PingRequest) WriteToShm(writer shmipc.BufferWriter) error {
	data, err := writer.Reserve(8)
	if err != nil {
		return err
	}
	binary.BigEndian.PutUint64(data, uint64(len(r.Data)))

	_, err = writer.WriteBytes(r.Data)
	return err
}

func (r *PingRequest) ReadFromShm(reader shmipc.BufferReader) error {
	data, err := reader.ReadBytes(8)
	if err != nil {
		return err
	}
	r.Data, err = reader.ReadBytes(int(binary.BigEndian.Uint64(data)))
	return err
}

type PingResponse struct {
	Data []byte
}

func (r *PingResponse) WriteToShm(writer shmipc.BufferWriter) error {
	data, err := writer.Reserve(8)
	if err != nil {
		return err
	}
	binary.BigEndian.PutUint64(data, uint64(len(r.Data)))

	_, err = writer.WriteBytes(r.Data)
	return err
}

func (r *PingResponse) ReadFromShm(reader shmipc.BufferReader) error {
	data, err := reader.ReadBytes(8)
	if err != nil {
		return err
	}
	r.Data, err = reader.ReadBytes(int(binary.BigEndian.Uint64(data)))
	return err
}
