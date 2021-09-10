//go:build windows
// +build windows

package twin

import (
	"os"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/term"
)

func (screen *UnixScreen) setupSigwinchNotification() {
	screen.sigwinch = make(chan int, 1)
	screen.sigwinch <- 0 // Trigger initial screen size query

	// No SIGWINCH handling on Windows for now, contributions welcome, see
	// sigwinch.go for inspiration.
}

func (screen *UnixScreen) setupTtyInTtyOut() error {
	in, err := syscall.Open("CONIN$", syscall.O_RDWR, 0)
	if err != nil {
		return err
	}

	screen.ttyIn = os.NewFile(uintptr(in), "/dev/tty")

	// Set input stream to raw mode
	stdin := windows.Handle(screen.ttyIn.Fd())
	err = windows.GetConsoleMode(stdin, &screen.oldTtyInMode)
	if err != nil {
		return err
	}
	err = windows.SetConsoleMode(stdin, screen.oldTtyInMode|windows.ENABLE_VIRTUAL_TERMINAL_INPUT)
	if err != nil {
		return err
	}

	screen.oldTerminalState, err = term.MakeRaw(int(screen.ttyIn.Fd()))
	if err != nil {
		screen.restoreTtyInTtyOut() // Error intentionally ignored, report the first one only
		return err
	}

	screen.ttyOut = os.Stdout

	// Enable console colors, from: https://stackoverflow.com/a/52579002
	stdout := windows.Handle(screen.ttyOut.Fd())
	err = windows.GetConsoleMode(stdout, &screen.oldTtyOutMode)
	if err != nil {
		screen.restoreTtyInTtyOut() // Error intentionally ignored, report the first one only
		return err
	}
	err = windows.SetConsoleMode(stdout, screen.oldTtyOutMode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
	if err != nil {
		screen.restoreTtyInTtyOut() // Error intentionally ignored, report the first one only
		return err
	}

	return nil
}

func (screen *UnixScreen) restoreTtyInTtyOut() error {
	stdin := windows.Handle(screen.ttyIn.Fd())
	err := windows.SetConsoleMode(stdin, screen.oldTtyInMode)
	if err != nil {
		return err
	}

	stdout := windows.Handle(screen.ttyOut.Fd())
	err = windows.SetConsoleMode(stdout, screen.oldTtyOutMode)
	if err != nil {
		return err
	}

	return nil
}
