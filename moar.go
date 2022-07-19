package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/styles"
	log "github.com/sirupsen/logrus"
	"golang.org/x/term"

	"github.com/walles/moar/m"
	"github.com/walles/moar/twin"
)

var versionString = "Should be set when building, please use build.sh to build"

func printUsage(output io.Writer, flagSet *flag.FlagSet, printCommandline bool) {
	// This controls where PrintDefaults() prints, see below
	flagSet.SetOutput(output)

	// FIXME: Log if any printouts fail?
	moarEnv := os.Getenv("MOAR")
	if printCommandline {
		_, _ = fmt.Fprintln(output, "Commandline: moar", strings.Join(os.Args[1:], " "))
		_, _ = fmt.Fprintf(output, "Environment: MOAR=\"%v\"\n", moarEnv)
		_, _ = fmt.Fprintln(output)
	}

	_, _ = fmt.Fprintln(output, "Usage:")
	_, _ = fmt.Fprintln(output, "  moar [options] <file>")
	_, _ = fmt.Fprintln(output, "  ... | moar")
	_, _ = fmt.Fprintln(output, "  moar < file")
	_, _ = fmt.Fprintln(output)
	_, _ = fmt.Fprintln(output, "Shows file contents. Compressed files will be transparently decompressed.")
	_, _ = fmt.Fprintln(output)
	_, _ = fmt.Fprintln(output, "More information + source code:")
	_, _ = fmt.Fprintln(output, "  <https://github.com/walles/moar#readme>")
	_, _ = fmt.Fprintln(output)
	_, _ = fmt.Fprintln(output, "Environment:")
	if len(moarEnv) == 0 {
		_, _ = fmt.Fprintln(output, "  Additional options are read from the MOAR environment variable if set.")
		_, _ = fmt.Fprintln(output, "  But currently, the MOAR environment variable is not set.")
	} else {
		_, _ = fmt.Fprintln(output, "  Additional options are read from the MOAR environment variable.")
		_, _ = fmt.Fprintf(output, "  Current setting: MOAR=\"%s\"\n", moarEnv)
	}

	absMoarPath, err := absLookPath(os.Args[0])
	if err == nil {
		absPagerValue, err := absLookPath(os.Getenv("PAGER"))
		if err != nil {
			absPagerValue = ""
		}
		if absPagerValue != absMoarPath {
			// We're not the default pager
			_, _ = fmt.Fprintln(output)
			_, _ = fmt.Fprintln(output, "Making moar your default pager:")
			_, _ = fmt.Fprintln(output, "  Put the following line in your ~/.bashrc, ~/.bash_profile or ~/.zshrc")
			_, _ = fmt.Fprintln(output, "  and moar will be used as the default pager in all new terminal windows:")
			_, _ = fmt.Fprintln(output)
			_, _ = fmt.Fprintf(output, "     export PAGER=%s\n", getMoarPath())
		}
	} else {
		log.Warn("Unable to find moar binary ", err)
	}

	_, _ = fmt.Fprintln(output)
	_, _ = fmt.Fprintln(output, "Options:")

	flagSet.PrintDefaults()
}

// "moar" if we're in the $PATH, otherwise an absolute path
func getMoarPath() string {
	moarPath := os.Args[0]
	if filepath.IsAbs(moarPath) {
		return moarPath
	}

	if strings.Contains(moarPath, string(os.PathSeparator)) {
		// Relative path
		moarPath, err := filepath.Abs(moarPath)
		if err != nil {
			panic(err)
		}
		return moarPath
	}

	// Neither absolute nor relative, try PATH
	_, err := exec.LookPath(moarPath)
	if err != nil {
		panic("Unable to find in $PATH: " + moarPath)
	}
	return moarPath
}

func absLookPath(path string) (string, error) {
	lookedPath, err := exec.LookPath(path)
	if err != nil {
		return "", err
	}

	absLookedPath, err := filepath.Abs(lookedPath)
	if err != nil {
		return "", err
	}

	return absLookedPath, err
}

// printProblemsHeader prints bug reporting information to stderr
func printProblemsHeader() {
	fmt.Fprintln(os.Stderr, "Please post the following report at <https://github.com/walles/moar/issues>,")
	fmt.Fprintln(os.Stderr, "or e-mail it to johan.walles@gmail.com.")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Version:", versionString)
	fmt.Fprintln(os.Stderr, "LANG   :", os.Getenv("LANG"))
	fmt.Fprintln(os.Stderr, "TERM   :", os.Getenv("TERM"))
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "GOOS    :", runtime.GOOS)
	fmt.Fprintln(os.Stderr, "GOARCH  :", runtime.GOARCH)
	fmt.Fprintln(os.Stderr, "Compiler:", runtime.Compiler)
	fmt.Fprintln(os.Stderr, "NumCPU  :", runtime.NumCPU())

	fmt.Fprintln(os.Stderr)
}

