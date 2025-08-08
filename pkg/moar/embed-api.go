package moar

import (
	"io"

	"github.com/walles/moar/internal"
	internalReader "github.com/walles/moar/internal/reader"
	"github.com/walles/moar/twin"
)

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
	pagerReader := internalReader.NewFromText(options.Title, text)
	return pageFromReader(pagerReader, options)
}

func pageFromReader(reader *internalReader.ReaderImpl, options Options) error {
	pager := internal.NewPager(reader)
	pager.WrapLongLines = options.WrapLongLines

	screen, e := twin.NewScreen()
	if e != nil {
		// Screen setup failed
		return e
	}

	pager.StartPaging(screen, nil, nil)
	screen.Close()
	return nil
}
