package m

import "io"

// Pass-through reader that counts the number of bytes read.
type inspectionReader struct {
	base       io.Reader
	bytesCount int64

	endedWithNewline bool
}

func (r *inspectionReader) Read(p []byte) (n int, err error) {
	n, err = r.base.Read(p)
	r.bytesCount += int64(n)

	if err != nil {
		return
	}

	if n > 0 {
		r.endedWithNewline = p[n-1] == '\n'
	} else {
		r.endedWithNewline = false
	}

	return
}
