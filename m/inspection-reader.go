package m

import "io"

// Pass-through reader that counts the number of bytes read.
type inspectionReader struct {
	base       io.Reader
	bytesCount int64
}

func (r *inspectionReader) Read(p []byte) (n int, err error) {
	n, err = r.base.Read(p)
	r.bytesCount += int64(n)
	return
}
