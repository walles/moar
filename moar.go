package main

import (
	"fmt"
	"io"
	"os"

	"github.com/gdamore/tcell"
	"github.com/walles/moar/m"

	"golang.org/x/crypto/ssh/terminal"
)

func main() {
	stdinIsRedirected := !terminal.IsTerminal(int(os.Stdin.Fd()))
	stdoutIsRedirected := !terminal.IsTerminal(int(os.Stdout.Fd()))
	if stdinIsRedirected && stdoutIsRedirected {
		io.Copy(os.Stdout, os.Stdin)
		os.Exit(0)
	}

	if stdinIsRedirected && !stdoutIsRedirected {
		// Display input pipe contents
		reader, _ := m.NewReaderFromStream(os.Stdin) // FIXME: Error handling

		screen, e := tcell.NewScreen()
		if e != nil {
			fmt.Fprintf(os.Stderr, "%v\n", e)
			os.Exit(1)
		}
		defer screen.Fini()

		m.NewPager(*reader).StartPaging(screen)
		return
	}

	// FIXME: Support --help

	// FIXME: Support --version

	if len(os.Args) != 2 {
		// FIXME: Improve this message
		fmt.Fprintf(os.Stderr, "ERROR: Expected exactly one parameter, got: %v\n", os.Args[1:])
		os.Exit(1)
	}

	if stdoutIsRedirected {
		// Pump from file by given name onto stdout which is redirected
		input, err := os.Open(os.Args[1])
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
	reader, _ := m.NewReaderFromFilename(os.Args[1]) // FIXME: Error handling

	screen, e := tcell.NewScreen()
	if e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}
	defer screen.Fini()

	m.NewPager(*reader).StartPaging(screen)
}
