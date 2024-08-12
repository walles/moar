//go:build !windows
// +build !windows

package twin

import (
	"io"
	"os"
	"os/signal"
	"runtime/debug"
	"sync/atomic"
	"syscall"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

type interruptableReaderImpl struct {
	base *os.File

	shutdownPipeReader *os.File
	shutdownPipeWriter *os.File

	interrupted atomic.Bool
}

func (r *interruptableReaderImpl) Read(p []byte) (n int, err error) {
	for {
		if r.interrupted.Load() {
			return 0, io.EOF
		}

		n, err = r.read(p)

		if err == syscall.EINTR {
			// Not really a problem, we can get this on window resizes for
			// example, just try again.
			continue
		}

		return
	}
}

func (r *interruptableReaderImpl) read(p []byte) (n int, err error) {
	// "This argument should be set to the highest-numbered file descriptor in
	// any of the three sets, plus 1. The indicated file descriptors in each set
	// are checked, up to this limit"
	//
	// Ref: https://man7.org/linux/man-pages/man2/select.2.html
	nfds := r.base.Fd()
	if r.shutdownPipeReader.Fd() > nfds {
		nfds = r.shutdownPipeReader.Fd()
	}

	readFds := unix.FdSet{}
	readFds.Set(int(r.shutdownPipeReader.Fd()))
	readFds.Set(int(r.base.Fd()))

	_, err = unix.Select(int(nfds)+1, &readFds, nil, nil, nil)
	if err != nil {
		// Select failed
		return
	}

	if readFds.IsSet(int(r.shutdownPipeReader.Fd())) {
		// Shutdown requested
		closeErr := r.shutdownPipeReader.Close()
		if closeErr != nil {
			// This should never happen, but if it does we should log it
			log.Debug("Failed to close shutdown pipe reader: ", closeErr)
		}

		err = io.EOF

		return
	}

	if readFds.IsSet(int(r.base.Fd())) {
		// Base has stuff
		return r.base.Read(p)
	}

	// Neither base nor shutdown pipe was ready, this should never happen
	return
}

func (r *interruptableReaderImpl) Interrupt() {
	r.interrupted.Store(true)

	err := r.shutdownPipeWriter.Close()
	if err != nil {
		// This should never happen, but if it does we should log it
		log.Warn("Failed to close shutdown pipe writer: ", err)
	}
}

func newInterruptableReader(base *os.File) (interruptableReader, error) {
	reader := interruptableReaderImpl{
		base: base,
	}
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	reader.shutdownPipeReader = pr
	reader.shutdownPipeWriter = pw

	return &reader, nil
}

func (screen *UnixScreen) setupSigwinchNotification() {
	screen.sigwinch = make(chan int, 1)
	screen.sigwinch <- 0 // Trigger initial screen size query

	sigwinch := make(chan os.Signal, 1)
	signal.Notify(sigwinch, syscall.SIGWINCH)
	go func() {
		defer func() {
			panicHandler("setupSigwinchNotification()/SIGWINCH", recover(), debug.Stack())
		}()

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