func parseStyleOption(styleOption string, flagSet *flag.FlagSet) chroma.Style {
	style, ok := styles.Registry[styleOption]
	if !ok {
		fmt.Fprintf(os.Stderr,
			"ERROR: Unrecognized style \"%s\", pick a style from here: https://xyproto.github.io/splash/docs/longer/all.html\n",
			styleOption)
		fmt.Fprintln(os.Stderr)
		printUsage(os.Stderr, flagSet, true)

		os.Exit(1)
	}

	return *style
}

func parseColorsOption(colorsOption string, flagSet *flag.FlagSet) chroma.Formatter {
	if strings.ToLower(colorsOption) == "auto" {
		colorsOption = "16M"
		if strings.Contains(os.Getenv("TERM"), "256") {
			// Covers "xterm-256color" as used by the macOS Terminal
			colorsOption = "256"
		}
	}

	switch strings.ToUpper(colorsOption) {
	case "8":
		return formatters.TTY8
	case "16":
		return formatters.TTY16
	case "256":
		return formatters.TTY256
	case "16M":
		return formatters.TTY16m
	}

	fmt.Fprintf(os.Stderr, "ERROR: Invalid color count \"%s\", valid counts are 8, 16, 256 or 16M.\n", colorsOption)
	fmt.Fprintln(os.Stderr)
	printUsage(os.Stderr, flagSet, true)

	os.Exit(1)
	panic("We just did os.Exit(), why are we still executing?")
}

func parseStatusBarStyle(styleOption string, flagSet *flag.FlagSet) m.StatusBarStyle {
	if styleOption == "inverse" {
		return m.STATUSBAR_STYLE_INVERSE
	}
	if styleOption == "plain" {
		return m.STATUSBAR_STYLE_PLAIN
	}
	if styleOption == "bold" {
		return m.STATUSBAR_STYLE_BOLD
	}

	fmt.Fprintf(os.Stderr,
		"ERROR: Unrecognized status bar style \"%s\", good ones are inverse, plain and bold.\n",
		styleOption)
	fmt.Fprintln(os.Stderr)
	printUsage(os.Stderr, flagSet, true)

	os.Exit(1)
	panic("os.Exit(1) just failed")
}

func parseUnprintableStyle(styleOption string, flagSet *flag.FlagSet) m.UnprintableStyle {
	if styleOption == "highlight" {
		return m.UNPRINTABLE_STYLE_HIGHLIGHT
	}
	if styleOption == "whitespace" {
		return m.UNPRINTABLE_STYLE_WHITESPACE
	}

	fmt.Fprintf(os.Stderr,
		"ERROR: Unrecognized invalid UTF8 rendering style \"%s\", good ones are highlight or whitespace.\n",
		styleOption)
	fmt.Fprintln(os.Stderr)
	printUsage(os.Stderr, flagSet, true)

	os.Exit(1)
	panic("os.Exit(1) just failed")
}

