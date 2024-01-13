package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	log "github.com/sirupsen/logrus"
	"golang.org/x/term"

	"github.com/walles/moar/m"
	"github.com/walles/moar/m/linenumbers"
	"github.com/walles/moar/m/textstyles"
	"github.com/walles/moar/twin"
)

var versionString = "Should be set when building, please use build.sh to build"

func printUsageEnvVar(output io.Writer, envVarName string, description string) {
	value := os.Getenv(envVarName)
	if len(value) == 0 {
		return
	}

	_, _ = fmt.Fprintf(output, "  %s (%s): %s\n",
		envVarName,
		description,
		strings.ReplaceAll(value, "\x1b", "ESC"),
	)
}

func printCommandline(output io.Writer) {
	_, _ = fmt.Fprintln(output, "Commandline: moar", strings.Join(os.Args[1:], " "))
	_, _ = fmt.Fprintf(output, "Environment: MOAR=\"%v\"\n", os.Getenv("MOAR"))
	_, _ = fmt.Fprintln(output)
}

func printUsage(output io.Writer, flagSet *flag.FlagSet, withCommandline bool) {
	// This controls where PrintDefaults() prints, see below
	flagSet.SetOutput(output)

	// FIXME: Log if any printouts fail?

	if withCommandline {
		printCommandline(output)
	}

	_, _ = fmt.Fprintln(output, "Usage:")
	_, _ = fmt.Fprintln(output, "  moar [options] <file>")
	_, _ = fmt.Fprintln(output, "  ... | moar")
	_, _ = fmt.Fprintln(output, "  moar < file")
	_, _ = fmt.Fprintln(output)
	_, _ = fmt.Fprintln(output, "Shows file contents. Compressed files will be transparently decompressed.")
	_, _ = fmt.Fprintln(output, "Input is expected to be (possibly compressed) UTF-8 encoded text. Invalid /")
	_, _ = fmt.Fprintln(output, "non-printable characters are by default rendered as '?'.")
	_, _ = fmt.Fprintln(output)
	_, _ = fmt.Fprintln(output, "More information + source code:")
	_, _ = fmt.Fprintln(output, "  <https://github.com/walles/moar#readme>")
	_, _ = fmt.Fprintln(output)
	_, _ = fmt.Fprintln(output, "Environment:")

	moarEnv := os.Getenv("MOAR")
	if len(moarEnv) == 0 {
		_, _ = fmt.Fprintln(output, "  Additional options are read from the MOAR environment variable if set.")
		_, _ = fmt.Fprintln(output, "  But currently, the MOAR environment variable is not set.")
	} else {
		_, _ = fmt.Fprintln(output, "  Additional options are read from the MOAR environment variable.")
		_, _ = fmt.Fprintf(output, "  Current setting: MOAR=\"%s\"\n", moarEnv)
	}

	printUsageEnvVar(output, "LESS_TERMCAP_md", "man page bold style")
	printUsageEnvVar(output, "LESS_TERMCAP_us", "man page underline style")
	printUsageEnvVar(output, "LESS_TERMCAP_so", "search hits and footer style")

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

	_, _ = fmt.Fprintln(output, "  +1234")
	_, _ = fmt.Fprintln(output, "    \tImmediately scroll to line 1234")
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
}

func parseLexerOption(lexerOption string) (chroma.Lexer, error) {
	byMimeType := lexers.MatchMimeType(lexerOption)
	if byMimeType != nil {
		return byMimeType, nil
	}

	// Use Chroma's built-in fuzzy lexer picker
	lexer := lexers.Get(lexerOption)
	if lexer != nil {
		return lexer, nil
	}

	return nil, fmt.Errorf(
		"Look here for inspiration: https://github.com/alecthomas/chroma/tree/master/lexers/embedded",
	)
}

func parseStyleOption(styleOption string) (*chroma.Style, error) {
	style, ok := styles.Registry[styleOption]
	if !ok {
		return &chroma.Style{}, fmt.Errorf(
			"Pick a style from here: https://xyproto.github.io/splash/docs/longer/all.html")
	}

	return style, nil
}

func parseColorsOption(colorsOption string) (twin.ColorType, error) {
	if strings.ToLower(colorsOption) == "auto" {
		colorsOption = "16M"
		if strings.Contains(os.Getenv("TERM"), "256") {
			// Covers "xterm-256color" as used by the macOS Terminal
			colorsOption = "256"
		}
	}

	switch strings.ToUpper(colorsOption) {
	case "8":
		return twin.ColorType8, nil
	case "16":
		return twin.ColorType16, nil
	case "256":
		return twin.ColorType256, nil
	case "16M":
		return twin.ColorType24bit, nil
	}

	var noColor twin.ColorType
	return noColor, fmt.Errorf("Valid counts are 8, 16, 256, 16M or auto")
}

