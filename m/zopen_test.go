package m

import (
	"bytes"
	"io"
	"testing"

	"gotest.tools/v3/assert"
)

// Test that ZReader works with an empty stream
func TestZReaderEmpty(t *testing.T) {
	bytesReader := bytes.NewReader([]byte{})

	zReader, err := ZReader(bytesReader)
	assert.NilError(t, err)

	all, err := io.ReadAll(zReader)
	assert.NilError(t, err)

	assert.Equal(t, 0, len(all))
}

// Test that ZReader works with a one-byte stream
func TestZReaderOneByte(t *testing.T) {
	bytesReader := bytes.NewReader([]byte{42})

	zReader, err := ZReader(bytesReader)
	assert.NilError(t, err)

	all, err := io.ReadAll(zReader)
	assert.NilError(t, err)

	assert.Equal(t, 1, len(all))
	assert.Equal(t, byte(42), all[0])
}
