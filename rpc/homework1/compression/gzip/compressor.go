package gzip

import (
	"bytes"
	"compress/gzip"
	"io"
)

type Compressor struct {
}

func (c Compressor) Code() byte {
	return 1
}

func (c Compressor) Compress(uncompressed []byte) (compressed []byte, err error) {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)

	_, err = gz.Write(uncompressed)
	if err != nil {
		return
	}

	if err = gz.Flush(); err != nil {
		return
	}

	if err = gz.Close(); err != nil {
		return
	}

	compressed = b.Bytes()
	return
}

func (c Compressor) Decompress(compressed []byte) (uncompressed []byte, err error) {
	b := bytes.NewBuffer(compressed)

	var r io.Reader
	r, err = gzip.NewReader(b)
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
