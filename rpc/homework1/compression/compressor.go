package compression

type Compressor interface {
	Code() byte
	Compress(uncompressed []byte) (compressed []byte, err error)
	Decompress(compressed []byte) (uncompressed []byte, err error)
}
