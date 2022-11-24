package homework1

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
)

const lenBytes = 8

func ReadMsg(conn net.Conn) (bs []byte, err error) {
	msgLenBytes := make([]byte, lenBytes)
	length, err := conn.Read(msgLenBytes)
	defer func() {
		if msg := recover(); msg != nil {
			err = errors.New(fmt.Sprintf("%v", msg))
		}
	}()
	if err != nil {
		return nil, err
	}
	if length != lenBytes {
		return nil, errors.New("can not read data length")
	}
	headLength := binary.BigEndian.Uint32(msgLenBytes[:4])
	bodyLength := binary.BigEndian.Uint32(msgLenBytes[4:])

	bs = make([]byte, headLength+bodyLength)
	_, err = io.ReadFull(conn, bs[lenBytes:])
	copy(bs, msgLenBytes)
	return bs, err
}
