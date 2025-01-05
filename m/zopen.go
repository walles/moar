package m

import (
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/klauspost/compress/zstd"
	log "github.com/sirupsen/logrus"
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

// ZReader returns a reader that decompresses the input stream. Any input stream
// compression will be automatically detected. Uncompressed streams will be
// returned as-is.
//
// Ref: https://github.com/walles/moar/issues/261
func ZReader(input io.Reader) (io.Reader, error) {
	// Read the first 6 bytes to determine the compression type
	buffer := make([]byte, 6)
	_, err := input.Read(buffer)
	if err != nil {
		if err == io.EOF {
			// Return a reader for the short input
			return bytes.NewReader(buffer), nil
		}
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	// Reset input reader to start of stream
	input = io.MultiReader(bytes.NewReader(buffer), input)

	switch {
	case bytes.HasPrefix(buffer, []byte{0x1f, 0x8b}): // Gzip magic numbers
		log.Info("Input stream is gzip compressed")
		return gzip.NewReader(input)
	case bytes.HasPrefix(buffer, []byte{0x28, 0xb5, 0x2f, 0xfd}): // Zstd magic numbers
		log.Info("Input stream is zstd compressed")
		return zstd.NewReader(input)
	case bytes.HasPrefix(buffer, []byte{0x42, 0x5a, 0x68}): // Bzip2 magic numbers
		log.Info("Input stream is bzip2 compressed")
		return bzip2.NewReader(input), nil
	case bytes.HasPrefix(buffer, []byte{0xfd, 0x37, 0x7a, 0x58, 0x5a, 0x00}): // XZ magic numbers
		log.Info("Input stream is xz compressed")
		return xz.NewReader(input)
	default:
		// No magic numbers matched
		log.Info("Input stream is assumed to be uncompressed")
		return input, nil
	}
}
