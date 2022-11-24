package gzip

import (
	"bytes"
	"compress/zlib"
	"io"
)

type Compressor struct {
}

func (c Compressor) Code() byte {
	return 1
}

func (c Compressor) Compress(uncompressed []byte) (compressed []byte, err error) {
	var b bytes.Buffer
	z := zlib.NewWriter(&b)

	_, err = z.Write(uncompressed)
	if err != nil {
		return
	}

	if err = z.Flush(); err != nil {
		return
	}

	if err = z.Close(); err != nil {
		return
	}

	compressed = b.Bytes()
	return
}

func (c Compressor) Decompress(compressed []byte) (uncompressed []byte, err error) {
	b := bytes.NewBuffer(compressed)

	var r io.Reader
	r, err = zlib.NewReader(b)
	if err != nil {
		return
	}

	var resB bytes.Buffer
	_, err = resB.ReadFrom(r)
	if err != nil {
		return
	}

	uncompressed = resB.Bytes()

	return
}
