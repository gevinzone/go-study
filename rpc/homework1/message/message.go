package message

import (
	"bytes"
	"encoding/binary"
)

const (
	splitter     = '\n'
	pairSplitter = '\r'
)

type Request struct {
	HeadLength uint32
	BodyLength uint32
	MessageId  uint32

	Version    byte
	Compressor byte
	Serializer byte

	ServiceName string
	MethodName  string

	Meta map[string]string

	Data []byte
}

func EncodeReq(req *Request) []byte {
	bs := make([]byte, req.HeadLength+req.BodyLength)
	cur := bs

	binary.BigEndian.PutUint32(cur[:4], req.HeadLength)
	binary.BigEndian.PutUint32(cur[4:8], req.BodyLength)
	binary.BigEndian.PutUint32(cur[8:12], req.MessageId)
	cur = cur[12:]

	cur[0], cur[1], cur[2] = req.Version, req.Compressor, req.Serializer
	cur = cur[3:]

	copy(cur, req.ServiceName)
	cur[len(req.ServiceName)] = splitter
	cur = cur[len(req.ServiceName)+1:]

	copy(cur, req.MethodName)
	cur[len(req.MethodName)] = splitter
	cur = cur[len(req.MethodName)+1:]

	for key, value := range req.Meta {
		copy(cur, key)
		cur[len(key)] = pairSplitter
		cur = cur[len(key)+1:]
		copy(cur, value)
		cur[len(value)] = splitter
		cur = cur[len(value)+1:]
	}
	copy(cur, req.Data)
	return bs
}

func DecodeReq(data []byte) *Request {
	req := &Request{}
	req.HeadLength = binary.BigEndian.Uint32(data[:4])
	req.BodyLength = binary.BigEndian.Uint32(data[4:8])
	req.MessageId = binary.BigEndian.Uint32(data[8:12])
	req.Version = data[12]
	req.Compressor = data[13]
	req.Serializer = data[14]

	head := data[15:req.HeadLength]
	index := bytes.IndexByte(head, splitter)
	req.ServiceName = string(head[:index])

	// 跳过splitter
	head = head[index+1:]
	index = bytes.IndexByte(head, splitter)
	req.MethodName = string(head[:index])

	head = head[index+1:]
	if len(head) > 0 {
		metaMap := make(map[string]string)
		index = bytes.IndexByte(head, splitter)
		for index != -1 {
			pair := head[:index]
			pairIndex := bytes.IndexByte(pair, pairSplitter)
			key := string(pair[:pairIndex])
			value := string(pair[pairIndex+1:])
			metaMap[key] = value

			head = head[index+1:]
			index = bytes.IndexByte(head, splitter)
		}
		req.Meta = metaMap
	}
	req.Data = data[req.HeadLength:]
	return req
}

func (r *Request) CalHeadLength() {
	res := 15
	res += len(r.ServiceName)
	res += 1
	res += len(r.MethodName)
	res += 1

	for key, value := range r.Meta {
		res = res + len(key) + 1 + len(value) + 1
	}

	r.HeadLength = uint32(res)
}

type Response struct {
	HeadLength uint32
	BodyLength uint32
	MessageId  uint32

	Version    byte
	Compressor byte
	Serializer byte

	Error []byte
	Data  []byte
}

func (r *Response) SetHeadLength() {
	r.HeadLength = uint32(15 + len(r.Error))
}

func EncodeResp(resp *Response) []byte {
	bs := make([]byte, resp.HeadLength+resp.BodyLength)
	cur := bs
	binary.BigEndian.PutUint32(cur[:4], resp.HeadLength)
	binary.BigEndian.PutUint32(cur[4:8], resp.BodyLength)
	binary.BigEndian.PutUint32(cur[8:12], resp.MessageId)
	cur = cur[12:]

	cur[0], cur[1], cur[2] = resp.Version, resp.Compressor, resp.Serializer
	cur = cur[3:]

	copy(cur, resp.Error)
	cur = cur[len(resp.Error):]
	copy(cur, resp.Data)
	return bs
}

func DecodeResp(bs []byte) *Response {
	resp := &Response{}
	resp.HeadLength = binary.BigEndian.Uint32(bs[:4])
	resp.BodyLength = binary.BigEndian.Uint32(bs[4:8])
	resp.MessageId = binary.BigEndian.Uint32(bs[8:12])
	resp.Version = bs[12]
	resp.Compressor = bs[13]
	resp.Serializer = bs[14]
	resp.Error = bs[15:resp.HeadLength]
	resp.Data = bs[resp.HeadLength:]
	return resp
}
