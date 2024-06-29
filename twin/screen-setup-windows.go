//go:build windows
// +build windows

package twin

import (
	"fmt"
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
		return fmt.Errorf("failed to open CONIN$: %w", err)
	}

	screen.ttyIn = os.NewFile(uintptr(in), "/dev/tty")

	// Set input stream to raw mode
	stdin := windows.Handle(screen.ttyIn.Fd())
	err = windows.GetConsoleMode(stdin, &screen.oldTtyInMode)
	if err != nil {
		return fmt.Errorf("failed to get stdin console mode: %w", err)
	}
	err = windows.SetConsoleMode(stdin, screen.oldTtyInMode|windows.ENABLE_VIRTUAL_TERMINAL_INPUT)
	if err != nil {
		return fmt.Errorf("failed to set stdin console mode: %w", err)
	}

	screen.oldTerminalState, err = term.MakeRaw(int(screen.ttyIn.Fd()))
	if err != nil {
		screen.restoreTtyInTtyOut() // Error intentionally ignored, report the first one only
		return fmt.Errorf("failed to set raw mode: %w", err)
	}

	screen.ttyOut = os.Stdout

	// Enable console colors, from: https://stackoverflow.com/a/52579002
	stdout := windows.Handle(screen.ttyOut.Fd())
	err = windows.GetConsoleMode(stdout, &screen.oldTtyOutMode)
	if err != nil {
		screen.restoreTtyInTtyOut() // Error intentionally ignored, report the first one only
		return fmt.Errorf("failed to get stdout console mode: %w", err)
	}
	err = windows.SetConsoleMode(stdout, screen.oldTtyOutMode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
	if err != nil {
		screen.restoreTtyInTtyOut() // Error intentionally ignored, report the first one only
		return fmt.Errorf("failed to set stdout console mode: %w", err)
	}

	return nil
}

func (screen *UnixScreen) restoreTtyInTtyOut() error {
	errors := []error{}

	stdin := windows.Handle(screen.ttyIn.Fd())
	err := windows.SetConsoleMode(stdin, screen.oldTtyInMode)
	if err != nil {
		errors = append(errors, fmt.Errorf("failed to restore stdin console mode: %w", err))
	}

	stdout := windows.Handle(screen.ttyOut.Fd())
	err = windows.SetConsoleMode(stdout, screen.oldTtyOutMode)
	if err != nil {
		errors = append(errors, fmt.Errorf("failed to restore stdout console mode: %w", err))
	}

	if len(errors) == 0 {
		return nil
	}

	return fmt.Errorf("failed to restore terminal state: %v", errors)
}
