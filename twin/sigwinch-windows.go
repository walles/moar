// +build windows

package twin

func (screen *UnixScreen) setupSigwinchNotification() {
	screen.sigwinch = make(chan int, 1)
	screen.sigwinch <- 0 // Trigger initial screen size query

	// No SIGWINCH handling on Windows for now, contributions welcome, see
	// sigwinch.go for inspiration.
}
