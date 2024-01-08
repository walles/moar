package m

import (
	"compress/bzip2"
	"compress/gzip"
	"io"
	"os"
	"strings"
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
	}

	return file, nil
}
