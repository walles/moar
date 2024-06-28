//go:build !windows
// +build !windows

package twin

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
	"golang.org/x/term"
)

func (screen *UnixScreen) setupSigwinchNotification() {
	screen.sigwinch = make(chan int, 1)
	screen.sigwinch <- 0 // Trigger initial screen size query

	sigwinch := make(chan os.Signal, 1)
	signal.Notify(sigwinch, syscall.SIGWINCH)
	go func() {
		for {
			// Await window resize signal
			<-sigwinch

			select {
			case screen.sigwinch <- 0:
				// Screen.Size() method notified about resize
			default:
				// Notification already pending, never mind
			}

			// Notify client app.
			select {
			case screen.events <- EventResize{}:
				// Event delivered
			default:
				// This likely means that the user isn't processing events
				// quickly enough. Maybe the user's queue will get flooded if
				// the window is resized too quickly?
				log.Warn("Unable to deliver EventResize, event queue full")
			}
		}
	}()
}

func (screen *UnixScreen) setupTtyInTtyOut() error {
	// Dup stdout so we can close stdin in Close() without closing stdout.
	// Before this dupping, we crashed on using --quit-if-one-screen.
	//
	// Ref:https://github.com/walles/moar/issues/214
	stdoutDupFd, err := syscall.Dup(int(os.Stdout.Fd()))
	if err != nil {
		return err
	}
	stdoutDup := os.NewFile(uintptr(stdoutDupFd), "moar-stdout-dup")

	// os.Stdout is a stream that goes to our terminal window.
	//
	// So if we read from there, we'll get input from the terminal window.
	//
	// If we just read from os.Stdin that would fail when getting data piped
	// into ourselves from some other command.
	//
	// Tested on macOS and Linux, works like a charm!
	screen.ttyIn = stdoutDup // <- YES, WE SHOULD ASSIGN STDOUT TO TTYIN

	// Set input stream to raw mode
	screen.oldTerminalState, err = term.MakeRaw(int(screen.ttyIn.Fd()))
	if err != nil {
		return err
	}

	screen.ttyOut = os.Stdout
	return nil
}

func (screen *UnixScreen) restoreTtyInTtyOut() error {
	return term.Restore(int(screen.ttyIn.Fd()), screen.oldTerminalState)
}
