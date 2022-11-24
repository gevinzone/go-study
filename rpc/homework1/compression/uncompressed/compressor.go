package uncompressed

type Compressor struct {
}

func (c Compressor) Code() byte {
	return 0
}

func (c Compressor) Compress(uncompressed []byte) (compressed []byte, err error) {
	return uncompressed, nil
}

func (c Compressor) Decompress(compressed []byte) (uncompressed []byte, err error) {
	return compressed, nil
}
