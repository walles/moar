// +build windows

package twin

import "syscall"

func (screen *UnixScreen) setupSigwinchNotification() {
	screen.sigwinch = make(chan int, 1)
	screen.sigwinch <- 0 // Trigger initial screen size query

	// No SIGWINCH handling on Windows for now, contributions welcome, see
	// sigwinch.go for inspiration.
}

func (screen *UnixScreen) setupTtyIn() {
	screen.ttyIn = syscall.Open("CONIN$", syscall.O_RDONLY, 0)
}
