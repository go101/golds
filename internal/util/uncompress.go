package util

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

// This function is very simple. It is intended to get a single file.
func UncompressTarGzipData(data []byte) ([]byte, error) {
	gzipReader, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("gzip.NewReader error: %s", err)
	}

	tarReader := tar.NewReader(gzipReader)
	header, err := tarReader.Next()
	if err != nil {
		return nil, fmt.Errorf("tarReader.Next error: %s", err)
	}
	if header.Typeflag != tar.TypeReg {
		return nil, fmt.Errorf("not a file")
	}

	buf := bytes.NewBuffer(make([]byte, 0, 128*1024))
	if _, err := io.Copy(buf, tarReader); err != nil {
		return nil, fmt.Errorf("io.Copy error: %s", err)
	}

	return buf.Bytes(), err
}
