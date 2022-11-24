package proto

import (
	"errors"
	"github.com/golang/protobuf/proto"
)

var errWrongProtoStruct = errors.New("micro: 使用 proto 序列化协议必须使用 protoc 编译的类型")

type Serializer struct {
}

func (s Serializer) Code() byte {
	return 1
}

func (s Serializer) Encode(val any) ([]byte, error) {
	msg, ok := val.(proto.Message)
	if !ok {
		return nil, errWrongProtoStruct
	}
	return proto.Marshal(msg)
}

func (s Serializer) Decode(data []byte, val any) error {
	msg, ok := val.(proto.Message)
	if !ok {
		return errWrongProtoStruct
	}
	return proto.Unmarshal(data, msg)
}
