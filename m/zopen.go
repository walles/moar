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

var gzipMagic = []byte{0x1f, 0x8b}
var bzip2Magic = []byte{0x42, 0x5a, 0x68}
var zstdMagic = []byte{0x28, 0xb5, 0x2f, 0xfd}
var xzMagic = []byte{0xfd, 0x37, 0x7a, 0x58, 0x5a, 0x00}

// The second return value is the file name with any compression extension removed.
func ZOpen(filename string) (io.ReadCloser, string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, "", err
	}

	// Read the first 6 bytes to determine the compression type
	firstBytes := make([]byte, 6)
	_, err = file.Read(firstBytes)
	if err != nil {
		if err == io.EOF {
			// File was empty
			return file, filename, nil
		}
		return nil, "", fmt.Errorf("failed to read file: %w", err)
	}

	// Reset file reader to start of file
	_, err = file.Seek(0, 0)
	if err != nil {
		return nil, "", fmt.Errorf("failed to seek to start of file: %w", err)
	}

	switch {
	case bytes.HasPrefix(firstBytes, gzipMagic):
		reader, err := gzip.NewReader(file)
		if err != nil {
			return nil, "", err
		}

		newName := strings.TrimSuffix(filename, ".gz")

		// Ref: https://github.com/walles/moar/issues/194
		if strings.HasSuffix(newName, ".tgz") {
			newName = strings.TrimSuffix(newName, ".tgz") + ".tar"
		}

		return reader, newName, err

	case bytes.HasPrefix(firstBytes, bzip2Magic):
		return struct {
			io.Reader
			io.Closer
		}{bzip2.NewReader(file), file}, strings.TrimSuffix(filename, ".bz2"), nil

	case bytes.HasPrefix(firstBytes, zstdMagic):
		decoder, err := zstd.NewReader(file)
		if err != nil {
			return nil, "", err
		}

		newName := strings.TrimSuffix(filename, ".zst")
		newName = strings.TrimSuffix(newName, ".zstd")
		return decoder.IOReadCloser(), newName, nil

	case bytes.HasPrefix(firstBytes, xzMagic):
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
	firstBytes := make([]byte, 6)
	_, err := input.Read(firstBytes)
	if err != nil {
		if err == io.EOF {
			// Stream was empty
			return input, nil
		}
		return nil, fmt.Errorf("failed to read stream: %w", err)
	}

	// Reset input reader to start of stream
	input = io.MultiReader(bytes.NewReader(firstBytes), input)

	switch {
	case bytes.HasPrefix(firstBytes, gzipMagic):
		log.Info("Input stream is gzip compressed")
		return gzip.NewReader(input)
	case bytes.HasPrefix(firstBytes, zstdMagic):
		log.Info("Input stream is zstd compressed")
		return zstd.NewReader(input)
	case bytes.HasPrefix(firstBytes, bzip2Magic):
		log.Info("Input stream is bzip2 compressed")
		return bzip2.NewReader(input), nil
	case bytes.HasPrefix(firstBytes, xzMagic):
		log.Info("Input stream is xz compressed")
		return xz.NewReader(input)
	default:
		// No magic numbers matched
		log.Info("Input stream is assumed to be uncompressed")
		return input, nil
	}
}
