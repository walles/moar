package cmd

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/styles"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/walles/moar/m"
	"github.com/walles/moar/twin"
)

var versionString = "Should be set when building, please use build.sh to build"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "moar <file>",
	Short: "A nice pager doing the right thing without any configuration",
	Long:  "Bug reporting, source code and more: https://github.com/walles/moar",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

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
	_, _ = fmt.Fprintln(output, "")
	_, _ = fmt.Fprintln(output, "Shows file contents. Compressed files will be transparently decompressed.")
	_, _ = fmt.Fprintln(output)
	_, _ = fmt.Fprintln(output, "Environment:")
	if len(moarEnv) == 0 {
		_, _ = fmt.Fprintln(output, "  Additional options are read from the MOAR environment variable if set.")
		_, _ = fmt.Fprintln(output, "  But currently, the MOAR environment variable is not set.")
	} else {
		_, _ = fmt.Fprintln(output, "  Additional options are read from the MOAR environment variable.")
		_, _ = fmt.Fprintf(output, "  Current setting: MOAR=\"%s\"\n", moarEnv)
	}
	_, _ = fmt.Fprintln(output)
	_, _ = fmt.Fprintln(output, "Options:")

	flagSet.PrintDefaults()

	moarPath, err := filepath.Abs(os.Args[0])
	if err == nil {
		pagerValue, err := filepath.Abs(os.Getenv("PAGER"))
		if err != nil {
			pagerValue = ""
		}
		if pagerValue != moarPath {
			// We're not the default pager
			_, _ = fmt.Fprintln(output)
			_, _ = fmt.Fprintln(output, "To make Moar your default pager, put the following line in")
			_, _ = fmt.Fprintln(output, "your .bashrc or .bash_profile and it will be default in all")
			_, _ = fmt.Fprintln(output, "new terminal windows:")
			_, _ = fmt.Fprintf(output, "   export PAGER=%s\n", moarPath)
		}
	} else {
		log.Warn("Unable to find moar binary ", err)
	}
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
	colorsOption := flagSet.String("colors", "16M", "Highlighting palette size: 8, 16, 256, 16M")
	noLineNumbers := flagSet.Bool("no-linenumbers", false, "Hide line numbers on startup, press left arrow key to show")

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
		fmt.Fprintln(os.Stderr, "")
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
		startPaging(reader, *wrap, *noLineNumbers)
		return
	}

	// Display the input file contents
	reader, err := m.NewReaderFromFilename(*inputFilename, style, formatter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
	startPaging(reader, *wrap, *noLineNumbers)
}

func startPaging(reader *m.Reader, wrapLongLines bool, noLineNumbers bool) {
	screen, e := twin.NewScreen()
	if e != nil {
		panic(e)
	}

	var loglines strings.Builder
	defer func() {
		// Restore screen...
		screen.Close()

		// ... before printing panic() output, otherwise the output will have
		// broken linefeeds and be hard to follow.
		if err := recover(); err != nil {
			panic(err)
		}

		if len(loglines.String()) > 0 {
			printProblemsHeader()

			// FIXME: Don't print duplicate log messages more than once,
			// maybe invent our own logger for this?
			fmt.Fprintf(os.Stderr, "%s", loglines.String())
			os.Exit(1)
		}
	}()

	log.SetOutput(&loglines)
	pager := m.NewPager(reader)
	pager.WrapLongLines = wrapLongLines
	pager.ShowLineNumbers = !noLineNumbers
	pager.StartPaging(screen)
}
