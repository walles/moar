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
	// This won't work if we're getting data piped to us, contributions welcome.
	screen.ttyIn = os.Stdin

	// Set input stream to raw mode
	var err error
	stdin := windows.Handle(screen.ttyIn.Fd())
	var originalMode uint32
	err = windows.GetConsoleMode(stdin, &originalMode)
	if err != nil {
		panic(err)
	}
	err = windows.SetConsoleMode(stdin, originalMode|windows.ENABLE_VIRTUAL_TERMINAL_INPUT)
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
	err = windows.GetConsoleMode(stdout, &originalMode)
	if err != nil {
		panic(err)
	}
	err = windows.SetConsoleMode(stdout, originalMode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
	if err != nil {
		panic(err)
	}
}
