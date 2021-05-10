package m

import (
	"bufio"
	"bytes"
	"io"
)

type LineReader struct {
	reader         io.Reader
	completeLine   []byte
	buffer         []byte
	bufferSize     int
	bufferPosition int
	err            error
	done           bool
	eof            bool
}

func NewLineReader(reader io.Reader) LineReader {
	return LineReader{
		reader: reader,
		buffer: make([]byte, bufio.MaxScanTokenSize),
	}
}

// Reads one line from the stream
//
// On EOF, *string will be nil.
func (lineReader *LineReader) GetLine() (*string, error) {
	if lineReader.done {
		return nil, lineReader.err
	}

	for {
		lineReader.refillBuffer()
		if lineReader.done {
			return nil, lineReader.err
		}

		// INVARIANT: At this point, we know the line reader has data to go through

		// Find the next linefeed, starting at lineReader.bufferPosition
		activeBuffer := lineReader.buffer[lineReader.bufferPosition:lineReader.bufferSize]
		nextLinefeedIndex := bytes.IndexByte(activeBuffer, '\n')

		if nextLinefeedIndex >= 0 {
			// Next line ends here
			nextLineBytesNoNewline := activeBuffer[0:nextLinefeedIndex]

			// FIXME: Handle newline being preceeded by '\r'

			lineReader.completeLine = append(lineReader.completeLine, nextLineBytesNoNewline...)

			lineReader.bufferPosition += nextLinefeedIndex + 1
			completeLineString := string(lineReader.completeLine)
			lineReader.completeLine = lineReader.completeLine[:0]
			return &completeLineString, lineReader.err
		}

		// INVARIANT: No more newlines found

		lineReader.completeLine = append(lineReader.completeLine, activeBuffer...)
		lineReader.bufferPosition = lineReader.bufferSize
		if lineReader.eof {
			lineReader.done = true

			completeLineString := string(lineReader.completeLine)
			lineReader.completeLine = lineReader.completeLine[:0]
			if len(completeLineString) > 0 {
				return &completeLineString, lineReader.err
			} else {
				return nil, lineReader.err
			}
		}
	}
}

func (lineReader *LineReader) refillBuffer() {
	if lineReader.bufferPosition < lineReader.bufferSize {
		// We have more data to go through before asking for more
		return
	}

	lineReader.bufferSize, lineReader.err =
		lineReader.reader.Read(lineReader.buffer)
	lineReader.bufferPosition = 0
	if lineReader.err == io.EOF {
		// This is not an error
		lineReader.err = nil
		lineReader.eof = true
	}

	if lineReader.err != nil {
		// Can't go on after error
		lineReader.done = true
	}
}
