// +build !windows

package twin

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
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
