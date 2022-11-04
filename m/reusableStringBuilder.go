package m

import "unicode/utf8"

type reusableStringBuilder struct {
	buf    []byte
	length int // Number of used bytes in buf
}

func (rsb *reusableStringBuilder) Reset() {
	rsb.length = 0
}

// Inspired by the source code of strings.Builder
func (rsb *reusableStringBuilder) WriteRune(r rune) {
	availableBytes := cap(rsb.buf) - rsb.length
	if availableBytes < utf8.UTFMax {
		newLength := 2*cap(rsb.buf) + utf8.UTFMax
		newBuf := make([]byte, newLength)
		for i := 0; i < rsb.length; i++ {
			newBuf[i] = rsb.buf[i]
		}
		rsb.buf = newBuf
	}

	encodedBytesCount := utf8.EncodeRune(rsb.buf[rsb.length:rsb.length+utf8.UTFMax], r)
	rsb.length += encodedBytesCount
}

func (rsb *reusableStringBuilder) String() string {
	return string(rsb.buf[0:rsb.length])
}
