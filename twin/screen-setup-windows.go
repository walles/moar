// +build windows

package twin

import (
	"os"

	"golang.org/x/sys/windows"
	"golang.org/x/term"
)

func (screen *UnixScreen) setupSigwinchNotification() {
	screen.sigwinch = make(chan int, 1)
	screen.sigwinch <- 0 // Trigger initial screen size query

	// No SIGWINCH handling on Windows for now, contributions welcome, see
	// sigwinch.go for inspiration.
}

func (screen *UnixScreen) setupTtyInTtyOut() {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		// See the counterpart method in screen-setup.go for inspiration.
		//
		// A fix might be centered about opening "CONIN$" as screen.ttyIn rather
		// than using os.Stdin, but I never got that working fully.
		// Contributions welcome.
		panic("Getting piped data on stdin is not supported on Windows, fixes needed in here: https://github.com/walles/moar/blob/walles/fix-windows/twin/screen-setup.go")
	}

	// This won't work if we're getting data piped to us, contributions welcome.
	screen.ttyIn = os.Stdin

	// Set input stream to raw mode
	var err error
	stdin := windows.Handle(screen.ttyIn.Fd())
	err = windows.GetConsoleMode(stdin, &screen.oldTtyInMode)
	if err != nil {
		panic(err)
	}
	err = windows.SetConsoleMode(stdin, screen.oldTtyInMode|windows.ENABLE_VIRTUAL_TERMINAL_INPUT)
	if err != nil {
		panic(err)
	}

	screen.oldTerminalState, err = term.MakeRaw(int(screen.ttyIn.Fd()))
	if err != nil {
		panic(err)
	}

	screen.ttyOut = os.Stdout

	// Enable console colors, from: https://stackoverflow.com/a/52579002
	stdout := windows.Handle(screen.ttyOut.Fd())
	err = windows.GetConsoleMode(stdout, &screen.oldTtyOutMode)
	if err != nil {
		panic(err)
	}
	err = windows.SetConsoleMode(stdout, screen.oldTtyOutMode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
	if err != nil {
		panic(err)
	}
}

func (screen *UnixScreen) restoreTtyInTtyOut() {
	stdin := windows.Handle(screen.ttyIn.Fd())
	err := windows.SetConsoleMode(stdin, screen.oldTtyInMode)
	if err != nil {
		panic(err)
	}

	stdout := windows.Handle(screen.ttyOut.Fd())
	err = windows.SetConsoleMode(stdout, screen.oldTtyOutMode)
	if err != nil {
		panic(err)
	}
}