func main() {
	// FIXME: If we get a CTRL-C, get terminal back into a useful state before terminating

	defer func() {
		err := recover()
		if err == nil {
			return
		}

		printProblemsHeader()
		panic(err)
	}()

	flagSet := flag.NewFlagSet("", flag.ExitOnError)
	flagSet.Usage = func() {
		printUsage(os.Stdout, flagSet, false)
	}
	printVersion := flagSet.Bool("version", false, "Prints the moar version number")
	debug := flagSet.Bool("debug", false, "Print debug logs after exiting")
	trace := flagSet.Bool("trace", false, "Print trace logs after exiting")
	wrap := flagSet.Bool("wrap", false, "Wrap long lines")
	styleOption := flagSet.String("style", "native",
		"Highlighting style from https://xyproto.github.io/splash/docs/longer/all.html")
	colorsOption := flagSet.String("colors", "auto", "Highlighting palette size: 8, 16, 256, 16M, auto")
	noLineNumbers := flagSet.Bool("no-linenumbers", false, "Hide line numbers on startup, press left arrow key to show")
	noClearOnExit := flagSet.Bool("no-clear-on-exit", false, "Retain screen contents when exiting moar")
	statusBarStyleOption := flagSet.String("statusbar", "inverse", "Status bar style: inverse, plain or bold")
	UnprintableStyleOption := flagSet.String("render-unprintable", "highlight", "How unprintable characters are rendered: highlight or whitespace")

	// Combine flags from environment and from command line
	flags := os.Args[1:]
	moarEnv := strings.Trim(os.Getenv("MOAR"), " ")
	if len(moarEnv) > 0 {
		// FIXME: It would be nice if we could debug log that we're doing this,
		// but logging is not yet set up and depends on command line parameters.
		flags = append(strings.Split(moarEnv, " "), flags...)
	}

	err := flagSet.Parse(flags)
	if err != nil {
		printProblemsHeader()
		fmt.Fprintln(os.Stderr, "ERROR: Command line parsing failed:", err.Error())
		fmt.Fprintln(os.Stderr)
		printUsage(os.Stderr, flagSet, true)

		os.Exit(1)
	}

	if *printVersion {
		fmt.Println(versionString)
		os.Exit(0)
	}

	style := parseStyleOption(*styleOption, flagSet)
	formatter := parseColorsOption(*colorsOption, flagSet)
	statusBarStyle := parseStatusBarStyle(*statusBarStyleOption, flagSet)
	unprintableStyle := parseUnprintableStyle(*UnprintableStyleOption, flagSet)

	log.SetLevel(log.InfoLevel)
	if *trace {
		log.SetLevel(log.TraceLevel)
	} else if *debug {
		log.SetLevel(log.DebugLevel)
	}

	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: time.RFC3339Nano,
	})

	if len(flagSet.Args()) > 1 {
		fmt.Fprintln(os.Stderr, "ERROR: Expected exactly one filename, or data piped from stdin, got:", flagSet.Args())
		fmt.Fprintln(os.Stderr)
		printUsage(os.Stderr, flagSet, true)

		os.Exit(1)
	}

	stdinIsRedirected := !term.IsTerminal(int(os.Stdin.Fd()))
	stdoutIsRedirected := !term.IsTerminal(int(os.Stdout.Fd()))
	var inputFilename *string
	if len(flagSet.Args()) == 1 {
		word := flagSet.Arg(0)
		inputFilename = &word
	}

	if inputFilename == nil && !stdinIsRedirected {
		fmt.Fprintln(os.Stderr, "ERROR: Filename or input pipe required")
		fmt.Fprintln(os.Stderr)
		printUsage(os.Stderr, flagSet, true)
		os.Exit(1)
	}

	if inputFilename != nil && stdoutIsRedirected {
		// Pump file to stdout.
		//
		// If we get both redirected stdin and an input filename, we must prefer
		// to copy the file, because that's how less works.
		inputFile, err := os.Open(*inputFilename)
		if err != nil {
			fmt.Fprintln(os.Stderr, "ERROR: Failed to open", inputFile, ": ")
			os.Exit(1)
		}

		_, err = io.Copy(os.Stdout, inputFile)
		if err != nil {
			log.Fatal("Failed to copy ", inputFilename, " to stdout: ", err)
		}
		os.Exit(0)
	}

	if stdinIsRedirected && stdoutIsRedirected {
		// Must be done after trying to pump the input filename to stdout to be
		// compatible with less, see above.
		_, err := io.Copy(os.Stdout, os.Stdin)
		if err != nil {
			log.Fatal("Failed to copy stdin to stdout: ", err)
		}
		os.Exit(0)
	}

	// INVARIANT: At this point, stdoutIsRedirected is false and we should
	// proceed with paging.

	if stdinIsRedirected {
		// Display input pipe contents
		reader := m.NewReaderFromStream("", os.Stdin)
		startPaging(reader, *wrap, *noLineNumbers, *noClearOnExit, statusBarStyle, unprintableStyle)
		return
	}

	// Display the input file contents
	reader, err := m.NewReaderFromFilename(*inputFilename, style, formatter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
	startPaging(reader, *wrap, *noLineNumbers, *noClearOnExit, statusBarStyle, unprintableStyle)
}

func startPaging(reader *m.Reader,
	wrapLongLines, noLineNumbers, noClearOnExit bool,
	statusBarStyle m.StatusBarStyle,
	unprintableStyle m.UnprintableStyle,
) {
	screen, e := twin.NewScreen()
	if e != nil {
		panic(e)
	}

	var loglines strings.Builder
	log.SetOutput(&loglines)
	pager := m.NewPager(reader)
	pager.WrapLongLines = wrapLongLines
	pager.ShowLineNumbers = !noLineNumbers
	pager.StatusBarStyle = statusBarStyle
	pager.UnprintableStyle = unprintableStyle

	defer func() {
		// Restore screen...
		screen.Close()

		// ... before printing panic() output, otherwise the output will have
		// broken linefeeds and be hard to follow.
		if err := recover(); err != nil {
			panic(err)
		}

		if noClearOnExit {
			err := pager.ReprintAfterExit()
			if err != nil {
				log.Error("Failed reprinting pager view after exit", err)
			}
		}

		if len(loglines.String()) > 0 {
			printProblemsHeader()

			// FIXME: Don't print duplicate log messages more than once,
			// maybe invent our own logger for this?
			fmt.Fprintf(os.Stderr, "%s", loglines.String())
			os.Exit(1)
		}
	}()

	pager.StartPaging(screen)
}
