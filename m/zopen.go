package m

import (
	"io"
	"os"

	"github.com/mholt/archiver/v4"
)

func ZOpen(filename string) (io.ReadCloser, error) {
	format, _, err := archiver.Identify(filename, nil)
	if err == archiver.ErrNoMatch {
		// Not compressed, just open the file
		return os.Open(filename)
	}

	if err != nil {
		return nil, err
	}

	if decompressor, ok := format.(archiver.Decompressor); ok {
		fileStream, err := os.Open(filename)
		if err != nil {
			return nil, err
		}

		return decompressor.OpenReader(fileStream)
	}

	// An archive of some sort, not much we can do with that, just hand the user
	// the binary gibberish.
	return os.Open(filename)
}
