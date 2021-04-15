// +build windows

package twin

import "os"

func (screen *UnixScreen) setupSigwinchNotification() {
	screen.sigwinch = make(chan int, 1)
	screen.sigwinch <- 0 // Trigger initial screen size query

	// No SIGWINCH handling on Windows for now, contributions welcome, see
	// sigwinch.go for inspiration.
}

func (screen *UnixScreen) setupTtyIn() {
	var err error
	screen.ttyIn, err := os.Open("CONIN$")
	if err != nil {
		panic(err)
	}
}