func parseStatusBarStyle(styleOption string) (m.StatusBarOption, error) {
	if styleOption == "inverse" {
		return m.STATUSBAR_STYLE_INVERSE, nil
	}
	if styleOption == "plain" {
		return m.STATUSBAR_STYLE_PLAIN, nil
	}
	if styleOption == "bold" {
		return m.STATUSBAR_STYLE_BOLD, nil
	}

	return 0, fmt.Errorf("Good ones are inverse, plain and bold")
}

func parseUnprintableStyle(styleOption string) (textstyles.UnprintableStyleT, error) {
	if styleOption == "highlight" {
		return textstyles.UnprintableStyleHighlight, nil
	}
	if styleOption == "whitespace" {
		return textstyles.UnprintableStyleWhitespace, nil
	}

	return 0, fmt.Errorf("Good ones are highlight or whitespace")
}

func parseScrollHint(scrollHint string) (twin.Cell, error) {
	scrollHint = strings.ReplaceAll(scrollHint, "ESC", "\x1b")
	hintAsLine := m.NewLine(scrollHint)
	parsedTokens := hintAsLine.HighlightedTokens("", nil, nil).Cells
	if len(parsedTokens) == 1 {
		return parsedTokens[0], nil
	}

	return twin.Cell{}, fmt.Errorf("Expected exactly one (optionally highlighted) character. For example: 'ESC[2m…'")
}

func parseShiftAmount(shiftAmount string) (uint, error) {
	value, err := strconv.ParseUint(shiftAmount, 10, 32)
	if err != nil {
		return 0, err
	}

	if value < 1 {
		return 0, fmt.Errorf("Shift amount must be at least 1")
	}

	// Let's add an upper bound as well if / when requested

	return uint(value), nil
}

func parseMouseMode(mouseMode string) (twin.MouseMode, error) {
	switch mouseMode {
	case "auto":
		return twin.MouseModeAuto, nil
	case "select", "mark":
		return twin.MouseModeSelect, nil
	case "scroll":
		return twin.MouseModeScroll, nil
	}

	return twin.MouseModeAuto, fmt.Errorf("Valid modes are auto, select and scroll")
}

func pumpToStdout(inputFilename *string) error {
	if inputFilename != nil {
		// If we get both redirected stdin and an input filename, we must prefer
		// to copy the file, because that's how less works. That's why we go for
		// the filename first.
		inputFile, err := m.ZOpen(*inputFilename)
		if err != nil {
			return fmt.Errorf("Failed to open %s: %w", *inputFilename, err)
		}

		_, err = io.Copy(os.Stdout, inputFile)
		if err != nil {
			return fmt.Errorf("Failed to copy %s to stdout: %w", *inputFilename, err)
		}
		return nil
	}

	// Must be done after trying to pump the input filename to stdout to be
	// compatible with less, see above.
	_, err := io.Copy(os.Stdout, os.Stdin)
	if err != nil {
		return fmt.Errorf("Failed to copy stdin to stdout: %w", err)
	}
	return nil
}

// Duplicate of m/reader.go:tryOpen
func tryOpen(filename string) error {
	// Try opening the file
	tryMe, err := os.Open(filename)
	if err != nil {
		return err
	}

	// Try reading a byte
	buffer := make([]byte, 1)
	_, err = tryMe.Read(buffer)

	if err != nil && err.Error() == "EOF" {
		// Empty file, this is fine
		err = nil
	}

	closeErr := tryMe.Close()
	if err == nil && closeErr != nil {
		// Everything worked up until Close(), report the Close() error
		return closeErr
	}

	return err
}

// Parses an argument like "+123" anywhere on the command line into a one-based
// line number, and returns the remaining args.
//
// Returns nil on no target line number specified.
func getTargetLineNumber(args []string) (*linenumbers.LineNumber, []string) {
	for i, arg := range args {
		if !strings.HasPrefix(arg, "+") {
			continue
		}

		lineNumber, err := strconv.ParseInt(arg[1:], 10, 32)
		if err != nil {
			// Let's pretend this is a file name
			continue
		}
		if lineNumber < 1 {
			// Pretend this is a file name
			continue
		}

		// Remove the target line number from the args
		//
		// Ref: https://stackoverflow.com/a/57213476/473672
		remainingArgs := make([]string, 0)
		remainingArgs = append(remainingArgs, args[:i]...)
		remainingArgs = append(remainingArgs, args[i+1:]...)

		returnMe := linenumbers.LineNumberFromOneBased(int(lineNumber))
		return &returnMe, remainingArgs
	}

	return nil, args
}

