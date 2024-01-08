package m

import (
	"compress/bzip2"
	"compress/gzip"
	"io"
	"os"
	"strings"

	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
)

func ZOpen(filename string) (io.ReadCloser, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	switch {
	case strings.HasSuffix(filename, ".gz"):
		return gzip.NewReader(file)

	case strings.HasSuffix(filename, ".bz2"):
		return struct {
			io.Reader
			io.Closer
		}{bzip2.NewReader(file), file}, nil

	case strings.HasSuffix(filename, ".zst") || strings.HasSuffix(filename, ".zstd"):
		decoder, err := zstd.NewReader(file)
		if err != nil {
			return nil, err
		}
		return decoder.IOReadCloser(), nil

	case strings.HasSuffix(filename, ".xz"):
		xzReader, err := xz.NewReader(file)
		if err != nil {
			return nil, err
		}

		return struct {
			io.Reader
			io.Closer
		}{xzReader, file}, nil
	}

	return file, nil
}
