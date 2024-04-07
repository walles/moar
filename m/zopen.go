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

// The second return value is the file name with any compression extension removed.
func ZOpen(filename string) (io.ReadCloser, string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, "", err
	}

	switch {
	case strings.HasSuffix(filename, ".gz"):
		reader, err := gzip.NewReader(file)
		return reader, strings.TrimSuffix(filename, ".gz"), err

	// Ref: https://github.com/walles/moar/issues/194
	case strings.HasSuffix(filename, ".tgz"):
		reader, err := gzip.NewReader(file)
		return reader, strings.TrimSuffix(filename, ".tgz"), err

	case strings.HasSuffix(filename, ".bz2"):
		return struct {
			io.Reader
			io.Closer
		}{bzip2.NewReader(file), file}, strings.TrimSuffix(filename, ".bz2"), nil

	case strings.HasSuffix(filename, ".zst") || strings.HasSuffix(filename, ".zstd"):
		decoder, err := zstd.NewReader(file)
		if err != nil {
			return nil, "", err
		}
		return decoder.IOReadCloser(), strings.TrimSuffix(filename, ".zst"), nil

	case strings.HasSuffix(filename, ".xz"):
		xzReader, err := xz.NewReader(file)
		if err != nil {
			return nil, "", err
		}

		return struct {
			io.Reader
			io.Closer
		}{xzReader, file}, strings.TrimSuffix(filename, ".xz"), nil
	}

	return file, filename, nil
}
