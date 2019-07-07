package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/gdamore/tcell"
	"github.com/walles/moar/m"

	"golang.org/x/crypto/ssh/terminal"
)

func main() {
	// FIXME: On any panic or warnings, also print system info and how to report bugs

	version := flag.Bool("version", false, "Prints the moar version number")

	// FIXME: Support --help

	// FIXME: Support --no-highlight

	flag.Parse()
	if *version {
		fmt.Println("FIXME: Imagine a version string here")
		os.Exit(0)
	}

	stdinIsRedirected := !terminal.IsTerminal(int(os.Stdin.Fd()))
	stdoutIsRedirected := !terminal.IsTerminal(int(os.Stdout.Fd()))
	if stdinIsRedirected && stdoutIsRedirected {
		io.Copy(os.Stdout, os.Stdin)
		os.Exit(0)
	}

	if stdinIsRedirected && !stdoutIsRedirected {
		// Display input pipe contents
		reader, err := m.NewReaderFromStream(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		_StartPaging(reader)
		return
	}

	if len(flag.Args()) != 1 {
		// FIXME: Improve this message
		fmt.Fprintln(os.Stderr, "ERROR: Expected exactly one filename, got: ", flag.Args())

		// FIXME: Print full usage here

		os.Exit(1)
	}

	if stdoutIsRedirected {
		// Pump from file by given name onto stdout which is redirected
		input, err := os.Open(flag.Arg(0))
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
			os.Exit(1)
		}
		defer input.Close()

		// Copy input file to redirected stdout
		io.Copy(os.Stdout, input)
		os.Exit(0)
	}

	// Display the input file contents
	reader, err := m.NewReaderFromFilename(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	_StartPaging(reader)
}

func _StartPaging(reader *m.Reader) {
	screen, e := tcell.NewScreen()
	if e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}

	var loglines strings.Builder
	defer func() {
		// Restore screen...
		screen.Fini()

		// ... before printing panic() output, otherwise the output will have
		// broken linefeeds and be hard to follow.
		if err := recover(); err != nil {
			panic(err)
		}

		if len(loglines.String()) > 0 {
			fmt.Fprintf(os.Stderr, "%s", loglines.String())
			os.Exit(1)
		}
	}()

	logger := log.New(&loglines, "", 0)
	m.NewPager(*reader).StartPaging(logger, screen)
}