func main() {
	// FIXME: If we get a CTRL-C, get terminal back into a useful state before terminating

	var loglines strings.Builder
	log.SetOutput(&loglines)
	defer func() {
		err := recover()
		if len(loglines.String()) == 0 && err == nil {
			// No problems
			return
		}

		printProblemsHeader()

		if len(loglines.String()) > 0 {
			fmt.Fprintln(os.Stderr)
			// Consider not printing duplicate log messages more than once
			fmt.Fprintf(os.Stderr, "%s", loglines.String())
		}

		if err != nil {
			fmt.Fprintln(os.Stderr)
			panic(err)
		}

		os.Exit(1)
	}()

	flagSet := flag.NewFlagSet("",
		flag.ContinueOnError, // We want to do our own error handling
	)
	flagSet.SetOutput(io.Discard) // We want to do our own printing

	printVersion := flagSet.Bool("version", false, "Prints the moar version number")
	debug := flagSet.Bool("debug", false, "Print debug logs after exiting")
	trace := flagSet.Bool("trace", false, "Print trace logs after exiting")

	wrap := flagSet.Bool("wrap", false, "Wrap long lines")
	follow := flagSet.Bool("follow", false, "Follow piped input just like \"tail -f\"")
	styleOption := flagSetFunc(flagSet,
		"style", nil,
		"Highlighting style from https://xyproto.github.io/splash/docs/longer/all.html", parseStyleOption)
	lexer := flagSetFunc(flagSet,
		"lang", nil,
		"File contents, used for highlighting. Mime type or file extension (\"html\"). Default is to guess by filename.", parseLexerOption)

	defaultFormatter, err := parseColorsOption("auto")
	if err != nil {
		panic(fmt.Errorf("Failed parsing default formatter: %w", err))
	}
	terminalColorsCount := flagSetFunc(flagSet,
		"colors", defaultFormatter, "Highlighting palette size: 8, 16, 256, 16M, auto", parseColorsOption)

	noLineNumbers := flagSet.Bool("no-linenumbers", false, "Hide line numbers on startup, press left arrow key to show")
	noStatusBar := flagSet.Bool("no-statusbar", false, "Hide the status bar, toggle with '='")
	quitIfOneScreen := flagSet.Bool("quit-if-one-screen", false, "Don't page if contents fits on one screen")
	noClearOnExit := flagSet.Bool("no-clear-on-exit", false, "Retain screen contents when exiting moar")
	statusBarStyle := flagSetFunc(flagSet, "statusbar", m.STATUSBAR_STYLE_INVERSE,
		"Status bar style: inverse, plain or bold", parseStatusBarStyle)
	unprintableStyle := flagSetFunc(flagSet, "render-unprintable", textstyles.UnprintableStyleHighlight,
		"How unprintable characters are rendered: highlight or whitespace", parseUnprintableStyle)
	scrollLeftHint := flagSetFunc(flagSet, "scroll-left-hint",
		twin.NewCell('<', twin.StyleDefault.WithAttr(twin.AttrReverse)),
		"Shown when view can scroll left. One character with optional ANSI highlighting.", parseScrollHint)
	scrollRightHint := flagSetFunc(flagSet, "scroll-right-hint",
		twin.NewCell('>', twin.StyleDefault.WithAttr(twin.AttrReverse)),
		"Shown when view can scroll right. One character with optional ANSI highlighting.", parseScrollHint)
	shift := flagSetFunc(flagSet, "shift", 16, "Horizontal scroll amount >=1, defaults to 16", parseShiftAmount)
	mouseMode := flagSetFunc(
		flagSet,
		"mousemode",
		twin.MouseModeAuto,
		"Mouse mode: auto, select or scroll: https://github.com/walles/moar/blob/master/MOUSE.md",
		parseMouseMode,
	)

	// Combine flags from environment and from command line
	flags := os.Args[1:]
	moarEnv := strings.Trim(os.Getenv("MOAR"), " ")
	if len(moarEnv) > 0 {
		// FIXME: It would be nice if we could debug log that we're doing this,
		// but logging is not yet set up and depends on command line parameters.
		flags = append(strings.Fields(moarEnv), flags...)
	}

	targetLineNumber, remainingArgs := getTargetLineNumber(flags)

	err = flagSet.Parse(remainingArgs)
	if err != nil {
		if err == flag.ErrHelp {
			printUsage(os.Stdout, flagSet, false)
			return
		}

		errorText := err.Error()
		if strings.HasPrefix(errorText, "invalid value") {
			errorText = strings.Replace(errorText, ": ", "\n\n", 1)
		}
		boldErrorMessage := "\x1b[1m" + errorText + "\x1b[m"
		fmt.Fprintln(os.Stderr, "ERROR:", boldErrorMessage)
		fmt.Fprintln(os.Stderr)

		printCommandline(os.Stderr)

		fmt.Fprintln(os.Stderr, "For help, run: \x1b[1mmoar --help\x1b[m")

		os.Exit(1)
	}

	if *printVersion {
		fmt.Println(versionString)
		os.Exit(0)
	}

	log.SetLevel(log.InfoLevel)
	if *trace {
		log.SetLevel(log.TraceLevel)
	} else if *debug {
		log.SetLevel(log.DebugLevel)
	}

	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: time.StampMicro,
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
		word := flagSet.Args()[0]
		inputFilename = &word

		// Need to check before twin.NewScreen() below, otherwise the screen
		// will be cleared before we print the "No such file" error.
		err := tryOpen(*inputFilename)
		if err != nil {
			fmt.Fprintln(os.Stderr, "ERROR:", err)
			os.Exit(1)
		}
	}

	if inputFilename == nil && !stdinIsRedirected {
		fmt.Fprintln(os.Stderr, "ERROR: Filename or input pipe required")
		fmt.Fprintln(os.Stderr)
		printUsage(os.Stderr, flagSet, true)
		os.Exit(1)
	}

	if stdoutIsRedirected {
		err := pumpToStdout(inputFilename)
		if err != nil {
			fmt.Fprintln(os.Stderr, "ERROR:", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// INVARIANT: At this point, stdout is a terminal and we should proceed with
	// paging.
	stdoutIsTerminal := !stdoutIsRedirected
	if !stdoutIsTerminal {
		panic("Invariant broken: stdout is not a terminal")
	}

	screen, err := twin.NewScreenWithMouseModeAndColorType(*mouseMode, *terminalColorsCount)
	if err != nil {
		// Ref: https://github.com/walles/moar/issues/149
		log.Debug("Failed to set up screen for paging, pumping to stdout instead: ", err)
		err := pumpToStdout(inputFilename)
		if err != nil {
			fmt.Fprintln(os.Stderr, "ERROR:", err)
			os.Exit(1)
		}
		return
	}

	formatter := formatters.TTY256
	if *terminalColorsCount == twin.ColorType8 {
		formatter = formatters.TTY8
	} else if *terminalColorsCount == twin.ColorType16 {
		formatter = formatters.TTY16
	} else if *terminalColorsCount == twin.ColorType24bit {
		formatter = formatters.TTY16m
	}

	style := decodeStyleOption(styleOption)

	var reader *m.Reader
	if stdinIsRedirected {
		// Display input pipe contents
		reader = m.NewReaderFromStream("", os.Stdin, *style, formatter, *lexer)
	} else {
		// Display the input file contents
		reader, err = m.NewReaderFromFilename(*inputFilename, *style, formatter, *lexer)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			os.Exit(1)
		}
	}

	pager := m.NewPager(reader)
	pager.WrapLongLines = *wrap
	pager.ShowLineNumbers = !*noLineNumbers
	pager.ShowStatusBar = !*noStatusBar
	pager.DeInit = !*noClearOnExit
	pager.QuitIfOneScreen = *quitIfOneScreen
	pager.StatusBarStyle = *statusBarStyle
	pager.UnprintableStyle = *unprintableStyle
	pager.ScrollLeftHint = *scrollLeftHint
	pager.ScrollRightHint = *scrollRightHint
	pager.SideScrollAmount = int(*shift)

	pager.TargetLineNumber = targetLineNumber
	if *follow && pager.TargetLineNumber == nil {
		reallyHigh := linenumbers.LineNumberMax()
		pager.TargetLineNumber = &reallyHigh
	}

	startPaging(pager, screen, style, &formatter)
}

func decodeStyleOption(styleOption **chroma.Style) *chroma.Style {
	if *styleOption != nil {
		return *styleOption
	}

	return styles.Get("native")
}

// Define a generic flag with specified name, default value, and usage string.
// The return value is the address of a variable that stores the parsed value of
// the flag.
func flagSetFunc[T any](flagSet *flag.FlagSet, name string, defaultValue T, usage string, parser func(valueString string) (T, error)) *T {
	parsed := defaultValue

	flagSet.Func(name, usage, func(valueString string) error {
		parseResult, err := parser(valueString)
		if err != nil {
			return err
		}
		parsed = parseResult
		return nil
	})

	return &parsed
}

func startPaging(pager *m.Pager, screen twin.Screen, chromaStyle *chroma.Style, chromaFormatter *chroma.Formatter) {
	defer func() {
		// Restore screen...
		screen.Close()

		// ... before printing any panic() output, otherwise the output will
		// have broken linefeeds and be hard to follow.
		if err := recover(); err != nil {
			panic(err)
		}

		if !pager.DeInit {
			err := pager.ReprintAfterExit()
			if err != nil {
				log.Error("Failed reprinting pager view after exit", err)
			}
		}
	}()

	pager.StartPaging(screen, chromaStyle, chromaFormatter)
}
