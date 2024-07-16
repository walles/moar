//go:build !windows
// +build !windows

package twin

import (
	"io"
	"os"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

// This test should replace TestInterruptableReader_blockedOnRead if or when
// Windows catches up with the shutdown implementation.
func TestInterruptableReader_blockedOnReadImmediate(t *testing.T) {
	// Make a pipe to read from and write to
	pipeReader, pipeWriter, err := os.Pipe()
	assert.NilError(t, err)

	// Make an interruptable reader
	testMe, err := newInterruptableReader(pipeReader)
	assert.NilError(t, err)
	assert.Assert(t, testMe != nil)

	// Start a thread that reads from the pipe
	type readResult struct {
		n   int
		err error
	}
	readResultChan := make(chan readResult)
	go func() {
		buffer := make([]byte, 1)
		n, err := testMe.Read(buffer)
		readResultChan <- readResult{n, err}
	}()

	// Give the reader thread some time to start waiting
	time.Sleep(100 * time.Millisecond)

	// Interrupt the reader
	testMe.Interrupt()

	// Wait for the reader thread to finish
	result := <-readResultChan

	// Check the result
	assert.Equal(t, result.n, 0)
	assert.Equal(t, result.err, io.EOF)

	// Another read should return EOF immediately
	buffer := make([]byte, 1)
	n, err := testMe.Read(buffer)
	assert.Equal(t, err, io.EOF)
	assert.Equal(t, n, 0)

	// Even if there are bytes, the interrupted reader should still return EOF
	n, err = pipeWriter.Write([]byte{42})
	assert.NilError(t, err)
	assert.Equal(t, n, 1)

	n, err = testMe.Read(buffer)
	assert.Equal(t, err, io.EOF)
	assert.Equal(t, n, 0)
}
