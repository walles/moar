package internal

import (
	"strings"
	"sync"
)

type LogWriter struct {
	lock   sync.Mutex
	buffer strings.Builder
}

func (lw *LogWriter) Write(p []byte) (n int, err error) {
	lw.lock.Lock()
	defer lw.lock.Unlock()

	return lw.buffer.Write(p)
}

func (lw *LogWriter) String() string {
	lw.lock.Lock()
	defer lw.lock.Unlock()

	return lw.buffer.String()
}
