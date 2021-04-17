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

func (screen *UnixScreen) setupTtyInTtyOut() {
	// os.Stdout is a stream that goes to our terminal window.
	//
	// So if we read from there, we'll get input from the terminal window.
	//
	// Reading from os.Stdin will fail if we're getting data piped into
	// ourselves from some other command.
	//
	// Tested on macOS and Linux, works like a charm!
	screen.ttyIn = os.Stdout // <- YES, WE SHOULD ASSIGN STDOUT TO TTYIN

	// Set input stream to raw mode
	var err error
	screen.oldTerminalState, err = term.MakeRaw(int(screen.ttyIn.Fd()))
	if err != nil {
		panic(err)
	}

	screen.ttyOut = os.Stdout
}
