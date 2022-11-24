package serialization

type Serializer interface {
	Code() byte
	Encode(val any) ([]byte, error)
	Decode(data []byte, val any) error
}
