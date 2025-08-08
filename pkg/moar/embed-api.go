package moar

import (
	"fmt"
	"io"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/walles/moar/internal"
	internalReader "github.com/walles/moar/internal/reader"
	"github.com/walles/moar/twin"
)

const logLevel = log.WarnLevel

// If you feel some option is missing, request more options at
// https://github.com/walles/moar/issues.
type Options struct {
	// Name displayed in the bottom left corner of the pager.
	//
	// Defaults to the file name when paging files, otherwise nothing. Leave
	// blank for default.
	Title string

	// The default is to auto format JSON input. Set this to true to disable
	// auto formatting.
	NoAutoFormat bool

	// Long lines are truncated by default. Set this to true to wrap them.
	// Users can toggle wrapping on / off using the 'w' key while paging.
	WrapLongLines bool
}

func PageFromStream(reader io.Reader, options Options) error {
	logs := startLogCollection()
	defer collectLogs(logs)

	pagerReader, err := internalReader.NewFromStream(
		options.Title,
		reader,
		nil,
		internalReader.ReaderOptions{
			ShouldFormat: !options.NoAutoFormat,
		})
	if err != nil {
		return err
	}

	return pageFromReader(pagerReader, options)
}

func PageFromFile(name string, options Options) error {
	logs := startLogCollection()
	defer collectLogs(logs)

	pagerReader, err := internalReader.NewFromFilename(
		name,
		nil,
		internalReader.ReaderOptions{
			ShouldFormat: !options.NoAutoFormat,
		})
	if err != nil {
		return err
	}

	if options.Title != "" {
		pagerReader.Name = &options.Title
	}

	return pageFromReader(pagerReader, options)
}

func PageFromString(text string, options Options) error {
	logs := startLogCollection()
	defer collectLogs(logs)

	pagerReader := internalReader.NewFromText(options.Title, text)
	return pageFromReader(pagerReader, options)
}

func startLogCollection() *internal.LogWriter {
	log.SetLevel(logLevel)

	var logLines internal.LogWriter
	log.SetOutput(&logLines)
	return &logLines
}

func collectLogs(logs *internal.LogWriter) {
	if len(logs.String()) == 0 {
		return
	}
	fmt.Fprintln(os.Stderr, logs.String())
}

func pageFromReader(reader *internalReader.ReaderImpl, options Options) error {
	pager := internal.NewPager(reader)
	pager.WrapLongLines = options.WrapLongLines

	screen, e := twin.NewScreen()
	if e != nil {
		// Screen setup failed
		return e
	}

	style := internal.GetStyleForScreen(screen)
	reader.SetStyleForHighlighting(style)

	pager.StartPaging(screen, nil, nil)
	screen.Close()
	return nil
}
