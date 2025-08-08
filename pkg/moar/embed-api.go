package moar

import "io"

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
	panic("not implemented")
}

func PageFromFile(name string, options Options) error {
	panic("not implemented")
}

func PageFromString(text string, options Options) error {
	panic("not implemented")
}
